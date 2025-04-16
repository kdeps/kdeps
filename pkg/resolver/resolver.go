package resolver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kartographer/graph"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
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
	}

	workflowConfiguration, err := pklWf.LoadFromPath(ctx, pklWfFile)
	if err != nil {
		return nil, err
	}
	dependencyResolver.Workflow = workflowConfiguration
	if workflowConfiguration.GetSettings() != nil {
		dependencyResolver.APIServerMode = workflowConfiguration.GetSettings().APIServerMode

		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		dependencyResolver.AnacondaInstalled = agentSettings.InstallAnaconda
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger.BaseLogger(), dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, errors.New("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
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

			rsc, err := pklRes.LoadFromPath(dr.Context, res.File)
			if err != nil {
				return dr.HandleAPIErrorResponse(500, err.Error(), true)
			}

			runBlock := rsc.Run
			if runBlock == nil {
				continue
			}

			if dr.APIServerMode {
				// Read the resource file content for validation
				fileContent, err := afero.ReadFile(dr.Fs, res.File)
				if err != nil {
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("failed to read resource file %s: %v", res.File, err), true)
				}

				// Validate request.params
				if err := dr.validateRequestParams(string(fileContent), *runBlock.AllowedParams); err != nil {
					dr.Logger.Error("request params validation failed", "actionID", res.ActionID, "error", err)
					return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request params validation failed for resource %s: %v", res.ActionID, err), false)
				}

				// Validate request.header
				if err := dr.validateRequestHeaders(string(fileContent), *runBlock.AllowedHeaders); err != nil {
					dr.Logger.Error("request headers validation failed", "actionID", res.ActionID, "error", err)
					return dr.HandleAPIErrorResponse(400, fmt.Sprintf("Request headers validation failed for resource %s: %v", res.ActionID, err), false)
				}

				// Validate request.path
				if err := dr.validateRequestPath(dr.Request, *runBlock.RestrictToRoutes); err != nil {
					dr.Logger.Info("skipping due to request path validation not allowed", "actionID", res.ActionID, "error", err)
					continue
				}

				// Validate request.method
				if err := dr.validateRequestMethod(dr.Request, *runBlock.RestrictToHTTPMethods); err != nil {
					dr.Logger.Info("skipping due to request method validation not allowed", "actionID", res.ActionID, "error", err)
					continue
				}
			}

			// Skip condition
			if runBlock.SkipCondition != nil && utils.ShouldSkip(runBlock.SkipCondition) {
				dr.Logger.Infof("skip condition met, skipping: %s", res.ActionID)
				continue
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
			if runBlock.Chat != nil && runBlock.Chat.Model != "" && runBlock.Chat.Prompt != "" {
				if err := dr.processResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
					return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
				}); err != nil {
					dr.Logger.Error("lLM chat error:", res.ActionID)
					return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
				}
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

			// API Response
			if dr.APIServerMode && runBlock.APIResponse != nil {
				if err := dr.CreateResponsePklFile(*runBlock.APIResponse); err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}
			}
		}
	}

	// Remove the request stamp file
	if err := dr.Fs.RemoveAll(requestFilePath); err != nil {
		dr.Logger.Error("failed to delete old requestID file", "file", requestFilePath, "error", err)
		return false, err
	}

	dr.Logger.Debug("all resources finished processing")
	return false, nil
}
