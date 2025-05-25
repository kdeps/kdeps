package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kartographer/graph"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

type DependencyResolver struct {
	Fs                   afero.Fs
	Logger               *logging.Logger
	Resources            []ResourceNodeEntry
	ResourceDependencies map[string][]string
	DependencyGraph      []string
	VisitedPaths         map[string]bool
	Context              context.Context //nolint:containedctx // TODO: move this context into function params
	Graph                *graph.DependencyGraph
	Environment          *environment.Environment
	Workflow             pklWf.Workflow
	Request              *gin.Context
	MemoryReader         *memory.PklResourceReader
	MemoryDBPath         string
	SessionReader        *session.PklResourceReader
	SessionDBPath        string
	ToolReader           *tool.PklResourceReader
	ToolDBPath           string
	ItemReader           *item.PklResourceReader
	ItemDBPath           string
	AgentName            string
	RequestID            string
	RequestPklFile       string
	ResponsePklFile      string
	ResponseTargetFile   string
	ProjectDir           string
	WorkflowDir          string
	AgentDir             string
	ActionDir            string
	FilesDir             string
	DataDir              string
	APIServerMode        bool
	AnacondaInstalled    bool
	FileRunCounter       map[string]int // Added to track run count per file
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, req *gin.Context, logger *logging.Logger) (*DependencyResolver, error) {
	var agentDir, graphID, actionDir string

	contextKeys := map[*string]ktx.ContextKey{
		&agentDir:  ktx.CtxKeyAgentDir,
		&graphID:   ktx.CtxKeyGraphID,
		&actionDir: ktx.CtxKeyActionDir,
	}

	for ptr, key := range contextKeys {
		if value, found := ktx.ReadContext(ctx, key); found {
			if strValue, ok := value.(string); ok {
				*ptr = strValue
			}
		}
	}

	workflowDir := filepath.Join(agentDir, "/workflow/")
	projectDir := filepath.Join(agentDir, "/project/")
	pklWfFile := filepath.Join(workflowDir, "workflow.pkl")

	exists, err := afero.Exists(fs, pklWfFile)
	if err != nil || !exists {
		return nil, fmt.Errorf("error checking %s: %w", pklWfFile, err)
	}

	dataDir := filepath.Join(projectDir, "/data/")
	filesDir := filepath.Join(actionDir, "/files/")

	directories := []string{
		projectDir,
		actionDir,
		filesDir,
	}

	// Create directories
	if err := utils.CreateDirectories(fs, ctx, directories); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	// List of files to create (stamp file)
	files := []string{
		filepath.Join(actionDir, graphID),
	}

	if err := utils.CreateFiles(fs, ctx, files); err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	requestPklFile := filepath.Join(actionDir, "/api/"+graphID+"__request.pkl")
	responsePklFile := filepath.Join(actionDir, "/api/"+graphID+"__response.pkl")
	responseTargetFile := filepath.Join(actionDir, "/api/"+graphID+"__response.json")

	workflowConfiguration, err := pklWf.LoadFromPath(ctx, pklWfFile)
	if err != nil {
		return nil, err
	}

	var apiServerMode, installAnaconda bool
	var agentName, memoryDBPath, sessionDBPath, toolDBPath, itemDBPath string

	if workflowConfiguration.GetSettings() != nil {
		apiServerMode = workflowConfiguration.GetSettings().APIServerMode
		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		installAnaconda = agentSettings.InstallAnaconda
		agentName = workflowConfiguration.GetName()
	}

	memoryDBPath = filepath.Join("/.kdeps/", agentName+"_memory.db")
	memoryReader, err := memory.InitializeMemory(memoryDBPath)
	if err != nil {
		memoryReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize DB memory: %w", err)
	}

	sessionDBPath = filepath.Join(actionDir, graphID+"_session.db")
	sessionReader, err := session.InitializeSession(sessionDBPath)
	if err != nil {
		sessionReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize session DB: %w", err)
	}

	toolDBPath = filepath.Join(actionDir, graphID+"_tool.db")
	toolReader, err := tool.InitializeTool(toolDBPath)
	if err != nil {
		toolReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize tool DB: %w", err)
	}

	itemDBPath = filepath.Join(actionDir, graphID+"_item.db")
	itemReader, err := item.InitializeItem(itemDBPath, nil)
	if err != nil {
		itemReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize item DB: %w", err)
	}

	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		Context:              ctx,
		Environment:          env,
		WorkflowDir:          workflowDir,
		AgentDir:             agentDir,
		ActionDir:            actionDir,
		FilesDir:             filesDir,
		DataDir:              dataDir,
		RequestID:            graphID,
		RequestPklFile:       requestPklFile,
		ResponsePklFile:      responsePklFile,
		ResponseTargetFile:   responseTargetFile,
		ProjectDir:           projectDir,
		Request:              req,
		Workflow:             workflowConfiguration,
		APIServerMode:        apiServerMode,
		AnacondaInstalled:    installAnaconda,
		AgentName:            agentName,
		MemoryDBPath:         memoryDBPath,
		MemoryReader:         memoryReader,
		SessionDBPath:        sessionDBPath,
		SessionReader:        sessionReader,
		ToolDBPath:           toolDBPath,
		ToolReader:           toolReader,
		ItemDBPath:           itemDBPath,
		ItemReader:           itemReader,
		FileRunCounter:       make(map[string]int), // Initialize the file run counter map
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger.BaseLogger(), dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, errors.New("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}

// ClearItemDB clears all contents of the item database.
func (dr *DependencyResolver) ClearItemDB() error {
	// Clear all records in the items table
	_, err := dr.ItemReader.DB.Exec("DELETE FROM items")
	if err != nil {
		return fmt.Errorf("failed to clear item database: %w", err)
	}
	dr.Logger.Info("cleared item database", "path", dr.ItemDBPath)
	return nil
}

// processResourceStep consolidates the pattern of: get timestamp, run a handler, adjust timeout (if provided),
// then wait for the timestamp change.
func (dr *DependencyResolver) processResourceStep(resourceID, step string, timeoutPtr *pkl.Duration, handler func() error) error {
	timestamp, err := dr.GetCurrentTimestamp(resourceID, step)
	if err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	timeout := 60 * time.Second
	if timeoutPtr != nil {
		timeout = timeoutPtr.GoDuration()
		dr.Logger.Infof("Timeout duration for '%s' is set to '%.0f' seconds", resourceID, timeout.Seconds())
	}

	if err := handler(); err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	if err := dr.WaitForTimestampChange(resourceID, timestamp, timeout, step); err != nil {
		return fmt.Errorf("%s timeout awaiting for output: %w", step, err)
	}
	return nil
}

// validateRequestParams checks if params in request.params("header_id") are in AllowedParams.
func (dr *DependencyResolver) validateRequestParams(file string, allowedParams []string) error {
	if len(allowedParams) == 0 {
		return nil // Allow all if empty
	}

	re := regexp.MustCompile(`request\.params\("([^"]+)"\)`)
	matches := re.FindAllStringSubmatch(file, -1)

	for _, match := range matches {
		param := match[1]
		if !utils.ContainsStringInsensitive(allowedParams, param) {
			return fmt.Errorf("param %s not in the allowed params", param)
		}
	}
	return nil
}

// validateRequestHeaders checks if headers in request.header("header_id") are in AllowedHeaders.
func (dr *DependencyResolver) validateRequestHeaders(file string, allowedHeaders []string) error {
	if len(allowedHeaders) == 0 {
		return nil // Allow all if empty
	}

	re := regexp.MustCompile(`request\.header\("([^"]+)"\)`)
	matches := re.FindAllStringSubmatch(file, -1)

	for _, match := range matches {
		header := match[1]
		if !utils.ContainsStringInsensitive(allowedHeaders, header) {
			return fmt.Errorf("header %s not in the allowed headers", header)
		}
	}
	return nil
}

// validateRequestPath checks if the actual request path is in AllowedRoutes.
func (dr *DependencyResolver) validateRequestPath(req *gin.Context, allowedRoutes []string) error {
	if len(allowedRoutes) == 0 {
		return nil // Allow all if empty
	}

	actualPath := req.Request.URL.Path
	if !utils.ContainsStringInsensitive(allowedRoutes, actualPath) {
		return fmt.Errorf("path %s not in the allowed routes", actualPath)
	}
	return nil
}

// validateRequestMethod checks if the actual request method is in AllowedHTTPMethods.
func (dr *DependencyResolver) validateRequestMethod(req *gin.Context, allowedMethods []string) error {
	if len(allowedMethods) == 0 {
		return nil // Allow all if empty
	}

	actualMethod := req.Request.Method
	if !utils.ContainsStringInsensitive(allowedMethods, actualMethod) {
		return fmt.Errorf("method %s not in the allowed HTTP methods", actualMethod)
	}
	return nil
}

// HandleRunAction is the main entry point to process resource run blocks.
func (dr *DependencyResolver) HandleRunAction() (bool, error) {
	// Recover from panics in this function.
	defer func() {
		if r := recover(); r != nil {
			dr.Logger.Error("panic recovered in HandleRunAction", "panic", r)

			// Close the DB
			dr.MemoryReader.DB.Close()
			dr.SessionReader.DB.Close()
			dr.ToolReader.DB.Close()
			dr.ItemReader.DB.Close()

			// Remove the session DB file
			if err := dr.Fs.RemoveAll(dr.SessionDBPath); err != nil {
				dr.Logger.Error("failed to delete the SessionDB file", "file", dr.SessionDBPath, "error", err)
			}

			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, false)
			dr.Logger.Error("stack trace", "stack", string(buf[:stackSize]))
		}
	}()

	requestFilePath := filepath.Join(dr.ActionDir, dr.RequestID)

	visited := make(map[string]bool)
	actionID := dr.Workflow.GetTargetActionID()
	dr.Logger.Debug("processing resources...")

	if err := dr.LoadResourceEntries(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error(), true)
	}

	// Build dependency stack for the target action
	stack := dr.Graph.BuildDependencyStack(actionID, visited)

	// Process each resource in the dependency stack
	for _, nodeActionID := range stack {
		for _, res := range dr.Resources {
			if res.ActionID != nodeActionID {
				continue
			}

			// Load the resource
			resPkl, err := dr.LoadResource(dr.Context, res.File, Resource)
			if err != nil {
				return dr.HandleAPIErrorResponse(500, err.Error(), true)
			}

			// Explicitly type rsc as *pklRes.Resource
			rsc, ok := resPkl.(*pklRes.Resource)
			if !ok {
				return dr.HandleAPIErrorResponse(500, "failed to cast resource to *pklRes.Resource for file "+res.File, true)
			}

			// Reinitialize item database with items, if any
			var items []string
			if rsc.Items != nil && len(*rsc.Items) > 0 {
				items = *rsc.Items
				// Close existing item database
				dr.ItemReader.DB.Close()
				// Reinitialize item database with items
				itemReader, err := item.InitializeItem(dr.ItemDBPath, items)
				if err != nil {
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to reinitialize item DB with items: %v", err), true)
				}
				dr.ItemReader = itemReader
				dr.Logger.Info("reinitialized item database with items", "actionID", nodeActionID, "itemCount", len(items))
			}

			// Process run block: once if no items, or once per item
			if len(items) == 0 {
				dr.Logger.Info("no items specified, processing run block once", "actionID", res.ActionID)
				proceed, err := dr.processRunBlock(res, rsc, nodeActionID, false)
				if err != nil {
					return false, err
				} else if !proceed {
					continue
				}
			} else {
				for _, itemValue := range items {
					dr.Logger.Info("processing item", "actionID", res.ActionID, "item", itemValue)
					// Set the current item in the database
					query := url.Values{"op": []string{"set"}, "value": []string{itemValue}}
					uri := url.URL{Scheme: "item", RawQuery: query.Encode()}
					if _, err := dr.ItemReader.Read(uri); err != nil {
						dr.Logger.Error("failed to set item", "item", itemValue, "error", err)
						return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to set item %s: %v", itemValue, err), true)
					}

					// reload the resource
					resPkl, err = dr.LoadResource(dr.Context, res.File, Resource)
					if err != nil {
						return dr.HandleAPIErrorResponse(500, err.Error(), true)
					}

					// Explicitly type rsc as *pklRes.Resource
					rsc, ok = resPkl.(*pklRes.Resource)
					if !ok {
						return dr.HandleAPIErrorResponse(500, "failed to cast resource to *pklRes.Resource for file "+res.File, true)
					}

					// Process runBlock for the current item
					_, err := dr.processRunBlock(res, rsc, nodeActionID, true)
					if err != nil {
						return false, err
					}
				}
				// Clear the item database after processing all items
				if err := dr.ClearItemDB(); err != nil {
					dr.Logger.Error("failed to clear item database after iteration", "actionID", res.ActionID, "error", err)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to clear item database for resource %s: %v", res.ActionID, err), true)
				}
			}

			// Process APIResponse once, outside the items loop
			if dr.APIServerMode && rsc.Run != nil && rsc.Run.APIResponse != nil {
				if err := dr.CreateResponsePklFile(*rsc.Run.APIResponse); err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}
			}
		}
	}

	// Close the DB
	dr.MemoryReader.DB.Close()
	dr.SessionReader.DB.Close()
	dr.ToolReader.DB.Close()
	dr.ItemReader.DB.Close()

	// Remove the request stamp file
	if err := dr.Fs.RemoveAll(requestFilePath); err != nil {
		dr.Logger.Error("failed to delete old requestID file", "file", requestFilePath, "error", err)
		return false, err
	}

	// Remove the session DB file
	if err := dr.Fs.RemoveAll(dr.SessionDBPath); err != nil {
		dr.Logger.Error("failed to delete the SessionDB file", "file", dr.SessionDBPath, "error", err)
		return false, err
	}

	// Log the final file run counts
	for file, count := range dr.FileRunCounter {
		dr.Logger.Info("file run count", "file", file, "count", count)
	}

	dr.Logger.Debug("all resources finished processing")
	return false, nil
}

