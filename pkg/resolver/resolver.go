package resolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kartographer/graph"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/workflow"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklResource "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

type DependencyResolver struct {
	Fs                      afero.Fs
	Logger                  *logging.Logger
	Resources               []ResourceNodeEntry
	ResourceDependencies    map[string][]string
	DependencyGraph         []string
	VisitedPaths            map[string]bool
	Context                 context.Context // TODO: move this context into function params
	Graph                   *graph.DependencyGraph
	Environment             *environment.Environment
	Workflow                pklWf.Workflow
	Request                 *gin.Context
	MemoryReader            *memory.PklResourceReader
	MemoryDBPath            string
	SessionReader           *session.PklResourceReader
	SessionDBPath           string
	ToolReader              *tool.PklResourceReader
	ToolDBPath              string
	ItemReader              *item.PklResourceReader
	ItemDBPath              string
	DBs                     []*sql.DB // collection of DB connections used by the resolver
	AgentName               string
	RequestID               string
	RequestPklFile          string
	ResponsePklFile         string
	ResponseTargetFile      string
	ProjectDir              string
	WorkflowDir             string
	AgentDir                string
	ActionDir               string
	FilesDir                string
	DataDir                 string
	APIServerMode           bool
	AnacondaInstalled       bool
	FileRunCounter          map[string]int // Added to track run count per file
	DefaultTimeoutSec       int            // default timeout value in seconds
	CurrentResourceActionID string         // Track the currently processing resource actionID

	// Injectable helpers (overridable in tests)
	GetCurrentTimestampFn    func(string, string) (pkl.Duration, error)              `json:"-"`
	WaitForTimestampChangeFn func(string, pkl.Duration, time.Duration, string) error `json:"-"`

	// Additional injectable helpers for broader unit testing
	LoadResourceEntriesFn  func() error                                                               `json:"-"`
	LoadResourceFn         func(context.Context, string, ResourceType) (interface{}, error)           `json:"-"`
	BuildDependencyStackFn func(string, map[string]bool) []string                                     `json:"-"`
	ProcessRunBlockFn      func(ResourceNodeEntry, *pklResource.Resource, string, bool) (bool, error) `json:"-"`
	ClearItemDBFn          func() error                                                               `json:"-"`

	// Chat / HTTP injection helpers
	NewLLMFn               func(model string) (*ollama.LLM, error)                                                                                      `json:"-"`
	GenerateChatResponseFn func(context.Context, afero.Fs, *ollama.LLM, *pklLLM.ResourceChat, *tool.PklResourceReader, *logging.Logger) (string, error) `json:"-"`

	DoRequestFn func(*pklHTTP.ResourceHTTPClient) error `json:"-"`

	// Python / Conda execution injector
	ExecTaskRunnerFn func(context.Context, execute.ExecTask) (string, string, error) `json:"-"`

	// Import handling injectors
	PrependDynamicImportsFn func(string) error                              `json:"-"`
	AddPlaceholderImportsFn func(string) error                              `json:"-"`
	WalkFn                  func(afero.Fs, string, filepath.WalkFunc) error `json:"-"`
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

// getResourceReaders returns all configured custom resource readers.
func (dr *DependencyResolver) getResourceReaders() []pkl.ResourceReader {
	readers := make([]pkl.ResourceReader, 0, 4)
	if dr.MemoryReader != nil {
		readers = append(readers, dr.MemoryReader)
	}
	if dr.SessionReader != nil {
		readers = append(readers, dr.SessionReader)
	}
	if dr.ToolReader != nil {
		readers = append(readers, dr.ToolReader)
	}
	if dr.ItemReader != nil {
		readers = append(readers, dr.ItemReader)
	}
	return readers
}

// (removed) createEvaluator: use pkg/evaluator.NewConfiguredEvaluator instead

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

	// Use our patched workflow loading function
	workflowConfiguration, err := workflow.LoadWorkflow(ctx, pklWfFile, logger)
	if err != nil {
		return nil, fmt.Errorf("error reading workflow file '%s': %w", pklWfFile, err)
	}

	var apiServerMode, installAnaconda bool
	var agentName, memoryDBPath, sessionDBPath, toolDBPath, itemDBPath string

	// GetSettings() returns a struct, not a pointer, so we can always access it
	settings := workflowConfiguration.GetSettings()
	apiServerMode = settings.APIServerMode
	agentSettings := settings.AgentSettings
	installAnaconda = agentSettings.InstallAnaconda
	agentName = workflowConfiguration.GetAgentID()

	// Use configurable kdeps path; in Docker default to /agent/volume/, otherwise /.kdeps/
	kdepsBase := os.Getenv("KDEPS_VOLUME_PATH")
	if kdepsBase == "" {
		if env != nil && env.DockerMode == "1" {
			kdepsBase = "/agent/volume/"
		} else {
			kdepsBase = "/.kdeps/"
		}
	}
	// Ensure kdeps base directory exists before initializing SQLite
	if err := fs.MkdirAll(kdepsBase, 0o777); err != nil {
		return nil, fmt.Errorf("failed to create kdeps base directory %s: %w", kdepsBase, err)
	}
	memoryDBPath = filepath.Join(kdepsBase, agentName+"_memory.db")
	memoryReader, err := memory.InitializeMemory(memoryDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DB memory: %w", err)
	}

	sessionDBPath = ":memory:"
	sessionReader, err := session.InitializeSession(sessionDBPath)
	if err != nil {
		sessionReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize session DB: %w", err)
	}

	toolDBPath = ":memory:"
	toolReader, err := tool.InitializeTool(toolDBPath)
	if err != nil {
		toolReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize tool DB: %w", err)
	}

	itemDBPath = ":memory:"
	itemReader, err := item.InitializeItem(itemDBPath, nil)
	if err != nil {
		itemReader.DB.Close()
		return nil, fmt.Errorf("failed to initialize item DB: %w", err)
	}

	dependencyResolver := &DependencyResolver{
		Fs:                      fs,
		ResourceDependencies:    make(map[string][]string),
		Logger:                  logger,
		VisitedPaths:            make(map[string]bool),
		Context:                 ctx,
		Environment:             env,
		WorkflowDir:             workflowDir,
		AgentDir:                agentDir,
		ActionDir:               actionDir,
		FilesDir:                filesDir,
		DataDir:                 dataDir,
		RequestID:               graphID,
		RequestPklFile:          requestPklFile,
		ResponsePklFile:         responsePklFile,
		ResponseTargetFile:      responseTargetFile,
		ProjectDir:              projectDir,
		Request:                 req,
		Workflow:                workflowConfiguration,
		APIServerMode:           apiServerMode,
		AnacondaInstalled:       installAnaconda,
		AgentName:               agentName,
		MemoryDBPath:            memoryDBPath,
		MemoryReader:            memoryReader,
		SessionDBPath:           sessionDBPath,
		SessionReader:           sessionReader,
		ToolDBPath:              toolDBPath,
		ToolReader:              toolReader,
		ItemDBPath:              itemDBPath,
		ItemReader:              itemReader,
		CurrentResourceActionID: "", // Initialize as empty, will be set during resource processing
		DBs: []*sql.DB{
			memoryReader.DB,
			sessionReader.DB,
			toolReader.DB,
			itemReader.DB,
		},
		FileRunCounter: make(map[string]int), // Initialize the file run counter map
		DefaultTimeoutSec: func() int {
			if v, ok := os.LookupEnv("TIMEOUT"); ok {
				if i, err := strconv.Atoi(v); err == nil {
					return i // could be 0 (unlimited) or positive override
				}
			}
			return -1 // absent -> sentinel to allow PKL/default fallback
		}(),
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger.BaseLogger(), dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, errors.New("failed to initialize dependency graph")
	}

	// Default injectable helpers
	dependencyResolver.GetCurrentTimestampFn = dependencyResolver.GetCurrentTimestamp
	dependencyResolver.WaitForTimestampChangeFn = dependencyResolver.WaitForTimestampChange

	// Default injection for broader functions (now that Graph is initialized)
	dependencyResolver.LoadResourceEntriesFn = dependencyResolver.LoadResourceEntries
	dependencyResolver.LoadResourceFn = dependencyResolver.LoadResource
	dependencyResolver.BuildDependencyStackFn = dependencyResolver.Graph.BuildDependencyStack
	dependencyResolver.ProcessRunBlockFn = dependencyResolver.processRunBlock
	dependencyResolver.ClearItemDBFn = dependencyResolver.ClearItemDB

	// Chat helpers
	dependencyResolver.NewLLMFn = func(model string) (*ollama.LLM, error) {
		llm, err := ollama.New(ollama.WithModel(model))
		if err != nil {
			errMsg := strings.ToLower(err.Error())

			// Check for various Ollama error conditions that indicate we should try to pull the model
			shouldTryPull := strings.Contains(errMsg, "not found") ||
				strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found") ||
				strings.Contains(errMsg, "no such file or directory") ||
				strings.Contains(errMsg, "connection refused") ||
				strings.Contains(errMsg, "eof") ||
				strings.Contains(errMsg, "try pulling it first")

			if shouldTryPull {
				dependencyResolver.Logger.Info("model not available or server not running, attempting to pull", "model", model, "error", err.Error())

				// Try to pull the model (this will also ensure Ollama server is running)
				if pullErr := dependencyResolver.pullOllamaModel(dependencyResolver.Context, model); pullErr != nil {
					dependencyResolver.Logger.Error("failed to pull model", "model", model, "error", pullErr)
					return nil, fmt.Errorf("failed to pull model %s: %w", model, pullErr)
				}

				// Retry creating LLM after pulling
				llm, err = ollama.New(ollama.WithModel(model))
				if err != nil {
					// Try once more after a brief delay to allow server to fully start
					time.Sleep(1 * time.Second)
					llm, err = ollama.New(ollama.WithModel(model))
					if err != nil {
						return nil, fmt.Errorf("failed to create LLM after pulling model %s: %w", model, err)
					}
				}
			} else {
				return nil, fmt.Errorf("failed to create LLM: %w", err)
			}
		}
		return llm, nil
	}
	dependencyResolver.GenerateChatResponseFn = generateChatResponse
	dependencyResolver.DoRequestFn = dependencyResolver.DoRequest

	// Default Python/Conda runner
	dependencyResolver.ExecTaskRunnerFn = func(ctx context.Context, task execute.ExecTask) (string, string, error) {
		stdout, stderr, _, err := kdepsexec.RunExecTask(ctx, task, dependencyResolver.Logger, false)
		return stdout, stderr, err
	}

	// Import helpers
	dependencyResolver.PrependDynamicImportsFn = dependencyResolver.PrependDynamicImports
	dependencyResolver.AddPlaceholderImportsFn = dependencyResolver.AddPlaceholderImports
	dependencyResolver.WalkFn = afero.Walk

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
	timestamp, err := dr.GetCurrentTimestampFn(resourceID, step)
	if err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	var timeout time.Duration
	switch {
	case dr.DefaultTimeoutSec > 0: // positive value overrides everything
		timeout = time.Duration(dr.DefaultTimeoutSec) * time.Second
	case dr.DefaultTimeoutSec == 0: // 0 => unlimited
		timeout = 0
	case timeoutPtr != nil: // negative or unset – fall back to resource value
		timeout = timeoutPtr.GoDuration()
	default:
		timeout = 60 * time.Second
	}

	if err := handler(); err != nil {
		return fmt.Errorf("%s error: %w", step, err)
	}

	if err := dr.WaitForTimestampChangeFn(resourceID, timestamp, timeout, step); err != nil {
		return fmt.Errorf("%s timeout awaiting for output: %w", step, err)
	}
	return nil
}

