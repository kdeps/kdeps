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
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklRes "github.com/kdeps/schema/gen/resource"
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
	AgentReader             *agent.PklResourceReader
	AgentDBPath             string
	PklresReader            *pklres.PklResourceReader
	PklresDBPath            string
	DBs                     []*sql.DB     // collection of DB connections used by the resolver
	PklresHelper            *PklresHelper // Helper for pklres operations
	AgentName               string
	RequestID               string
	RequestPklFile          string
	ResponsePklFile         string
	ResponseTargetFile      string
	ProjectDir              string
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
	LoadResourceEntriesFn            func() error                                                          `json:"-"`
	LoadResourceFn                   func(context.Context, string, ResourceType) (interface{}, error)      `json:"-"`
	LoadResourceWithRequestContextFn func(context.Context, string, ResourceType) (interface{}, error)      `json:"-"`
	BuildDependencyStackFn           func(string, map[string]bool) []string                                `json:"-"`
	ProcessRunBlockFn                func(ResourceNodeEntry, *pklRes.Resource, string, bool) (bool, error) `json:"-"`
	ClearItemDBFn                    func() error                                                          `json:"-"`

	// Chat / HTTP injection helpers
	NewLLMFn               func(model string) (*ollama.LLM, error)                                                                                      `json:"-"`
	GenerateChatResponseFn func(context.Context, afero.Fs, *ollama.LLM, *pklLLM.ResourceChat, *tool.PklResourceReader, *logging.Logger) (string, error) `json:"-"`

	DoRequestFn func(*pklHTTP.ResourceHTTPClient) error `json:"-"`

	// Python / Conda execution injector
	ExecTaskRunnerFn func(context.Context, execute.ExecTask) (string, string, error) `json:"-"`

	// Import handling injectors
	WalkFn func(afero.Fs, string, filepath.WalkFunc) error `json:"-"`

	// New injectable helpers
	GetCurrentTimestampFn2    func(string, string) (pkl.Duration, error)
	WaitForTimestampChangeFn2 func(string, pkl.Duration, time.Duration, string) error
	HandleAPIErrorResponseFn  func(int, string, bool) (bool, error)
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

	projectDir := filepath.Join(agentDir, "/project/")
	pklWfFile := filepath.Join(projectDir, "workflow.pkl")

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
	if err := utils.CreateDirectories(ctx, fs, directories); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	// List of files to create (stamp file)
	files := []string{
		filepath.Join(actionDir, graphID),
	}

	if err := utils.CreateFiles(ctx, fs, files); err != nil {
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
	var memoryDBPath, sessionDBPath, toolDBPath, itemDBPath, agentDBPath string

	// Always set agentName from workflow configuration
	agentName := workflowConfiguration.GetAgentID()

	// Set environment variables early to ensure agent reader has proper context
	os.Setenv("KDEPS_CURRENT_AGENT", workflowConfiguration.GetAgentID())
	os.Setenv("KDEPS_CURRENT_VERSION", workflowConfiguration.GetVersion())

	if workflowConfiguration.GetSettings() != nil {
		apiServerMode = workflowConfiguration.GetSettings().APIServerMode != nil && *workflowConfiguration.GetSettings().APIServerMode
		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		if agentSettings != nil {
			installAnaconda = agentSettings.InstallAnaconda != nil && *agentSettings.InstallAnaconda
		}
	}

	// Use configurable kdeps path for tests or default to /.kdeps/
	kdepsBase := os.Getenv("KDEPS_PATH")
	if kdepsBase == "" {
		kdepsBase = "/.kdeps/"
	}

	// Ensure kdepsBase directory exists
	if err := utils.CreateDirectories(ctx, fs, []string{kdepsBase}); err != nil {
		return nil, fmt.Errorf("error creating kdeps base directory: %w", err)
	}

	memoryDBPath = filepath.Join(kdepsBase, agentName+"_memory.db")
	memoryReader, err := memory.InitializeMemory(memoryDBPath)
	if err != nil {
		if memoryReader != nil {
			memoryReader.DB.Close()
		}
		return nil, fmt.Errorf("failed to initialize DB memory: %w", err)
	}

	// Use in-memory database tied to graphID for session
	sessionDBPath = ":memory:"
	sessionReader, err := session.InitializeSession(sessionDBPath)
	if err != nil {
		if sessionReader != nil {
			sessionReader.DB.Close()
		}
		return nil, fmt.Errorf("failed to initialize session DB: %w", err)
	}

	// Use in-memory database tied to graphID for tool
	toolDBPath = ":memory:"
	toolReader, err := tool.InitializeTool(toolDBPath)
	if err != nil {
		if toolReader != nil {
			toolReader.DB.Close()
		}
		return nil, fmt.Errorf("failed to initialize tool DB: %w", err)
	}

	// Use in-memory database tied to graphID for item
	itemDBPath = ":memory:"
	itemReader, err := item.InitializeItem(itemDBPath, nil)
	if err != nil {
		if itemReader != nil {
			itemReader.DB.Close()
		}
		return nil, fmt.Errorf("failed to initialize item DB: %w", err)
	}

	agentDBPath = filepath.Join(actionDir, graphID+"_agent.db")
	agentReader, err := agent.GetGlobalAgentReader(fs, kdepsBase, agentName, workflowConfiguration.GetVersion(), logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent DB: %w", err)
	}

	// Use graphID-specific database file for pklres to share data between evaluator and resolver
	pklresDBPath := filepath.Join(actionDir, graphID+"_pklres.db")
	pklresReader, err := pklres.InitializePklResource(pklresDBPath, graphID)
	if err != nil {
		if pklresReader != nil {
			pklresReader.DB.Close()
		}
		return nil, fmt.Errorf("failed to initialize pklres DB: %w", err)
	}

	// Always reinitialize the evaluator with graphID-specific resource readers
	// This ensures the PKL evaluator uses the same graphID-scoped databases as the resolver
	evaluatorConfig := &evaluator.EvaluatorConfig{
		ResourceReaders: []pkl.ResourceReader{
			memoryReader,
			sessionReader,
			toolReader,
			itemReader,
			agentReader,
			pklresReader, // This pklres reader has the correct graphID
		},
		Logger: logger,
	}

	if err := evaluator.InitializeEvaluator(ctx, evaluatorConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize PKL evaluator: %w", err)
	}

	dependencyResolver := &DependencyResolver{
		Fs:                      fs,
		ResourceDependencies:    make(map[string][]string),
		Logger:                  logger,
		VisitedPaths:            make(map[string]bool),
		Context:                 ctx,
		Environment:             env,
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
		AgentDBPath:             agentDBPath,
		AgentReader:             agentReader,
		PklresReader:            pklresReader,
		PklresDBPath:            pklresDBPath,
		CurrentResourceActionID: "", // Initialize as empty, will be set during resource processing
		DBs: []*sql.DB{
			memoryReader.DB,
			sessionReader.DB,
			toolReader.DB,
			itemReader.DB,
			agentReader.DB,
			pklresReader.DB,
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

	// Initialize the PklresHelper
	dependencyResolver.PklresHelper = NewPklresHelper(dependencyResolver)

	// Default injectable helpers
	dependencyResolver.GetCurrentTimestampFn = dependencyResolver.GetCurrentTimestamp
	dependencyResolver.WaitForTimestampChangeFn = dependencyResolver.WaitForTimestampChange

	// Default injection for broader functions (now that Graph is initialized)
	dependencyResolver.LoadResourceEntriesFn = dependencyResolver.LoadResourceEntries
	dependencyResolver.LoadResourceFn = dependencyResolver.LoadResource
	dependencyResolver.LoadResourceWithRequestContextFn = dependencyResolver.LoadResourceWithRequestContext
	dependencyResolver.BuildDependencyStackFn = dependencyResolver.Graph.BuildDependencyStack
	dependencyResolver.ProcessRunBlockFn = dependencyResolver.ProcessRunBlock
	dependencyResolver.ClearItemDBFn = dependencyResolver.ClearItemDB

	// Chat helpers
	dependencyResolver.NewLLMFn = func(model string) (*ollama.LLM, error) {
		return ollama.New(ollama.WithModel(model))
	}
	dependencyResolver.GenerateChatResponseFn = generateChatResponse
	dependencyResolver.DoRequestFn = dependencyResolver.DoRequest

	// Default Python/Conda runner
	dependencyResolver.ExecTaskRunnerFn = func(ctx context.Context, task execute.ExecTask) (string, string, error) {
		stdout, stderr, _, err := kdepsexec.RunExecTask(ctx, task, dependencyResolver.Logger, false)
		return stdout, stderr, err
	}

	// Import helpers
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
func (dr *DependencyResolver) ProcessResourceStep(resourceID, step string, timeoutPtr *pkl.Duration, handler func() error) error {
	dr.Logger.Debug("processResourceStep: about to call handler", "resourceID", resourceID, "step", step, "handler_is_nil", handler == nil)
	// Canonicalize the resourceID if it's a short ActionID
	canonicalResourceID := resourceID
	if dr.PklresHelper != nil {
		canonicalResourceID = dr.PklresHelper.resolveActionID(resourceID)
		if canonicalResourceID != resourceID {
			dr.Logger.Debug("canonicalized resourceID", "original", resourceID, "canonical", canonicalResourceID)
		}
	}

	dr.Logger.Debug("processResourceStep: getting initial timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step)
	timestamp, err := dr.GetCurrentTimestampFn(canonicalResourceID, step)
	if err != nil {
		dr.Logger.Error("processResourceStep: failed to get initial timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "error", err)
		return fmt.Errorf("%s error: %w", step, err)
	}
	dr.Logger.Debug("processResourceStep: got initial timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "timestamp", timestamp.Value)

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

	dr.Logger.Debug("processResourceStep: executing handler", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "timeout", timeout)
	if handler == nil {
		dr.Logger.Error("processResourceStep: handler is nil", "resourceID", resourceID, "step", step)
		return fmt.Errorf("handler is nil for resourceID %s step %s", resourceID, step)
	}
	dr.Logger.Info("processResourceStep: about to call handler", "resourceID", resourceID, "step", step)
	dr.showProcessingProgress(resourceID, step, "starting")
	err = handler()
	dr.Logger.Info("processResourceStep: handler returned", "resourceID", resourceID, "step", step, "err", err)
	if err != nil {
		dr.Logger.Error("processResourceStep: handler failed", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "error", err)
		return fmt.Errorf("%s error: %w", step, err)
	}
	dr.Logger.Debug("processResourceStep: handler completed successfully", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step)

	// Since handlers are synchronous and complete processing immediately,
	// we can show progress instead of waiting for timestamp changes
	dr.showProcessingProgress(resourceID, step, "completed")
	dr.Logger.Debug("processResourceStep: handler completed successfully", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step)
	return nil
}

// hasResourceOutput checks if a resource already has output files
func (dr *DependencyResolver) hasResourceOutput(resourceID string) bool {
	if dr.Fs == nil || dr.FilesDir == "" || dr.RequestID == "" {
		return false
	}

	// Generate the expected output file path for this resource
	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Check if the output file already exists
	exists, err := afero.Exists(dr.Fs, outputFilePath)
	if err != nil {
		return false
	}

	// If file exists, check if it has content
	if exists {
		info, err := dr.Fs.Stat(outputFilePath)
		if err != nil {
			return false
		}
		// Consider it has output if file size > 0
		return info.Size() > 0
	}

	return false
}

// showProcessingProgress displays a progress indicator for resource processing
func (dr *DependencyResolver) showProcessingProgress(resourceID, step, status string) {
	if dr.Logger == nil {
		return
	}

	// Check if this resource already has output - if so, show 100% for this resource
	if dr.hasResourceOutput(resourceID) {
		dr.Logger.Info("processing progress",
			"resourceID", resourceID,
			"step", step,
			"status", "already_completed",
			"progress", "[====================] 100% (cached)")
		return
	}

	// Calculate progress based on total resources processed
	totalResources := len(dr.Resources)
	if totalResources == 0 {
		return
	}

	// Count completed resources (either processed or have existing output)
	completedCount := 0
	for _, resource := range dr.Resources {
		if dr.FileRunCounter[resource.File] > 0 || dr.hasResourceOutput(resource.ActionID) {
			completedCount++
		}
	}

	// Calculate percentage
	percentage := (completedCount * 100) / totalResources

	// Create progress bar (20 characters wide)
	barWidth := 20
	filledWidth := (percentage * barWidth) / 100

	progressBar := "["
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			progressBar += "="
		} else if i == filledWidth {
			progressBar += ">"
		} else {
			progressBar += " "
		}
	}
	progressBar += "]"

	dr.Logger.Info("processing progress",
		"resourceID", resourceID,
		"step", step,
		"status", status,
		"progress", fmt.Sprintf("%s %d%% (%d/%d)", progressBar, percentage, completedCount, totalResources))
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
			if dr.MemoryReader != nil && dr.MemoryReader.DB != nil {
				dr.MemoryReader.DB.Close()
			}
			if dr.SessionReader != nil && dr.SessionReader.DB != nil {
				dr.SessionReader.DB.Close()
			}
			if dr.ToolReader != nil && dr.ToolReader.DB != nil {
				dr.ToolReader.DB.Close()
			}
			if dr.ItemReader != nil && dr.ItemReader.DB != nil {
				dr.ItemReader.DB.Close()
			}
			if dr.AgentReader != nil {
				dr.AgentReader.Close()
			}
			if dr.PklresReader != nil && dr.PklresReader.DB != nil {
				dr.PklresReader.DB.Close()
			}

			// Close the singleton evaluator
			if evaluatorMgr, err := evaluator.GetEvaluatorManager(); err == nil {
				if err := evaluatorMgr.Close(); err != nil {
					dr.Logger.Error("failed to close PKL evaluator", "error", err)
				}
			}

			// Remove the session DB file
			if err := dr.Fs.RemoveAll(dr.SessionDBPath); err != nil {
				dr.Logger.Error("failed to delete the SessionDB file", "file", dr.SessionDBPath, "error", err)
			}
			// Note: Do not delete the pklres database file here, as the response resource needs to access
			// LLM content that was stored during workflow execution
			// if err := dr.Fs.RemoveAll(dr.PklresDBPath); err != nil {
			// 	dr.Logger.Error("failed to delete the PklresDB file", "file", dr.PklresDBPath, "error", err)
			// }

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

			// Load the resource with request context if in API server mode
			var resPkl interface{}
			var err error
			if dr.APIServerMode {
				resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
			} else {
				resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
			}
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
				proceed, err := dr.ProcessRunBlockFn(res, rsc, nodeActionID, false)
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
					if dr.APIServerMode {
						resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
					} else {
						resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
					}
					if err != nil {
						return dr.HandleAPIErrorResponse(500, err.Error(), true)
					}

					// Explicitly type rsc as *pklRes.Resource
					rsc, ok = resPkl.(*pklRes.Resource)
					if !ok {
						return dr.HandleAPIErrorResponse(500, "failed to cast resource to *pklRes.Resource for file "+res.File, true)
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

			// Process APIResponse once, outside the items loop
			if dr.APIServerMode && rsc.Run != nil && rsc.Run.APIResponse != nil {
				if err := dr.CreateResponsePklFile(*rsc.Run.APIResponse); err != nil {
					return dr.HandleAPIErrorResponse(500, err.Error(), true)
				}
			}
		}
	}

	// Close the DB
	if dr.MemoryReader != nil && dr.MemoryReader.DB != nil {
		dr.MemoryReader.DB.Close()
	}
	if dr.SessionReader != nil && dr.SessionReader.DB != nil {
		dr.SessionReader.DB.Close()
	}
	if dr.ToolReader != nil && dr.ToolReader.DB != nil {
		dr.ToolReader.DB.Close()
	}
	if dr.ItemReader != nil && dr.ItemReader.DB != nil {
		dr.ItemReader.DB.Close()
	}
	if dr.AgentReader != nil {
		dr.AgentReader.Close()
	}
	if dr.PklresReader != nil && dr.PklresReader.DB != nil {
		dr.PklresReader.DB.Close()
	}

	// Note: Evaluator is closed by the caller after EvalPklFormattedResponseFile
	// to ensure it's available for response file evaluation

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
	// Note: Do not delete the pklres database file here, as the response resource needs to access
	// LLM content that was stored during workflow execution
	// if err := dr.Fs.RemoveAll(dr.PklresDBPath); err != nil {
	// 	dr.Logger.Error("failed to delete the PklresDB file", "file", dr.PklresDBPath, "error", err)
	// }

	// Log the final file run counts
	for file, count := range dr.FileRunCounter {
		dr.Logger.Info("file run count", "file", file, "count", count)
	}

	dr.Logger.Debug(messages.MsgAllResourcesProcessed)
	return false, nil
}

// processRunBlock handles the runBlock processing for a resource, excluding APIResponse.
func (dr *DependencyResolver) ProcessRunBlock(res ResourceNodeEntry, rsc *pklRes.Resource, actionID string, hasItems bool) (bool, error) {
	// Increment the run counter for this file
	dr.FileRunCounter[res.File]++
	dr.Logger.Info("processing run block for file", "file", res.File, "runCount", dr.FileRunCounter[res.File], "actionID", actionID)

	// Debug logging for Chat block values
	if rsc.Run != nil && rsc.Run.Chat != nil {
		dr.Logger.Info("processRunBlock: Chat block found", "actionID", actionID,
			"model", rsc.Run.Chat.Model,
			"prompt_nil", rsc.Run.Chat.Prompt == nil,
			"scenario_nil", rsc.Run.Chat.Scenario == nil)
	} else {
		dr.Logger.Info("processRunBlock: No Chat block", "actionID", actionID,
			"run_nil", rsc.Run == nil,
			"chat_nil", rsc.Run == nil || rsc.Run.Chat == nil)
	}

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
			if runBlock.PreflightCheck.Error != nil && runBlock.PreflightCheck.Error.Message != nil && *runBlock.PreflightCheck.Error.Message != "" {
				// Use the custom error message if provided
				errorMessage = *runBlock.PreflightCheck.Error.Message
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
			if runBlock.PreflightCheck.Error != nil && runBlock.PreflightCheck.Error.Code != nil {
				_, _ = dr.HandleAPIErrorResponse(*runBlock.PreflightCheck.Error.Code, errorMessage, false)
			} else {
				_, _ = dr.HandleAPIErrorResponse(500, errorMessage, false)
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

	// Store initial resource state in pklres for timestamp lookups
	// This ensures that when processResourceStep calls GetCurrentTimestamp, the resource exists in pklres
	if dr.PklresHelper != nil {
		dr.Logger.Debug("storing initial resource state in pklres for timestamp lookups", "actionID", res.ActionID)

		// Store initial state for each step type that might be processed
		if runBlock.Exec != nil && runBlock.Exec.Command != "" {
			if err := dr.storeInitialResourceState(res.ActionID, "exec", runBlock.Exec); err != nil {
				dr.Logger.Warn("failed to store initial exec resource state", "actionID", res.ActionID, "error", err)
			}
		}

		if runBlock.Python != nil && runBlock.Python.Script != "" {
			if err := dr.storeInitialResourceState(res.ActionID, "python", runBlock.Python); err != nil {
				dr.Logger.Warn("failed to store initial python resource state", "actionID", res.ActionID, "error", err)
			}
		}

		if runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil) {
			if err := dr.storeInitialResourceState(res.ActionID, "llm", runBlock.Chat); err != nil {
				dr.Logger.Warn("failed to store initial llm resource state", "actionID", res.ActionID, "error", err)
			}
		}

		if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
			if err := dr.storeInitialResourceState(res.ActionID, "client", runBlock.HTTPClient); err != nil {
				dr.Logger.Warn("failed to store initial http client resource state", "actionID", res.ActionID, "error", err)
			}
		}
	}

	// Process Exec step, if defined
	if runBlock.Exec != nil && runBlock.Exec.Command != "" {
		if err := dr.ProcessResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
			return dr.HandleExec(res.ActionID, runBlock.Exec)
		}); err != nil {
			dr.Logger.Error("exec error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Python step, if defined
	if runBlock.Python != nil && runBlock.Python.Script != "" {
		if err := dr.ProcessResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
			return dr.HandlePython(res.ActionID, runBlock.Python)
		}); err != nil {
			dr.Logger.Error("python error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Chat (LLM) step, if defined
	if runBlock.Chat != nil && runBlock.Chat.Model != "" && (runBlock.Chat.Prompt != nil || runBlock.Chat.Scenario != nil) {
		model := ""
		promptNil := true
		if runBlock.Chat != nil {
			model = runBlock.Chat.Model
			promptNil = runBlock.Chat.Prompt == nil
		}
		dr.Logger.Info("processRunBlock: about to call LLM handler", "actionID", res.ActionID, "model", model, "promptNil", promptNil)
		dr.Logger.Info("[DEBUG] About to process LLM chat step", "actionID", res.ActionID)
		dr.Logger.Info("Processing LLM chat step", "actionID", res.ActionID, "hasPrompt", runBlock.Chat.Prompt != nil, "hasScenario", runBlock.Chat.Scenario != nil)
		if runBlock.Chat.Scenario != nil {
			dr.Logger.Info("Scenario present", "length", len(*runBlock.Chat.Scenario))
		}
		if err := dr.ProcessResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
			return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
		}); err != nil {
			dr.Logger.Error("LLM chat error", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
		}
		dr.Logger.Info("[DEBUG] Finished processing LLM chat step", "actionID", res.ActionID)
	} else {
		// Debug logging to see why the LLM step is being skipped
		dr.Logger.Info("processRunBlock: LLM step skipped - debug info", "actionID", res.ActionID,
			"runBlock.Chat_nil", runBlock.Chat == nil,
			"model_empty", func() bool {
				if runBlock.Chat != nil {
					return runBlock.Chat.Model == ""
				} else {
					return true
				}
			}(),
			"prompt_and_scenario_nil", func() bool {
				if runBlock.Chat != nil {
					return (runBlock.Chat.Prompt == nil && runBlock.Chat.Scenario == nil)
				} else {
					return true
				}
			}())
		if runBlock.Chat != nil {
			dr.Logger.Info("processRunBlock: Chat block details", "actionID", res.ActionID,
				"model", runBlock.Chat.Model,
				"prompt_nil", runBlock.Chat.Prompt == nil,
				"scenario_nil", runBlock.Chat.Scenario == nil)
		}
		dr.Logger.Info("Skipping LLM chat step", "actionID", res.ActionID, "chatNil",
			runBlock.Chat == nil, "modelEmpty", runBlock.Chat == nil || runBlock.Chat.Model == "",
			"promptAndScenarioNil", runBlock.Chat != nil && runBlock.Chat.Prompt == nil &&
				runBlock.Chat.Scenario == nil)

		// Fallback: If this file is an LLM resource, try to load and process as LLM
		resourceType := dr.detectResourceType(res.File)
		if resourceType == LLMResource {
			dr.Logger.Warn("Fallback: Entered fallback LLM processing block", "file", res.File, "actionID", res.ActionID)
			llmImplIface, err := pklLLM.Load(dr.Context, nil, pkl.FileSource(res.File))
			if err != nil {
				dr.Logger.Error("Fallback: Failed to load as LLM resource", "file", res.File, "error", err)
				return dr.HandleAPIErrorResponse(500, fmt.Sprintf("Failed to load LLM resource for file %s: %v", res.File, err), true)
			}
			llmImpl, ok := llmImplIface.(*pklLLM.LLMImpl)
			if ok && llmImpl != nil {
				dr.Logger.Warn("Fallback: Loaded LLMImpl, iterating resources", "resourceCount", len(llmImpl.Resources))
				for llmActionID, llmRes := range llmImpl.Resources {
					dr.Logger.Warn("Fallback: Processing LLM resource", "llmActionID", llmActionID, "llmRes_nil", llmRes == nil)
					if llmRes != nil && llmRes.Model != "" && (llmRes.Prompt != nil || llmRes.Scenario != nil) {
						dr.Logger.Info("Fallback: About to call processResourceStep for LLM resource", "llmActionID", llmActionID, "handler_is_nil", false)
						err := dr.ProcessResourceStep(llmActionID, "llm", llmRes.TimeoutDuration, func() error {
							return dr.HandleLLMChat(llmActionID, llmRes)
						})
						if err != nil {
							dr.Logger.Error("Fallback: LLM chat error", "actionID", llmActionID, "error", err)
							return dr.HandleAPIErrorResponse(500, fmt.Sprintf("LLM chat failed for resource: %s - %s", llmActionID, err), true)
						}
						dr.Logger.Info("Fallback: Finished processResourceStep for LLM resource", "llmActionID", llmActionID)
					}
				}
			}
		}
	}

	// Process HTTP Client step, if defined
	if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
		if err := dr.ProcessResourceStep(res.ActionID, "client", runBlock.HTTPClient.TimeoutDuration, func() error {
			return dr.HandleHTTPClient(res.ActionID, runBlock.HTTPClient)
		}); err != nil {
			dr.Logger.Error("HTTP client error:", res.ActionID)
			return dr.HandleAPIErrorResponse(500, fmt.Sprintf("HTTP client failed for resource: %s - client error: %s", res.ActionID, err), true)
		}
	}

	return true, nil
}

// storeInitialResourceState stores the initial state of a resource in pklres for timestamp lookups
func (dr *DependencyResolver) storeInitialResourceState(actionID, resourceType string, resource interface{}) error {
	if dr.PklresHelper == nil {
		return errors.New("PklresHelper is not initialized")
	}

	// Create a basic PKL content structure for the resource
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@0.4.4#/%s.pkl\"\n\n", strings.Title(resourceType)))
	pklContent.WriteString("Resources {\n")
	pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", actionID))

	// Add basic fields based on resource type
	switch resourceType {
	case "exec":
		if exec, ok := resource.(*pklExec.ResourceExec); ok {
			pklContent.WriteString(fmt.Sprintf("    Command = \"%s\"\n", exec.Command))
			if exec.TimeoutDuration != nil {
				pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", int(exec.TimeoutDuration.GoDuration().Seconds())))
			} else {
				pklContent.WriteString("    TimeoutDuration = -1.s\n")
			}
		}
	case "python":
		if python, ok := resource.(*pklPython.ResourcePython); ok {
			pklContent.WriteString(fmt.Sprintf("    Script = \"%s\"\n", python.Script))
			if python.TimeoutDuration != nil {
				pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", int(python.TimeoutDuration.GoDuration().Seconds())))
			} else {
				pklContent.WriteString("    TimeoutDuration = 60.s\n")
			}
		}
	case "llm":
		if llm, ok := resource.(*pklLLM.ResourceChat); ok {
			pklContent.WriteString(fmt.Sprintf("    Model = \"%s\"\n", llm.Model))
			if llm.TimeoutDuration != nil {
				pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", int(llm.TimeoutDuration.GoDuration().Seconds())))
			} else {
				pklContent.WriteString("    TimeoutDuration = 60.s\n")
			}
		}
	case "client":
		if client, ok := resource.(*pklHTTP.ResourceHTTPClient); ok {
			pklContent.WriteString(fmt.Sprintf("    Method = \"%s\"\n", client.Method))
			pklContent.WriteString(fmt.Sprintf("    Url = \"%s\"\n", client.Url))
			if client.TimeoutDuration != nil {
				pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", int(client.TimeoutDuration.GoDuration().Seconds())))
			} else {
				pklContent.WriteString("    TimeoutDuration = -1.s\n")
			}
		}
	}

	// Add timestamp
	pklContent.WriteString(fmt.Sprintf("    Timestamp = %g.ns\n", float64(time.Now().UnixNano())))

	// Add empty fields
	pklContent.WriteString("    Env {}\n")
	pklContent.WriteString("    Stderr = \"\"\n")
	pklContent.WriteString("    Stdout = \"\"\n")
	pklContent.WriteString("    File = \"\"\n")
	pklContent.WriteString("    ExitCode = 0\n")
	pklContent.WriteString("    ItemValues {}\n")

	pklContent.WriteString("  }\n")
	pklContent.WriteString("}\n")

	// Store the PKL content in pklres using the canonical ActionID as the key
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
	}
	return dr.PklresHelper.StorePklContent(resourceType, canonicalActionID, pklContent.String())
}

// Exported for testing
func (dr *DependencyResolver) ActivateCondaEnvironment(envName string) error {
	return dr.activateCondaEnvironment(envName)
}

func (dr *DependencyResolver) DeactivateCondaEnvironment() error {
	return dr.deactivateCondaEnvironment()
}

// GetResourcePath returns the resource path using the PklresHelper
func (dr *DependencyResolver) GetResourcePath(resourceType string) string {
	if dr.PklresHelper != nil {
		return dr.PklresHelper.GetResourcePath(resourceType)
	}
	return ""
}

// Export the validation helpers for use in tests
func (dr *DependencyResolver) ValidateRequestParams(file string, allowedParams []string) error {
	return dr.validateRequestParams(file, allowedParams)
}

func (dr *DependencyResolver) ValidateRequestHeaders(file string, allowedHeaders []string) error {
	return dr.validateRequestHeaders(file, allowedHeaders)
}

func (dr *DependencyResolver) ValidateRequestPath(req *gin.Context, allowedRoutes []string) error {
	return dr.validateRequestPath(req, allowedRoutes)
}

func (dr *DependencyResolver) ValidateRequestMethod(req *gin.Context, allowedMethods []string) error {
	return dr.validateRequestMethod(req, allowedMethods)
}