// processRunBlock handles the runBlock processing for a resource, excluding APIResponse.
func (dr *DependencyResolver) processRunBlock(res ResourceNodeEntry, rsc *pklRes.Resource, actionID string, hasItems bool) (bool, error) {
	// Increment the run counter for this file
	dr.FileRunCounter[res.File]++
	dr.Logger.Info("processing run block for file", "file", res.File, "runCount", dr.FileRunCounter[res.File], "actionID", actionID)

	runBlock := rsc.Run
	if runBlock == nil {
		return false, nil
	}

	// When items are enabled, wait for the items database to have at least one item in the list
	if hasItems {
		const waitTimeout = 30 * time.Second
		const pollInterval = 500 * time.Millisecond
		deadline := time.Now().Add(waitTimeout)

		dr.Logger.Info("Waiting for items database to have a non-empty list", "actionID", actionID)
		for time.Now().Before(deadline) {
			// Query the items database to retrieve the list
			query := url.Values{"op": []string{"list"}}
			uri := url.URL{Scheme: "item", RawQuery: query.Encode()}
			result, err := dr.ItemReader.Read(uri)
			if err != nil {
				dr.Logger.Error("Failed to read list from items database", "actionID", actionID, "error", err)
				return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Failed to read list from items database for resource %s: %v", actionID, err), true)
			}
			// Parse the []byte result as a JSON array
			var items []string
			if len(result) > 0 {
				if err := json.Unmarshal(result, &items); err != nil {
					dr.Logger.Error("Failed to parse items database result as JSON array", "actionID", actionID, "error", err)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Failed to parse items database result for resource %s: %v", actionID, err), true)
				}
			}
			// Check if the list is non-empty
			if len(items) > 0 {
				dr.Logger.Info("Items database has a non-empty list", "actionID", actionID, "itemCount", len(items))
				break
			}
			dr.Logger.Debug("Items database list is empty, retrying", "actionID", actionID)
			time.Sleep(pollInterval)
		}

		// Check if we timed out
		if time.Now().After(deadline) {
			dr.Logger.Error("Timeout waiting for items database to have a non-empty list", "actionID", actionID)
			return dr.HandleAPIErrorResponse(500, "Timeout waiting for items database to have a non-empty list for resource "+actionID, true)
		}
	}

	if dr.APIServerMode {
		// Read the resource file content for validation
		fileContent, err := afero.ReadFile(dr.Fs, res.File)
		if err != nil {
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to read resource file %s: %v", res.File, err), true)
		}

		// Validate request.params
		allowedParams := []string{}
		if runBlock.AllowedParams != nil {
			allowedParams = *runBlock.AllowedParams
		}
		if err := dr.validateRequestParams(string(fileContent), allowedParams); err != nil {
			dr.Logger.Error("request params validation failed", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request params validation failed for resource %s: %v", res.ActionID, err), false)
		}

		// Validate request.header
		allowedHeaders := []string{}
		if runBlock.AllowedHeaders != nil {
			allowedHeaders = *runBlock.AllowedHeaders
		}
		if err := dr.validateRequestHeaders(string(fileContent), allowedHeaders); err != nil {
			dr.Logger.Error("request headers validation failed", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request headers validation failed for resource %s: %v", res.ActionID, err), false)
		}

		// Validate request.path
		allowedRoutes := []string{}
		if runBlock.RestrictToRoutes != nil {
			allowedRoutes = *runBlock.RestrictToRoutes
		}
		if err := dr.validateRequestPath(dr.Request, allowedRoutes); err != nil {
			dr.Logger.Info("skipping due to request path validation not allowed", "actionID", res.ActionID, "error", err)
			return false, nil
		}

		// Validate request.method
		allowedMethods := []string{}
		if runBlock.RestrictToHTTPMethods != nil {
			allowedMethods = *runBlock.RestrictToHTTPMethods
		}
		if err := dr.validateRequestMethod(dr.Request, allowedMethods); err != nil {
			dr.Logger.Info("skipping due to request method validation not allowed", "actionID", res.ActionID, "error", err)
			return false, nil
		}
	}

	// Skip condition
	if runBlock.SkipCondition != nil && utils.ShouldSkip(runBlock.SkipCondition) {
		dr.Logger.Infof("skip condition met, skipping: %s", res.ActionID)
		return false, nil
	}

	// Preflight check
	if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil &&
		!utils.AllConditionsMet(runBlock.PreflightCheck.Validations) {
		dr.Logger.Error("preflight check not met, failing:", res.ActionID)
		if runBlock.PreflightCheck.Error != nil {
			return dr.HandleAPIErrorResponse(
				runBlock.PreflightCheck.Error.Code,
				fmt.Sprintf("%s: %s", runBlock.PreflightCheck.Error.Message, res.ActionID), false)
		}
		return dr.HandleAPIErrorResponse(500, "Preflight check failed for resource: "+res.ActionID, false)
	}

	// Process Exec step, if defined
	if runBlock.Exec != nil && runBlock.Exec.Command != "" {
		if err := dr.processResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
			return dr.HandleExec(res.ActionID, runBlock.Exec)
		}); err != nil {
			dr.Logger.Error("exec error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), false)
		}
	}

	// Process Python step, if defined
	if runBlock.Python != nil && runBlock.Python.Script != "" {
		if err := dr.processResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
			return dr.HandlePython(res.ActionID, runBlock.Python)
		}); err != nil {
			dr.Logger.Error("python error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), false)
		}
	}

	// Process Chat (LLM) step, if defined
	if runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil) {
		dr.Logger.Info("Processing LLM chat step", "actionID", res.ActionID, "hasPrompt", runBlock.Chat.Prompt != nil, "hasScenario", runBlock.Chat.Scenario != nil)
		if runBlock.Chat.Scenario != nil {
			dr.Logger.Info("Scenario present", "length", len(*runBlock.Chat.Scenario))
		}
		if err := dr.processResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
			return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
		}); err != nil {
			dr.Logger.Error("LLM chat error", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
		}
	} else {
		dr.Logger.Info("Skipping LLM chat step", "actionID", res.ActionID, "chatNil",
			runBlock.Chat == nil, "modelEmpty", runBlock.Chat == nil || runBlock.Chat.Model == "",
			"promptAndScenarioNil", runBlock.Chat != nil && runBlock.Chat.Prompt == nil &&
				runBlock.Chat.Scenario == nil)
	}

	// Process HTTP Client step, if defined
	if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
		if err := dr.processResourceStep(res.ActionID, "client", runBlock.HTTPClient.TimeoutDuration, func() error {
			return dr.HandleHTTPClient(res.ActionID, runBlock.HTTPClient)
		}); err != nil {
			dr.Logger.Error("HTTP client error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ActionID, err), false)
		}
	}

	return true, nil
}