// validateRequestParams checks if params in request.params("header_id") are in AllowedParams.
func (dr *DependencyResolver) validateRequestParams(file string, allowedParams []string) error {
	if len(allowedParams) == 0 {
		return nil // Allow all if empty
	}

	// Split file into lines to check each line individually
	lines := strings.Split(file, "\n")
	re := regexp.MustCompile(`request\.params\("([^"]+)"\)`)

	for _, line := range lines {
		// Skip commented lines (lines that start with // after any whitespace)
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Find matches in non-commented lines only
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			param := match[1]
			if !utils.ContainsStringInsensitive(allowedParams, param) {
				return fmt.Errorf("param %s not in the allowed params", param)
			}
		}
	}
	return nil
}

// validateRequestHeaders checks if headers in request.header("header_id") are in AllowedHeaders.
func (dr *DependencyResolver) validateRequestHeaders(file string, allowedHeaders []string) error {
	if len(allowedHeaders) == 0 {
		return nil // Allow all if empty
	}

	// Split file into lines to check each line individually
	lines := strings.Split(file, "\n")
	re := regexp.MustCompile(`request\.header\("([^"]+)"\)`)

	for _, line := range lines {
		// Skip commented lines (lines that start with // after any whitespace)
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Find matches in non-commented lines only
		matches := re.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			header := match[1]
			if !utils.ContainsStringInsensitive(allowedHeaders, header) {
				return fmt.Errorf("header %s not in the allowed headers", header)
			}
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
	targetActionID := dr.Workflow.GetTargetActionID()
	dr.Logger.Debug(messages.MsgProcessingResources)

	if err := dr.LoadResourceEntriesFn(); err != nil {
		return dr.HandleAPIErrorResponse(500, err.Error(), true)
	}

	// Build dependency stack for the target action
	stack := dr.BuildDependencyStackFn(targetActionID, visited)

	// Process each resource in the dependency stack
	for _, nodeActionID := range stack {
		for _, res := range dr.Resources {
			if res.ActionID != nodeActionID {
				continue
			}

			// Set the current resource actionID for error context
			dr.CurrentResourceActionID = res.ActionID

			// Load the resource with robust fallback
			resPkl, err := dr.loadResourceWithFallbackResolver(res.File)
			if err != nil {
				dr.Logger.Error("failed to load resource with fallback", "file", res.File, "error", err)
				return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to load resource %s: %v", res.File, err), true)
			}

			// Robustly cast to pklResource.Resource
			rsc, err := dr.castToResource(resPkl, res.File)
			if err != nil {
				dr.Logger.Error("failed to cast resource", "file", res.File, "error", err)
				return dr.HandleAPIErrorResponse(500, err.Error(), true)
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
				proceed, err := dr.ProcessRunBlockFn(res, rsc, nodeActionID, false)
				if err != nil {
					return false, err
				}
				// For resources with no items, we still want to process APIResponse even if no run actions were performed
				_ = proceed
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

					// reload the resource with robust fallback
					resPkl, err = dr.loadResourceWithFallbackResolver(res.File)
					if err != nil {
						dr.Logger.Error("failed to reload resource with fallback", "file", res.File, "error", err)
						return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to reload resource %s: %v", res.File, err), true)
					}

					// Robustly cast to pklResource.Resource
					rsc, err = dr.castToResource(resPkl, res.File)
					if err != nil {
						dr.Logger.Error("failed to cast reloaded resource", "file", res.File, "error", err)
						return dr.HandleAPIErrorResponse(500, err.Error(), true)
					}

					// Process runBlock for the current item
					_, err = dr.ProcessRunBlockFn(res, rsc, nodeActionID, true)
					if err != nil {
						return false, err
					}
				}
				// Clear the item database after processing all items
				if err := dr.ClearItemDBFn(); err != nil {
					dr.Logger.Error("failed to clear item database after iteration", "actionID", res.ActionID, "error", err)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to clear item database for resource %s: %v", res.ActionID, err), true)
				}
			}

			// Process APIResponse regardless of whether run block proceeded
			// This ensures response resources that only have apiResponse (no exec/python/chat/http actions) are still processed
			if dr.APIServerMode && rsc.Run.APIResponse != nil {
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

	// Remove the session DB file if it's not in-memory
	if dr.SessionDBPath != ":memory:" {
		if err := dr.Fs.RemoveAll(dr.SessionDBPath); err != nil {
			dr.Logger.Error("failed to delete the SessionDB file", "file", dr.SessionDBPath, "error", err)
			return false, err
		}
	}

	// Log the final file run counts
	for file, count := range dr.FileRunCounter {
		dr.Logger.Info("file run count", "file", file, "count", count)
	}

	dr.Logger.Debug(messages.MsgAllResourcesProcessed)
	return false, nil
}

// processRunBlock handles the runBlock processing for a resource, excluding APIResponse.
func (dr *DependencyResolver) processRunBlock(res ResourceNodeEntry, rsc *pklResource.Resource, actionID string, hasItems bool) (bool, error) {
	// Increment the run counter for this file
	dr.FileRunCounter[res.File]++
	dr.Logger.Info("processing run block for file", "file", res.File, "runCount", dr.FileRunCounter[res.File], "actionID", actionID)

	runBlock := rsc.Run
	// ResourceAction is a struct, not a pointer, so we can always access it

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
			dr.Logger.Debug(messages.MsgItemsDBEmptyRetry, "actionID", actionID)
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
			return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request params validation failed for resource %s: %v", res.ActionID, err), true)
		}

		// Validate request.header
		allowedHeaders := []string{}
		if runBlock.AllowedHeaders != nil {
			allowedHeaders = *runBlock.AllowedHeaders
		}
		if err := dr.validateRequestHeaders(string(fileContent), allowedHeaders); err != nil {
			dr.Logger.Error("request headers validation failed", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request headers validation failed for resource %s: %v", res.ActionID, err), true)
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
	if runBlock.PreflightCheck != nil && runBlock.PreflightCheck.Validations != nil {
		conditionsMet, failedConditions := utils.AllConditionsMetWithDetails(runBlock.PreflightCheck.Validations)
		if !conditionsMet {
			dr.Logger.Error("preflight check not met, collecting error and continuing to gather all errors:", res.ActionID, "failedConditions", failedConditions)

			// Build user-friendly error message
			var errorMessage string
			if runBlock.PreflightCheck.Error != nil && runBlock.PreflightCheck.Error.Message != "" {
				// Use the custom error message if provided
				errorMessage = runBlock.PreflightCheck.Error.Message
			} else {
				// Default error message
				errorMessage = fmt.Sprintf("Validation failed for %s", res.ActionID)
			}

			// Add specific validation failure details for debugging
			if len(failedConditions) > 0 {
				if len(failedConditions) == 1 {
					errorMessage += fmt.Sprintf(" (%s)", failedConditions[0])
				} else {
					errorMessage += fmt.Sprintf(" (%s)", strings.Join(failedConditions, ", "))
				}
			}

			// Collect error but continue processing to gather ALL errors
			if runBlock.PreflightCheck.Error != nil {
				if _, err := dr.HandleAPIErrorResponse(runBlock.PreflightCheck.Error.Code, errorMessage, false); err != nil {
					dr.Logger.Error("failed to handle API error response", "error", err)
				}
			} else {
				if _, err := dr.HandleAPIErrorResponse(500, errorMessage, false); err != nil {
					dr.Logger.Error("failed to handle API error response", "error", err)
				}
			}
			// Continue processing instead of returning early - this allows collection of all errors
		}
	}

	// Check if there are already accumulated errors - if so, skip expensive operations for fail-fast behavior
	existingErrorsWithID := utils.GetRequestErrorsWithActionID(dr.RequestID)
	if len(existingErrorsWithID) > 0 {
		dr.Logger.Info("errors already accumulated, skipping expensive operations for fail-fast behavior", "actionID", res.ActionID, "errorCount", len(existingErrorsWithID))
		// Skip all expensive operations (LLM, Python, HTTP, Exec) but continue to process response resource
		return true, nil
	}

	// Process Exec step, if defined
	if runBlock.Exec != nil && runBlock.Exec.Command != "" {
		if err := dr.processResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
			return dr.HandleExec(res.ActionID, runBlock.Exec)
		}); err != nil {
			dr.Logger.Error("exec error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Python step, if defined
	if runBlock.Python != nil && runBlock.Python.Script != "" {
		if err := dr.processResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
			return dr.HandlePython(res.ActionID, runBlock.Python)
		}); err != nil {
			dr.Logger.Error("python error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), true)
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
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Check if any action was actually performed
	hasExec := runBlock.Exec != nil && runBlock.Exec.Command != ""
	hasPython := runBlock.Python != nil && runBlock.Python.Script != ""
	hasChat := runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil)
	hasHTTP := runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != ""

	// If no actions were performed, return false to indicate no processing occurred
	if !hasExec && !hasPython && !hasChat && !hasHTTP {
		dr.Logger.Debug("No run actions defined, skipping resource processing", "actionID", res.ActionID)
		return false, nil
	}

	return true, nil
}

// loadResourceWithFallbackResolver tries to load a resource file with different resource types as fallback.
func (dr *DependencyResolver) loadResourceWithFallbackResolver(file string) (interface{}, error) {
	resourceTypes := []ResourceType{Resource, LLMResource, HTTPResource, PythonResource, ExecResource}

	for _, resourceType := range resourceTypes {
		res, err := dr.LoadResourceFn(dr.Context, file, resourceType)
		if err != nil {
			dr.Logger.Debug("failed to load resource with type", "file", file, "type", resourceType, "error", err)
			continue
		}

		dr.Logger.Debug("successfully loaded resource", "file", file, "type", resourceType)

		// If we successfully loaded as a specific resource type, try to convert it to Resource type
		if resourceType != Resource {
			// Try to load the same file as Resource type
			resourceRes, err := dr.LoadResourceFn(dr.Context, file, Resource)
			if err != nil {
				dr.Logger.Debug("failed to convert resource to Resource type", "file", file, "originalType", resourceType, "error", err)
				// Continue with the original loaded resource if conversion fails
			} else {
				return resourceRes, nil
			}
		}

		return res, nil
	}

	return nil, fmt.Errorf("failed to load resource with any type for file %s", file)
}

// castToResource robustly casts a loaded resource to pklResource.Resource
func (dr *DependencyResolver) castToResource(res interface{}, file string) (*pklResource.Resource, error) {
	// Try direct pointer cast first
	if ptr, ok := res.(*pklResource.Resource); ok {
		return ptr, nil
	}

	// Try value cast
	if resource, ok := res.(pklResource.Resource); ok {
		return &resource, nil
	}

	// Check if we loaded a specific resource type instead of Resource
	if _, ok := res.(*pklLLM.LLMImpl); ok {
		dr.Logger.Warn("loaded LLM resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as LLM type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklHTTP.HTTPImpl); ok {
		dr.Logger.Warn("loaded HTTP resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as HTTP type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklPython.PythonImpl); ok {
		dr.Logger.Warn("loaded Python resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as Python type but expected Resource type for file %s", file)
	}

	if _, ok := res.(*pklExec.ExecImpl); ok {
		dr.Logger.Warn("loaded Exec resource as specific type, this may indicate a schema issue", "file", file)
		return nil, fmt.Errorf("resource loaded as Exec type but expected Resource type for file %s", file)
	}

	return nil, fmt.Errorf("failed to cast resource to pklResource.Resource for file %s (actual type: %T)", file, res)
}

// pullOllamaModel pulls a single Ollama model using the ollama CLI
func (dr *DependencyResolver) pullOllamaModel(ctx context.Context, model string) error {
	dr.Logger.Info("pulling Ollama model", "model", model)

	// First ensure Ollama server is running
	if err := dr.ensureOllamaServerRunning(ctx); err != nil {
		return fmt.Errorf("failed to ensure Ollama server is running: %w", err)
	}

	// Use a timeout for the model pull to prevent hanging
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Try to pull the exact model first
	stdout, stderr, exitCode, err := kdepsexec.KdepsExec(
		pullCtx,
		"ollama",
		[]string{"pull", model},
		"",    // use current directory
		false, // don't use env file
		false, // don't run in background
		dr.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to execute ollama pull: %w", err)
	}

	if exitCode == 0 {
		dr.Logger.Info("successfully pulled Ollama model", "model", model)
		return nil
	}

	// If exact model pull failed, try to find and pull a variant
	dr.Logger.Info("exact model pull failed, trying to find similar models", "model", model, "exitCode", exitCode, "stderr", stderr)
	if variantModel, err := dr.findModelVariant(ctx, model); err == nil && variantModel != "" {
		dr.Logger.Info("found similar model variant, attempting to pull", "original", model, "variant", variantModel)

		// Try pulling the variant
		stdout, stderr, exitCode, err = kdepsexec.KdepsExec(
			pullCtx,
			"ollama",
			[]string{"pull", variantModel},
			"",    // use current directory
			false, // don't use env file
			false, // don't run in background
			dr.Logger,
		)
		if err != nil {
			return fmt.Errorf("failed to execute ollama pull for variant: %w", err)
		}

		if exitCode == 0 {
			dr.Logger.Info("successfully pulled Ollama model variant", "original", model, "variant", variantModel)
			return nil
		}
	}

	return fmt.Errorf("ollama pull failed with exit code %d: stdout=%s, stderr=%s", exitCode, stdout, stderr)
}

// ensureOllamaServerRunning ensures the Ollama server is running
func (dr *DependencyResolver) ensureOllamaServerRunning(ctx context.Context) error {
	// Check if server is running by trying to list models
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, _, exitCode, err := kdepsexec.KdepsExec(
		checkCtx,
		"ollama",
		[]string{"list"},
		"",
		false,
		false,
		dr.Logger,
	)

	if err == nil && exitCode == 0 {
		// Server is already running
		return nil
	}

	dr.Logger.Info("Ollama server not running, starting it")

	// Start Ollama server in background
	serverCtx, serverCancel := context.WithTimeout(ctx, 30*time.Second)
	defer serverCancel()

	_, _, _, err = kdepsexec.KdepsExec(
		serverCtx,
		"ollama",
		[]string{"serve"},
		"",
		false,
		true, // run in background
		dr.Logger,
	)

	if err != nil {
		return fmt.Errorf("failed to start Ollama server: %w", err)
	}

	// Wait a bit for server to start
	time.Sleep(2 * time.Second)

	return nil
}

// findModelVariant tries to find a similar model variant
func (dr *DependencyResolver) findModelVariant(ctx context.Context, model string) (string, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stdout, _, exitCode, err := kdepsexec.KdepsExec(
		checkCtx,
		"ollama",
		[]string{"list"},
		"",
		false,
		false,
		dr.Logger,
	)

	if err != nil || exitCode != 0 {
		return "", fmt.Errorf("failed to list models: %w", err)
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, model+":") {
			// Found a variant like "llama3.2:1b"
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0], nil
			}
		}
	}

	return "", fmt.Errorf("no variant found for model %s", model)
}
