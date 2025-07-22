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
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/agent"
	kdepsctx "github.com/kdeps/kdeps/pkg/core"
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
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

const statusInternalServerError = 500

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
	LoadResourceEntriesFn            func() error                                                         `json:"-"`
	LoadResourceFn                   func(context.Context, string, ResourceType) (interface{}, error)     `json:"-"`
	LoadResourceWithRequestContextFn func(context.Context, string, ResourceType) (interface{}, error)     `json:"-"`
	BuildDependencyStackFn           func(string, map[string]bool) []string                               `json:"-"`
	ProcessRunBlockFn                func(ResourceNodeEntry, pklRes.Resource, string, bool) (bool, error) `json:"-"`
	ClearItemDBFn                    func() error                                                         `json:"-"`

	// Async pklres polling and reloading
	asyncPollingCancel  context.CancelFunc  `json:"-"`
	processedResources  map[string]bool     `json:"-"` // Track previously executed resources
	dependencyWaitQueue map[string][]string `json:"-"` // Resources waiting for dependencies
	resourceReloadQueue chan string         `json:"-"` // Queue for resources that need reloading

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

	// PKL evaluator
	Evaluator pkl.Evaluator
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, req *gin.Context, logger *logging.Logger, eval pkl.Evaluator) (*DependencyResolver, error) {
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

	var pklWfFile string
	var projectDir string

	// In Docker mode, look for workflow in /agents/<agentname>/<version>/workflow.pkl
	if env.DockerMode == "1" {
		// Find the workflow.pkl in the run directory structure
		pklWfFile = findWorkflowInRun(fs)
		if pklWfFile == "" {
			return nil, fmt.Errorf("workflow.pkl not found in /agents directory structure")
		}
		// Set projectDir to the directory containing workflow.pkl
		projectDir = filepath.Dir(pklWfFile)
	} else {
		// Non-Docker mode: use the traditional /agent/project structure
		projectDir = filepath.Join(agentDir, "/project/")
		pklWfFile = filepath.Join(projectDir, "workflow.pkl")
		exists, err := afero.Exists(fs, pklWfFile)
		if err != nil || !exists {
			return nil, fmt.Errorf("error checking %s: %w", pklWfFile, err)
		}
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

	// Use configurable shared volume path for tests or default to /.kdeps/
	kdepsBase := os.Getenv("KDEPS_SHARED_VOLUME_PATH")
	if kdepsBase == "" {
		kdepsBase = "/.kdeps/"
	}
	// Set the environment variable for pklres agent resolution
	os.Setenv("KDEPS_SHARED_VOLUME_PATH", kdepsBase)

	if workflowConfiguration.GetSettings() != nil {
		apiServerMode = workflowConfiguration.GetSettings().APIServerMode != nil && *workflowConfiguration.GetSettings().APIServerMode
		logger.Debug("APIServerMode set from workflow configuration", "apiServerMode", apiServerMode, "APIServerMode_nil", workflowConfiguration.GetSettings().APIServerMode == nil)
		if workflowConfiguration.GetSettings().APIServerMode != nil {
			logger.Debug("APIServerMode value", "value", *workflowConfiguration.GetSettings().APIServerMode)
		}
		agentSettings := workflowConfiguration.GetSettings().AgentSettings
		if agentSettings != nil {
			installAnaconda = agentSettings.InstallAnaconda != nil && *agentSettings.InstallAnaconda
		}
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

	// Use the unified context system
	kdepsCtx := kdepsctx.GetContext()
	if kdepsCtx == nil {
		return nil, errors.New("unified context not initialized")
	}

	// Update the unified context for this workflow
	err = kdepsctx.UpdateContext(graphID, workflowConfiguration.GetAgentID(), workflowConfiguration.GetVersion(), kdepsBase)
	if err != nil {
		return nil, fmt.Errorf("failed to update unified context: %w", err)
	}

	agentReader := kdepsCtx.AgentReader
	pklresReader := kdepsCtx.PklresReader

	// Use the passed evaluator directly
	// The evaluator should already be initialized with the correct resource readers from main
	if eval == nil {
		return nil, fmt.Errorf("evaluator is required but was nil")
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
		PklresDBPath:            "",   // No longer needed with in-memory key-value store
		CurrentResourceActionID: "",   // Initialize as empty, will be set during resource processing
		Evaluator:               eval, // Store the PKL evaluator
		DBs: []*sql.DB{
			memoryReader.DB,
			sessionReader.DB,
			toolReader.DB,
			itemReader.DB,
			agentReader.DB,
			// Note: pklresReader.DB is global and should not be closed here
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

	// Initialize async polling system
	dependencyResolver.processedResources = make(map[string]bool)
	dependencyResolver.dependencyWaitQueue = make(map[string][]string)
	dependencyResolver.resourceReloadQueue = make(chan string, 100)

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
	// Start timing for this resource step
	startTime := time.Now()
	dr.Logger.Info("processResourceStep: starting", "resourceID", resourceID, "step", step, "startTime", startTime.Format(time.RFC3339Nano))
	dr.Logger.Debug("processResourceStep: about to call handler", "resourceID", resourceID, "step", step, "handler_is_nil", handler == nil)
	// Canonicalize the resourceID if it's a short ActionID
	canonicalResourceID := resourceID
	if dr.PklresHelper != nil {
		canonicalResourceID = dr.PklresHelper.resolveActionID(resourceID)
		if canonicalResourceID != resourceID {
			dr.Logger.Debug("canonicalized resourceID", "original", resourceID, "canonical", canonicalResourceID)
		}
	}

	// Update processing status (dependency waiting now happens before resource loading)
	if dr.PklresReader != nil {
		// Check if this resource exists in the dependency graph
		depData, err := dr.PklresReader.GetDependencyData(canonicalResourceID)
		if err == nil && depData != nil {
			// Update status to processing
			if err := dr.PklresReader.UpdateDependencyStatus(canonicalResourceID, "processing", "", nil); err != nil {
				dr.Logger.Warn("Failed to update dependency status to processing", "resourceID", canonicalResourceID, "error", err)
			}
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

	// Execute handler with timeout if specified
	if timeout > 0 {
		dr.Logger.Info("processResourceStep: executing with timeout", "resourceID", resourceID, "step", step, "timeout", timeout)

		done := make(chan error, 1)
		go func() {
			done <- handler()
		}()

		// Progress ticker for timeout countdown
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Create timeout timer once at the beginning
		timeoutTimer := time.NewTimer(timeout)
		defer timeoutTimer.Stop()

		start := time.Now()

		for {
			select {
			case err = <-done:
				dr.Logger.Info("processResourceStep: handler completed", "resourceID", resourceID, "step", step, "elapsed", time.Since(start), "err", err)
				goto handlerComplete
			case <-ticker.C:
				elapsed := time.Since(start)
				remaining := timeout - elapsed
				if remaining > 0 {
					// Calculate progress percentage for timeout
					progress := int((elapsed.Seconds() / timeout.Seconds()) * 100)

					// Create timeout progress bar (20 characters wide)
					barWidth := 20
					filledWidth := (progress * barWidth) / 100

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

					dr.Logger.Info("processResourceStep: timeout progress",
						"resourceID", resourceID,
						"step", step,
						"progress", fmt.Sprintf("%s %d%% (%v/%v)", progressBar, progress, elapsed.Round(time.Second), timeout))
				} else {
					// Elapsed time exceeds timeout, force timeout
					dr.Logger.Error("processResourceStep: forcing timeout (elapsed > timeout)", "resourceID", resourceID, "step", step, "elapsed", elapsed, "timeout", timeout)
					err = fmt.Errorf("%s timed out after %v", step, timeout)
					goto handlerComplete
				}
			case <-timeoutTimer.C:
				elapsed := time.Since(start)
				dr.Logger.Error("processResourceStep: handler timed out", "resourceID", resourceID, "step", step, "timeout", timeout, "elapsed", elapsed)
				err = fmt.Errorf("%s timed out after %v", step, timeout)
				goto handlerComplete
			}
		}
	handlerComplete:
	} else {
		dr.Logger.Info("processResourceStep: executing without timeout", "resourceID", resourceID, "step", step)
		err = handler()
		dr.Logger.Info("processResourceStep: handler returned", "resourceID", resourceID, "step", step, "err", err)
	}
	if err != nil {
		elapsed := time.Since(startTime)
		dr.Logger.Error("processResourceStep: handler failed", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "error", err, "elapsed", elapsed.String(), "elapsedMs", elapsed.Milliseconds())

		// Update dependency status to error
		if dr.PklresReader != nil {
			if updateErr := dr.PklresReader.UpdateDependencyStatus(canonicalResourceID, "error", "", err); updateErr != nil {
				dr.Logger.Warn("Failed to update dependency status to error", "resourceID", canonicalResourceID, "error", updateErr)
			}
		}

		// Handle timeout errors for API server mode
		if strings.Contains(err.Error(), "timed out") {
			dr.Logger.Info("Generating timeout API response", "resourceID", resourceID, "error", err)
			if dr.HandleAPIErrorResponseFn != nil {
				_, apiErr := dr.HandleAPIErrorResponseFn(408, fmt.Sprintf("Request timeout: %v", err), false)
				if apiErr != nil {
					dr.Logger.Error("Failed to generate timeout API response", "error", apiErr)
				}
			}
		}

		return fmt.Errorf("%s error: %w", step, err)
	}
	dr.Logger.Debug("processResourceStep: handler completed successfully", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step)

	// Update dependency status to completed
	if dr.PklresReader != nil {
		if err := dr.PklresReader.UpdateDependencyStatus(canonicalResourceID, "completed", "", nil); err != nil {
			dr.Logger.Warn("Failed to update dependency status to completed", "resourceID", canonicalResourceID, "error", err)
		} else {
			// Log updated dependency status
			pendingDeps := dr.PklresReader.GetPendingDependencies()
			dr.Logger.Info("Resource completed, updated dependency status", "resourceID", canonicalResourceID, "pendingDependencies", pendingDeps)
		}
	}

	// Wait for processing to complete by monitoring timestamp changes
	// This ensures that pklres records are only available after the process is fully finished
	// All resource types now wait for timestamp changes for consistency
	dr.Logger.Debug("processResourceStep: waiting for processing to complete", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step)
	err = dr.WaitForTimestampChangeFn(canonicalResourceID, timestamp, timeout, step)
	if err != nil {
		dr.Logger.Error("processResourceStep: failed to wait for timestamp change", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "error", err)
		return fmt.Errorf("%s error: %w", step, err)
	}

	dr.showProcessingProgress(resourceID, step, "completed")

	// Calculate and log elapsed time
	elapsed := time.Since(startTime)
	dr.Logger.Info("processResourceStep: completed successfully", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "elapsed", elapsed.String(), "elapsedMs", elapsed.Milliseconds())
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
			// Note: PklresReader is global and should not be closed here

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
		return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
	}

	// Debug: Log the dependency graph before building the stack
	dr.Logger.Debug("dependency graph before build", "targetActionID", targetActionID, "dependencies", dr.ResourceDependencies)

	// Build dependency stack for the target action
	stack := dr.BuildDependencyStackFn(targetActionID, visited)

	dr.Logger.Debug("dependency stack after build", "targetActionID", targetActionID, "stack", stack)

	// Enhanced dependency logging with progress indicators
	if len(stack) > 0 {
		dr.Logger.Info("=== DEPENDENCY EXECUTION PLAN ===")
		for i, actionID := range stack {
			deps := dr.ResourceDependencies[actionID]
			if len(deps) > 0 {
				dr.Logger.Info(fmt.Sprintf("dependency [%d/%d] (%s <- %v)", i+1, len(stack), actionID, deps))
			} else {
				dr.Logger.Info(fmt.Sprintf("dependency [%d/%d] (%s <- no dependencies)", i+1, len(stack), actionID))
			}
		}
		dr.Logger.Info("=== END DEPENDENCY PLAN ===")
	} else {
		dr.Logger.Warn("EMPTY DEPENDENCY STACK - This will cause execution issues!")
	}

	// Ensure the target action is always included in the stack, even if it has no dependencies
	found := false
	for _, actionID := range stack {
		if actionID == targetActionID {
			found = true
			break
		}
	}
	if !found {
		stack = append(stack, targetActionID)
		dr.Logger.Debug("added target action to dependency stack", "targetActionID", targetActionID, "stack", stack)
	}

	// Pre-resolve all pklres dependencies based on the execution order
	if dr.PklresReader != nil {
		dr.Logger.Info("Pre-resolving pklres dependencies", "executionOrder", stack)
		if err := dr.PklresReader.PreResolveDependencies(stack, dr.ResourceDependencies); err != nil {
			dr.Logger.Error("Failed to pre-resolve pklres dependencies", "error", err)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Failed to pre-resolve dependencies: %v", err), true)
		}
		dr.Logger.Info("Successfully pre-resolved pklres dependencies", "actionCount", len(stack))

		// Log initial dependency status
		statusSummary := dr.PklresReader.GetDependencyStatusSummary()
		dr.Logger.Info("Initial dependency status", "statusSummary", statusSummary)
	}

	// Set the global pklres reader context before processing resources
	if dr.PklresReader != nil && dr.Workflow != nil {
		err := pklres.UpdateGlobalPklresReaderContext(dr.RequestID, dr.Workflow.GetAgentID(), dr.Workflow.GetVersion(), dr.AgentDir)
		if err != nil {
			dr.Logger.Warn("Failed to update global pklres reader context", "error", err)
		}
	}

	// In API server mode, ensure the request resource is processed first
	if dr.APIServerMode {
		dr.Logger.Debug("API server mode detected, adding virtual request resource", "requestID", dr.RequestID)

		// Create a proper canonical actionID for the request resource
		// Format: @<agentID>/requestResource:<version>
		var agentID, version string
		if dr.Workflow != nil {
			agentID = dr.Workflow.GetAgentID()
			version = dr.Workflow.GetVersion()
		}
		if agentID == "" || version == "" {
			dr.Logger.Error("missing agentID or version for canonical actionID generation", "agentID", agentID, "version", version)
			return dr.HandleAPIErrorResponse(statusInternalServerError, "missing agentID or version for canonical actionID generation", true)
		}
		requestResourceID := pkg.GenerateCanonicalActionID(agentID, "requestResource", version)
		dr.Logger.Debug("created canonical request resource ID", "requestID", dr.RequestID, "canonical", requestResourceID)

		// Add the request resource to the dependency graph
		dr.ResourceDependencies[requestResourceID] = []string{}
		dr.Logger.Debug("added request resource to dependencies", "requestResourceID", requestResourceID)

		// Add the request resource to the Resources list
		requestResourceEntry := ResourceNodeEntry{
			ActionID: requestResourceID,
			File:     "virtual://request.pkl", // Virtual file path for the request resource
		}
		dr.Resources = append(dr.Resources, requestResourceEntry)
		dr.Logger.Debug("added virtual request resource to Resources list", "requestResourceID", requestResourceID)

		// Add the request resource as a dependency for all other resources
		for actionID := range dr.ResourceDependencies {
			if actionID != requestResourceID {
				deps := dr.ResourceDependencies[actionID]
				if deps == nil {
					deps = []string{}
				}
				// Check if request resource is already in dependencies
				found := false
				for _, dep := range deps {
					if dep == requestResourceID {
						found = true
						break
					}
				}
				if !found {
					deps = append(deps, requestResourceID)
					dr.ResourceDependencies[actionID] = deps
					dr.Logger.Debug("added request resource as dependency", "resource", actionID, "requestResourceID", requestResourceID)
				}
			}
		}

		// Don't rebuild the dependency stack in API server mode - keep the original stack
		// The request resource is already added as a dependency to all resources, so the original stack is correct
		dr.Logger.Debug("keeping original dependency stack in API server mode", "stack", stack, "targetActionID", targetActionID)
	}

	// Ensure the response resource is processed last in the dependency stack
	// Find the response resource and move it to the end of the stack if present
	// Create a proper canonical actionID for the response resource
	// Format: @<agentID>/responseResource:<version>
	var agentID, version string
	if dr.Workflow != nil {
		agentID = dr.Workflow.GetAgentID()
		version = dr.Workflow.GetVersion()
	}
	if agentID == "" || version == "" {
		dr.Logger.Error("missing agentID or version for response resource canonical actionID generation", "agentID", agentID, "version", version)
		return dr.HandleAPIErrorResponse(statusInternalServerError, "missing agentID or version for response resource canonical actionID generation", true)
	}
	responseResourceID := pkg.GenerateCanonicalActionID(agentID, "responseResource", version)
	var newStack []string
	var foundResponseResource bool
	for _, id := range stack {
		if id == responseResourceID {
			foundResponseResource = true
			continue
		}
		newStack = append(newStack, id)
	}
	if foundResponseResource {
		newStack = append(newStack, responseResourceID)
		stack = newStack
		dr.Logger.Info("Moved response resource to end of dependency stack for correct execution order", "stack", stack)
	}

	// Process each resource in the dependency stack
	for i, nodeActionID := range stack {
		// Create process ID for this resource execution
		processID := fmt.Sprintf("[%d/%d] *%s*", i+1, len(stack), nodeActionID)

		// Set the process ID for pklres logging
		if dr.PklresReader != nil {
			if err := pklres.UpdateGlobalPklresReaderProcessID(processID); err != nil {
				dr.Logger.Warn("Failed to update pklres process ID", "error", err, "processID", processID)
			}
		}

		// Enhanced execution logging with progress
		dr.Logger.Info(fmt.Sprintf("🚀 EXECUTING %s", processID))
		dr.Logger.Debug("processing resource in dependency stack", "nodeActionID", nodeActionID, "processID", processID)

		// Debug: Log all available resources for comparison
		dr.Logger.Info("available resources for matching", "availableResourceIDs", func() []string {
			var ids []string
			for _, r := range dr.Resources {
				ids = append(ids, r.ActionID)
			}
			return ids
		}())

		resourceFound := false
		for _, res := range dr.Resources {
			if res.ActionID != nodeActionID {
				continue
			}
			resourceFound = true

			// Set the current resource actionID for error context
			dr.CurrentResourceActionID = res.ActionID
			dr.Logger.Debug("found matching resource", "actionID", res.ActionID, "file", res.File)

			// Handle virtual request resource specially
			if res.File == "virtual://request.pkl" {
				dr.Logger.Debug("processing virtual request resource", "actionID", res.ActionID)
				if err := dr.PopulateRequestDataInPklres(); err != nil {
					return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to process request resource: %v", err), true)
				}
				dr.Logger.Debug("virtual request resource processed successfully", "actionID", res.ActionID)
				continue // Skip normal resource processing for virtual request resource
			}

			// Load the resource with request context if in API server mode (keep as generic Resource for now)
			var resPkl interface{}
			var err error
			if dr.APIServerMode {
				resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
			} else {
				resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
			}
			if err != nil {
				return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
			}

			// Check if this resource has dependencies and reload PKL template with fresh data
			if dr.PklresReader != nil {
				canonicalResourceID := res.ActionID
				if dr.PklresHelper != nil {
					canonicalResourceID = dr.PklresHelper.resolveActionID(res.ActionID)
				}

				// Check if this resource has dependencies
				depData, err := dr.PklresReader.GetDependencyData(canonicalResourceID)
				if err == nil && depData != nil && len(depData.Dependencies) > 0 {
					dr.Logger.Info("Resource has dependencies, reloading PKL template with fresh data", "resourceID", res.ActionID, "dependencies", depData.Dependencies)

					// Reload the resource to ensure PKL expressions have access to dependency data
					if dr.APIServerMode {
						resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
					} else {
						resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
					}
					if err != nil {
						return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to reload resource %s: %v", res.ActionID, err), true)
					}

					dr.Logger.Info("Successfully reloaded resource with dependency data", "resourceID", res.ActionID)
				}
			}
			if err != nil {
				return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
			}

			// Explicitly type rsc as *pklRes.Resource
			rsc, ok := resPkl.(pklRes.Resource)
			if !ok {
				return dr.HandleAPIErrorResponse(statusInternalServerError, "failed to cast resource to pklRes.Resource for file "+res.File, true)
			}

			// Reinitialize item database with items, if any
			var items []string
			if rscItems := rsc.GetItems(); rscItems != nil && len(*rscItems) > 0 {
				items = *rscItems
				// Close existing item database
				dr.ItemReader.DB.Close()
				// Reinitialize item database with items
				itemReader, err := item.InitializeItem(dr.ItemDBPath, items)
				if err != nil {
					return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to reinitialize item DB with items: %v", err), true)
				}
				dr.ItemReader = itemReader
				dr.Logger.Info("reinitialized item database with items", "actionID", nodeActionID, "itemCount", len(items))
			}

			// Process run block: once if no items, or once per item
			if len(items) == 0 {
				// Process run block once
				_, err = dr.ProcessRunBlockFn(res, rsc, nodeActionID, false)
				if err != nil {
					return false, err
				}

				// Wait for pklres recording to complete before proceeding to next resource
				// Use configurable timeout: DefaultTimeoutSec > 0 overrides, otherwise use 30s default
				var pklresTimeout time.Duration
				switch {
				case dr.DefaultTimeoutSec > 0:
					pklresTimeout = time.Duration(dr.DefaultTimeoutSec) * time.Second
				case dr.DefaultTimeoutSec == 0:
					pklresTimeout = 0 // unlimited
				default:
					pklresTimeout = 30 * time.Second // reasonable default
				}
				if err := dr.WaitForPklresRecording(res.ActionID, pklresTimeout); err != nil {
					dr.Logger.Warn("Failed to wait for pklres recording, continuing to next resource", "actionID", res.ActionID, "error", err)
				}
			} else {
				for _, itemValue := range items {
					dr.Logger.Info("processing item", "actionID", res.ActionID, "item", itemValue)
					// Set the current item in the database
					query := url.Values{"op": []string{"set"}, "value": []string{itemValue}}
					uri := url.URL{Scheme: "item", RawQuery: query.Encode()}
					if _, err := dr.ItemReader.Read(uri); err != nil {
						dr.Logger.Error("failed to set item", "item", itemValue, "error", err)
						return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to set item %s: %v", itemValue, err), true)
					}

					// Wait for dependencies before reloading resource to ensure PKL templates have access to dependency data
					// This is critical for PKL template expressions like \(client.responseBody("clientResource"))
					if dr.PklresReader != nil {
						canonicalResourceID := res.ActionID
						if dr.PklresHelper != nil {
							canonicalResourceID = dr.PklresHelper.resolveActionID(res.ActionID)
						}

						// Check if this resource has dependencies
						depData, err := dr.PklresReader.GetDependencyData(canonicalResourceID)
						if err == nil && depData != nil && len(depData.Dependencies) > 0 {
							dr.Logger.Debug("Waiting for dependencies before reloading resource for item", "resourceID", res.ActionID, "item", itemValue)
							waitTimeout := 5 * time.Minute
							if err := dr.PklresReader.WaitForDependencies(canonicalResourceID, waitTimeout); err != nil {
								dr.Logger.Error("Timeout waiting for dependencies before reloading resource", "resourceID", res.ActionID, "error", err)
								return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Timeout waiting for dependencies before reloading resource %s: %v", res.ActionID, err), true)
							}
						}
					}

					// reload the resource
					if dr.APIServerMode {
						resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
					} else {
						resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
					}
					if err != nil {
						return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
					}

					// Explicitly type rsc as pklRes.Resource
					rsc, ok = resPkl.(pklRes.Resource)
					if !ok {
						return dr.HandleAPIErrorResponse(statusInternalServerError, "failed to cast resource to pklRes.Resource for file "+res.File, true)
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
					return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to clear item database for resource %s: %v", res.ActionID, err), true)
				}

				// Wait for pklres recording to complete after processing all items
				// Use configurable timeout: DefaultTimeoutSec > 0 overrides, otherwise use 30s default
				var pklresTimeout time.Duration
				switch {
				case dr.DefaultTimeoutSec > 0:
					pklresTimeout = time.Duration(dr.DefaultTimeoutSec) * time.Second
				case dr.DefaultTimeoutSec == 0:
					pklresTimeout = 0 // unlimited
				default:
					pklresTimeout = 30 * time.Second // reasonable default
				}
				if err := dr.WaitForPklresRecording(res.ActionID, pklresTimeout); err != nil {
					dr.Logger.Warn("Failed to wait for pklres recording after items processing, continuing to next resource", "actionID", res.ActionID, "error", err)
				}
			}

			// Process APIResponse once, outside the items loop
			if rscRun := rsc.GetRun(); dr.APIServerMode && rscRun != nil && rscRun.APIResponse != nil {
				if err := dr.CreateResponsePklFile(*rscRun.APIResponse); err != nil {
					return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
				}
			}
		}

		// Check if resource was found in the loaded resources
		if !resourceFound {
			dr.Logger.Error(fmt.Sprintf("❌ RESOURCE NOT FOUND [%d/%d] *%s* - Resource exists in dependency stack but not in loaded resources!", i+1, len(stack), nodeActionID))
			dr.Logger.Error("Available resources:", "loadedResources", func() []string {
				var loaded []string
				for _, r := range dr.Resources {
					loaded = append(loaded, r.ActionID)
				}
				return loaded
			}())
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
	// Note: Evaluator and PklresReader are closed by the caller after EvalPklFormattedResponseFile
	// to ensure they're available for response evaluation

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
func (dr *DependencyResolver) ProcessRunBlock(res ResourceNodeEntry, rsc pklRes.Resource, actionID string, hasItems bool) (bool, error) {
	// Increment the run counter for this file
	dr.FileRunCounter[res.File]++
	dr.Logger.Info("processing run block for file", "file", res.File, "runCount", dr.FileRunCounter[res.File], "actionID", actionID)

	// Processing status tracking removed - simplified to pure key-value store approach

	// Debug logging for Chat block values

	runBlock := rsc.GetRun()
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
				return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Failed to read list from items database for resource %s: %v", actionID, err), true)
			}
			// Parse the []byte result as a JSON array
			var items []string
			if len(result) > 0 {
				if err := json.Unmarshal(result, &items); err != nil {
					dr.Logger.Error("Failed to parse items database result as JSON array", "actionID", actionID, "error", err)
					return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Failed to parse items database result for resource %s: %v", actionID, err), true)
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
			return dr.HandleAPIErrorResponse(statusInternalServerError, "Timeout waiting for items database to have a non-empty list for resource "+actionID, true)
		}
	}

	if dr.APIServerMode {
		// Read the resource file content for validation
		fileContent, err := afero.ReadFile(dr.Fs, res.File)
		if err != nil {
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to read resource file %s: %v", res.File, err), true)
		}

		// Validate request.params
		allowedParams := []string{}
		if runBlock.AllowedParams != nil {
			allowedParams = *runBlock.AllowedParams
		}
		if err := dr.validateRequestParams(string(fileContent), allowedParams); err != nil {
			dr.Logger.Error("request params validation failed", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Request params validation failed for resource %s: %v", res.ActionID, err), true)
		}

		// Validate request.header
		allowedHeaders := []string{}
		if runBlock.AllowedHeaders != nil {
			allowedHeaders = *runBlock.AllowedHeaders
		}
		if err := dr.validateRequestHeaders(string(fileContent), allowedHeaders); err != nil {
			dr.Logger.Error("request headers validation failed", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Request headers validation failed for resource %s: %v", res.ActionID, err), true)
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
				_, _ = dr.HandleAPIErrorResponse(statusInternalServerError, errorMessage, false)
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
		if err := dr.ProcessResourceStep(res.ActionID, "exec", runBlock.Exec.TimeoutDuration, func() error {
			return dr.HandleExec(res.ActionID, runBlock.Exec)
		}); err != nil {
			dr.Logger.Error("exec error:", res.ActionID)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Exec failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Python step, if defined
	if runBlock.Python != nil && runBlock.Python.Script != "" {
		if err := dr.ProcessResourceStep(res.ActionID, "python", runBlock.Python.TimeoutDuration, func() error {
			return dr.HandlePython(res.ActionID, runBlock.Python)
		}); err != nil {
			dr.Logger.Error("python error:", res.ActionID)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Python script failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process Chat (LLM) step, if defined
	if runBlock.Chat != nil && runBlock.Chat.Model != nil {
		if err := dr.ProcessResourceStep(res.ActionID, "llm", runBlock.Chat.TimeoutDuration, func() error {
			return dr.HandleLLMChat(res.ActionID, runBlock.Chat)
		}); err != nil {
			dr.Logger.Error("LLM chat error", "actionID", res.ActionID, "error", err)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("LLM chat failed for resource: %s - %s", res.ActionID, err), true)
		}
	}

	// Process HTTP Client step, if defined
	if runBlock.HTTPClient != nil && runBlock.HTTPClient.Method != "" && runBlock.HTTPClient.Url != "" {
		if err := dr.ProcessResourceStep(res.ActionID, "client", runBlock.HTTPClient.TimeoutDuration, func() error {
			return dr.HandleHTTPClient(res.ActionID, runBlock.HTTPClient)
		}); err != nil {
			dr.Logger.Error("HTTP client error:", res.ActionID)
			return dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("HTTP client failed for resource: %s - client error: %s", res.ActionID, err), true)
		}
	}

	return true, nil
}

// Exported for testing
func (dr *DependencyResolver) ActivateCondaEnvironment(envName string) error {
	return dr.activateCondaEnvironment(envName)
}

func (dr *DependencyResolver) DeactivateCondaEnvironment() error {
	return dr.deactivateCondaEnvironment()
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

// WaitForPklresRecording waits for pklres to record the results of a resource execution
// before proceeding to the next resource. This ensures data consistency.
// The timeout is configurable via DefaultTimeoutSec or uses a reasonable default.
func (dr *DependencyResolver) WaitForPklresRecording(actionID string, timeout time.Duration) error {
	if dr.PklresReader == nil || dr.PklresHelper == nil {
		dr.Logger.Debug("PklresReader or PklresHelper is nil, skipping pklres recording verification", "actionID", actionID)
		return nil
	}

	canonicalActionID := dr.PklresHelper.resolveActionID(actionID)

	// Use configurable timeout: DefaultTimeoutSec > 0 overrides, otherwise use provided timeout
	var configurableTimeout time.Duration
	switch {
	case dr.DefaultTimeoutSec > 0:
		configurableTimeout = time.Duration(dr.DefaultTimeoutSec) * time.Second
	case dr.DefaultTimeoutSec == 0:
		configurableTimeout = 0 // unlimited
	default:
		configurableTimeout = timeout // use provided timeout as fallback
	}

	dr.Logger.Debug("Waiting for pklres recording to complete", "actionID", actionID, "canonicalActionID", canonicalActionID, "timeout", configurableTimeout, "defaultTimeoutSec", dr.DefaultTimeoutSec)

	// Helper function to check if any data exists for this resource
	checkForData := func() bool {
		// Try to get any key from the collection to see if data exists
		// We'll use a simple approach: try to list keys or check for common patterns
		keys := []string{"timestamp", "response", "stdout", "model", "prompt", "file", "command", "script", "url", "method"}

		for _, key := range keys {
			value, err := dr.PklresHelper.Get(canonicalActionID, key)
			if err == nil && value != "" && value != "null" {
				dr.Logger.Debug("Found data in pklres", "actionID", actionID, "key", key, "value", utils.TruncateString(value, 50))
				return true
			}
		}
		return false
	}

	// If timeout is 0 (unlimited), wait indefinitely
	if configurableTimeout == 0 {
		dr.Logger.Debug("Waiting indefinitely for pklres recording (unlimited timeout)", "actionID", actionID)
		pollInterval := 100 * time.Millisecond

		for {
			if checkForData() {
				dr.Logger.Debug("Pklres recording completed", "actionID", actionID, "status", "completed")
				return nil
			}

			dr.Logger.Debug("Still waiting for pklres recording", "actionID", actionID, "pendingDeps", []string{canonicalActionID})
			time.Sleep(pollInterval)
		}
	}

	// Use configurable timeout
	deadline := time.Now().Add(configurableTimeout)
	pollInterval := 100 * time.Millisecond

	// Wait for the resource to set any data in pklres
	for time.Now().Before(deadline) {
		if checkForData() {
			dr.Logger.Debug("Pklres recording completed", "actionID", actionID, "status", "completed")
			return nil
		}

		dr.Logger.Debug("Still waiting for pklres recording", "actionID", actionID, "pendingDeps", []string{canonicalActionID})
		time.Sleep(pollInterval)
	}

	// If we reach here, the resource didn't set any data within the timeout
	// This is normal for resources that don't use pklres for data storage
	dr.Logger.Debug("Resource did not set data in pklres within timeout, proceeding", "actionID", actionID, "timeout", configurableTimeout)
	return nil
}

// findWorkflowInRun searches for workflow.pkl in /agents/<agentname>/<version>/
func findWorkflowInRun(fs afero.Fs) string {
	runDir := "/agents"

	// Check if run directory exists
	if exists, err := afero.Exists(fs, runDir); err != nil || !exists {
		return ""
	}

	// Walk through run directory to find workflow.pkl
	var workflowPath string
	err := afero.Walk(fs, runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "workflow.pkl" {
			workflowPath = path
			return filepath.SkipDir // Stop walking once found
		}
		return nil
	})
	if err != nil {
		return ""
	}

	return workflowPath
}

// StartAsyncPklresPolling starts the async polling system for pklres updates
func (dr *DependencyResolver) StartAsyncPklresPolling(ctx context.Context) {
	pollCtx, cancel := context.WithCancel(ctx)
	dr.asyncPollingCancel = cancel

	// Start the pklres polling goroutine
	go dr.pollPklresUpdates(pollCtx)

	// Start the resource reloading worker
	go dr.processResourceReloads(pollCtx)

	dr.Logger.Info("Started async pklres polling system")
}

// StopAsyncPklresPolling stops the async polling system
func (dr *DependencyResolver) StopAsyncPklresPolling() {
	if dr.asyncPollingCancel != nil {
		dr.asyncPollingCancel()
		dr.Logger.Info("Stopped async pklres polling system")
	}
}

// pollPklresUpdates continuously polls for pklres updates
func (dr *DependencyResolver) pollPklresUpdates(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dr.checkForPklresUpdates()
		}
	}
}

// checkForPklresUpdates checks for new values in pklres and queues resources for reloading
func (dr *DependencyResolver) checkForPklresUpdates() {
	if dr.PklresReader == nil {
		dr.Logger.Debug("PklresReader is nil, skipping update check")
		return
	}

	// Get all pending dependencies with error handling
	pendingDeps := dr.PklresReader.GetPendingDependencies()
	if len(pendingDeps) == 0 {
		dr.Logger.Debug("No pending dependencies found")
		return
	}

	dr.Logger.Debug("Checking pklres updates", "pendingCount", len(pendingDeps))

	for _, actionID := range pendingDeps {
		// Check if this is a valid canonical action ID in our dependency graph
		if !dr.isValidDependencyGraphActionID(actionID) {
			dr.Logger.Debug("Skipping invalid dependency graph actionID", "actionID", actionID)
			continue
		}

		// Check if this resource was previously executed
		if dr.processedResources[actionID] {
			dr.Logger.Debug("Skipping previously executed resource", "actionID", actionID)
			continue
		}

		// Check if dependency data is now available with proper error handling
		depData, err := dr.PklresReader.GetDependencyData(actionID)
		if err != nil {
			dr.Logger.Debug("Failed to get dependency data", "actionID", actionID, "error", err)
			continue
		}

		if depData == nil {
			dr.Logger.Debug("No dependency data available yet", "actionID", actionID)
			continue
		}

		if depData.Status == "completed" {
			dr.Logger.Info("New dependency data available", "actionID", actionID, "status", depData.Status)

			// Queue this resource for reloading
			select {
			case dr.resourceReloadQueue <- actionID:
				dr.Logger.Debug("Queued resource for reloading", "actionID", actionID)
			default:
				dr.Logger.Warn("Resource reload queue is full", "actionID", actionID)
			}
		} else {
			dr.Logger.Debug("Dependency not completed yet", "actionID", actionID, "status", depData.Status)
		}
	}
}

// processResourceReloads processes the resource reload queue synchronously
func (dr *DependencyResolver) processResourceReloads(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case actionID := <-dr.resourceReloadQueue:
			// Process synchronously to maintain dependency order
			dr.reloadAndExecuteResourceSync(actionID)
		}
	}
}

// reloadAndExecuteResourceSync reloads PKL with new dependency values and re-executes synchronously
func (dr *DependencyResolver) reloadAndExecuteResourceSync(actionID string) {
	dr.Logger.Info("Reloading and executing resource", "actionID", actionID)

	// Find the resource entry
	var resourceEntry *ResourceNodeEntry
	for i, res := range dr.Resources {
		if res.ActionID == actionID {
			resourceEntry = &dr.Resources[i]
			break
		}
	}

	if resourceEntry == nil {
		dr.Logger.Warn("Resource not found for reloading", "actionID", actionID)
		return
	}

	// Reload the resource file with fresh dependency data
	ctx := context.Background()
	reloadedResource, err := dr.LoadResourceWithRequestContextFn(ctx, resourceEntry.File, Resource)
	if err != nil {
		dr.Logger.Error("Failed to reload resource", "actionID", actionID, "error", err)
		return
	}

	// Execute the reloaded resource
	switch typedResource := reloadedResource.(type) {
	case pklRes.Resource:
		_, err := dr.ProcessRunBlockFn(*resourceEntry, typedResource, actionID, false)
		if err != nil {
			dr.Logger.Error("Failed to execute reloaded resource", "actionID", actionID, "error", err)
			return
		}

		// Mark as processed
		dr.processedResources[actionID] = true
		dr.Logger.Info("Successfully reloaded and executed resource", "actionID", actionID)

	default:
		dr.Logger.Error("Invalid resource type for reloading", "actionID", actionID, "type", fmt.Sprintf("%T", reloadedResource))
	}
}

// isValidDependencyGraphActionID checks if the actionID is valid in the dependency graph
func (dr *DependencyResolver) isValidDependencyGraphActionID(actionID string) bool {
	// Check if the canonical action ID exists in our resources
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
	}

	// Look for the action ID in our loaded resources
	for _, res := range dr.Resources {
		if res.ActionID == canonicalActionID {
			return true
		}
	}

	return false
}
