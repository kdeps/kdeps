package resolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/kdeps/kdeps/pkg/reactive"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklRes "github.com/kdeps/schema/gen/resource"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

const statusInternalServerError = 500

type DependencyResolver struct {
	// Core reactive system - now primary interface
	*reactive.ReactiveResolver

	// Essential system components
	Fs          afero.Fs
	Logger      *logging.Logger
	Context     context.Context
	Environment *environment.Environment
	Workflow    pklWf.Workflow
	Request     *gin.Context
	Evaluator   pkl.Evaluator

	// Directory paths
	ProjectDir         string
	AgentDir           string
	ActionDir          string
	FilesDir           string
	DataDir            string
	RequestPklFile     string
	ResponsePklFile    string
	ResponseTargetFile string

	// Configuration
	AgentName         string
	RequestID         string
	APIServerMode     bool
	AnacondaInstalled bool
	DefaultTimeoutSec int

	// Legacy components needed by existing code
	PklresHelper *PklresHelper
	PklresReader *pklres.PklResourceReader
	AgentReader  *agent.PklResourceReader

	// Legacy fields for backward compatibility
	ItemReader     *item.PklResourceReader // Item database reader
	ItemDBPath     string
	MemoryReader   *memory.PklResourceReader  // Memory database reader
	SessionReader  *session.PklResourceReader // Session database reader
	ToolReader     *tool.PklResourceReader    // Tool database reader
	Resources      map[string]interface{}     // Resource cache
	FileRunCounter map[string]int             // File run tracking

	// Legacy database paths
	SessionDBPath string
	MemoryDBPath  string
	ToolDBPath    string

	// Legacy dependency tracking
	ResourceDependencies    map[string][]string
	CurrentResourceActionID string // Current resource being processed

	// Legacy function fields
	GetCurrentTimestampFn            func(string, string) (int64, error)
	WaitForTimestampChangeFn         func(string, interface{}, time.Duration, string) error
	HandleAPIErrorResponseFn         func(int, string, bool) (interface{}, error)
	WalkFn                           func(afero.Fs, string, filepath.WalkFunc) error
	LoadResourceEntriesFn            func(string) ([]ResourceNodeEntry, error)
	BuildDependencyStackFn           func([]string, interface{}) ([]interface{}, error)
	LoadResourceWithRequestContextFn func(context.Context, string, interface{}) (interface{}, error)
	LoadResourceFn                   func(context.Context, string, interface{}) (interface{}, error)
	ProcessRunBlockFn                func(ResourceNodeEntry, interface{}, string, bool) (bool, error)
	NewLLMFn                         func(string) (interface{}, error)
	GenerateChatResponseFn           func(context.Context, afero.Fs, interface{}, interface{}, interface{}, *logging.Logger) (string, error)
	ExecTaskRunnerFn                 func(context.Context, interface{}) (string, string, error)
	DoRequestFn                      func(interface{}) error

	// Legacy storage
	storedAPIResponses map[string]string

	// Legacy async processing
	asyncPollingCancel  context.CancelFunc
	processedResources  map[string]bool
	resourceReloadQueue chan string

	// Legacy PKL caching
	pklEvaluationCache map[string]interface{}
	pklCacheEnabled    bool
	pklCacheMaxSize    int
	pklCacheHitCount   int64
	pklCacheMissCount  int64

	// Legacy database connections
	DBs []*sql.DB
}

type ResourceNodeEntry struct {
	ActionID string `pkl:"actionID"`
	File     string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, req *gin.Context, logger *logging.Logger, eval pkl.Evaluator) (*DependencyResolver, error) {
	// Use new reactive resolver constructor
	return NewReactiveDependencyResolver(fs, ctx, env, req, logger, eval)
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
	dr.Logger.Debug("processResourceStep: got initial timestamp", "resourceID", resourceID, "canonicalResourceID", canonicalResourceID, "step", step, "timestamp", timestamp)

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
		// Skip if resource is not the expected type
		if res, ok := resource.(ResourceNodeEntry); ok {
			if dr.FileRunCounter[res.File] > 0 || dr.hasResourceOutput(res.ActionID) {
				completedCount++
			}
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

	_ = filepath.Join(dr.ActionDir, dr.RequestID) // Preserve original variable for reference

	visited := make(map[string]bool)
	targetActionID := dr.Workflow.GetTargetActionID()
	dr.Logger.Debug(messages.MsgProcessingResources)

	_, err := dr.LoadResourceEntriesFn("")
	if err != nil {
		return dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
	}

	// Debug: Log the dependency graph before building the stack
	dr.Logger.Debug("dependency graph before build", "targetActionID", targetActionID, "dependencies", dr.ResourceDependencies)

	// Build dependency stack for the target action
	stack, _ := dr.BuildDependencyStackFn([]string{targetActionID}, visited)

	dr.Logger.Debug("dependency stack after build", "targetActionID", targetActionID, "stack", stack)

	// Enhanced dependency logging with progress indicators
	if len(stack) > 0 {
		dr.Logger.Info("=== DEPENDENCY EXECUTION PLAN ===")
		for i, actionID := range stack {
			if id, ok := actionID.(string); ok {
				deps := dr.ResourceDependencies[id]
				if len(deps) > 0 {
					dr.Logger.Info(fmt.Sprintf("dependency [%d/%d] (%s <- %v)", i+1, len(stack), id, deps))
				} else {
					dr.Logger.Info(fmt.Sprintf("dependency [%d/%d] (%s)", i+1, len(stack), id))
				}
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
		// Convert []interface{} to []string
		stringStack := make([]string, 0, len(stack))
		for _, item := range stack {
			if id, ok := item.(string); ok {
				stringStack = append(stringStack, id)
			}
		}
		dr.Logger.Info("Pre-resolving pklres dependencies", "executionOrder", stringStack)
		if err := dr.PklresReader.PreResolveDependencies(stringStack, dr.ResourceDependencies); err != nil {
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

		// Add the request resource to the Resources map
		requestResourceEntry := ResourceNodeEntry{
			ActionID: requestResourceID,
			File:     "virtual://request.pkl", // Virtual file path for the request resource
		}
		dr.Resources[requestResourceID] = requestResourceEntry
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
	for _, item := range stack {
		if id, ok := item.(string); ok {
			if id == responseResourceID {
				foundResponseResource = true
				continue
			}
			newStack = append(newStack, id)
		}
	}
	if foundResponseResource {
		newStack = append(newStack, responseResourceID)
		// Convert back to []interface{}
		stack = make([]interface{}, len(newStack))
		for i, s := range newStack {
			stack[i] = s
		}
		dr.Logger.Info("Moved response resource to end of dependency stack for correct execution order", "stack", stack)
	}

	// Process resources in parallel based on dependency levels for better performance
	// Convert []interface{} to []string for processResourcesInParallel
	stringStack := make([]string, 0, len(stack))
	for _, item := range stack {
		if id, ok := item.(string); ok {
			stringStack = append(stringStack, id)
		}
	}
	if err := dr.processResourcesInParallel(stringStack); err != nil {
		return false, err
	}

	return true, nil
}

// processResourcesInParallel processes resources in parallel groups based on their dependency levels
func (dr *DependencyResolver) processResourcesInParallel(stack []string) error {
	// Group resources by dependency level (independent resources can run in parallel)
	dependencyLevels := dr.groupResourcesByDependencyLevel(stack)

	dr.Logger.Info("Resource dependency levels for parallel processing",
		"totalLevels", len(dependencyLevels),
		"totalResources", len(stack))

	// Process each dependency level in sequence, but resources within each level in parallel
	for level, resourceIDs := range dependencyLevels {
		dr.Logger.Info(fmt.Sprintf("🚀 PROCESSING DEPENDENCY LEVEL %d", level),
			"resourceCount", len(resourceIDs),
			"resources", resourceIDs)

		if err := dr.processResourceLevel(resourceIDs, level); err != nil {
			return err
		}
	}

	return nil
}

// groupResourcesByDependencyLevel groups resources into levels based on their dependencies
func (dr *DependencyResolver) groupResourcesByDependencyLevel(stack []string) map[int][]string {
	dependencyLevels := make(map[int][]string)
	resourceLevels := make(map[string]int)

	// Calculate the dependency level for each resource
	for _, resourceID := range stack {
		level := dr.calculateResourceDependencyLevel(resourceID, stack, resourceLevels)
		dependencyLevels[level] = append(dependencyLevels[level], resourceID)
		resourceLevels[resourceID] = level
	}

	return dependencyLevels
}

// calculateResourceDependencyLevel calculates the dependency level of a resource
func (dr *DependencyResolver) calculateResourceDependencyLevel(resourceID string, stack []string, calculated map[string]int) int {
	// If already calculated, return cached result
	if level, exists := calculated[resourceID]; exists {
		return level
	}

	// Get dependencies for this resource
	dependencies := dr.ResourceDependencies[resourceID]
	if len(dependencies) == 0 {
		// No dependencies = level 0
		calculated[resourceID] = 0
		return 0
	}

	// Find the maximum level of all dependencies + 1
	maxDepLevel := -1
	for _, dep := range dependencies {
		// Only consider dependencies that are in our processing stack
		depInStack := false
		for _, stackItem := range stack {
			if stackItem == dep {
				depInStack = true
				break
			}
		}

		if depInStack {
			depLevel := dr.calculateResourceDependencyLevel(dep, stack, calculated)
			if depLevel > maxDepLevel {
				maxDepLevel = depLevel
			}
		}
	}

	level := maxDepLevel + 1
	calculated[resourceID] = level
	return level
}

// processResourceLevel processes all resources in a dependency level in parallel
func (dr *DependencyResolver) processResourceLevel(resourceIDs []string, level int) error {
	if len(resourceIDs) == 1 {
		// Single resource, process directly
		return dr.processResource(resourceIDs[0], level, 0, 1)
	}

	// Multiple resources, process in parallel
	dr.Logger.Info(fmt.Sprintf("Processing %d resources in parallel for level %d", len(resourceIDs), level))

	// Use worker pool pattern for controlled parallelism
	workerCount := minInt(len(resourceIDs), 4) // Limit to 4 concurrent workers for resource safety
	resourceChan := make(chan string, len(resourceIDs))
	errorChan := make(chan error, len(resourceIDs))

	// Start workers
	for i := 0; i < workerCount; i++ {
		go func(workerID int) {
			for resourceID := range resourceChan {
				if err := dr.processResource(resourceID, level, workerID, len(resourceIDs)); err != nil {
					errorChan <- err
					return
				}
			}
			errorChan <- nil // Signal successful completion
		}(i)
	}

	// Send resources to workers
	for i, resourceID := range resourceIDs {
		dr.Logger.Debug(fmt.Sprintf("Queuing resource %d/%d for parallel processing", i+1, len(resourceIDs)),
			"resourceID", resourceID, "level", level)
		resourceChan <- resourceID
	}
	close(resourceChan)

	// Wait for all workers to complete and check for errors
	for i := 0; i < workerCount; i++ {
		if err := <-errorChan; err != nil {
			return err
		}
	}

	dr.Logger.Info(fmt.Sprintf("✅ COMPLETED DEPENDENCY LEVEL %d", level),
		"resourceCount", len(resourceIDs))
	return nil
}

// processResource processes a single resource (extracted from the original loop)
func (dr *DependencyResolver) processResource(nodeActionID string, level, workerID, totalInLevel int) error {
	// Create process ID for this resource execution
	processID := fmt.Sprintf("[L%d-W%d] *%s*", level, workerID, nodeActionID)

	// Set the process ID for pklres logging
	if dr.PklresReader != nil {
		dr.PklresReader.ProcessID = processID
	}

	// Enhanced execution logging with progress
	dr.Logger.Info(fmt.Sprintf("🚀 EXECUTING %s", processID))
	dr.Logger.Debug("processing resource in dependency stack", "nodeActionID", nodeActionID, "processID", processID)

	// Debug: Log all available resources for comparison
	dr.Logger.Info("available resources for matching", "availableResourceIDs", func() []string {
		var ids []string
		for _, r := range dr.Resources {
			if resource, ok := r.(ResourceNodeEntry); ok {
				ids = append(ids, resource.ActionID)
			}
		}
		return ids
	}())

	resourceFound := false
	for _, resInterface := range dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
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
					_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to process request resource: %v", err), true)
					return apiErr
				}
				dr.Logger.Debug("virtual request resource processed successfully", "actionID", res.ActionID)
				return nil // Continue to next resource
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
				_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
				return apiErr
			}

			// COMPREHENSIVE DEPENDENCY AND ATTRIBUTE VALIDATION
			if dr.PklresReader != nil && dr.PklresHelper != nil {
				canonicalResourceID := dr.PklresHelper.resolveActionID(res.ActionID)

				// STEP 1: Ensure all attributes are ready for evaluation (comprehensive dependency validation)
				if err := dr.PklresHelper.EnsureAttributesReadyForEvaluation(canonicalResourceID); err != nil {
					dr.Logger.Error("Dependency validation failed before PKL evaluation",
						"resourceID", res.ActionID,
						"canonicalResourceID", canonicalResourceID,
						"error", err)
					_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError,
						fmt.Sprintf("Dependency validation failed for resource %s: %v", res.ActionID, err), true)
					return apiErr
				}

				// STEP 2: Check if this resource has dependencies and reload PKL template with validated data
				depData, err := dr.PklresReader.GetDependencyData(canonicalResourceID)
				if err == nil && depData != nil && len(depData.Dependencies) > 0 {
					dr.Logger.Info("Resource has validated dependencies, reloading PKL template with verified data",
						"resourceID", res.ActionID,
						"dependencies", depData.Dependencies,
						"dependenciesValidated", true)

					// Reload the resource to ensure PKL expressions have access to validated dependency data
					if dr.APIServerMode {
						resPkl, err = dr.LoadResourceWithRequestContextFn(dr.Context, res.File, Resource)
					} else {
						resPkl, err = dr.LoadResourceFn(dr.Context, res.File, Resource)
					}
					if err != nil {
						_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("failed to reload resource %s: %v", res.ActionID, err), true)
						return apiErr
					}

					dr.Logger.Info("Successfully reloaded resource with validated dependency data",
						"resourceID", res.ActionID,
						"dependencyValidationPassed", true)
				} else {
					dr.Logger.Debug("Resource has no dependencies or dependency validation not required",
						"resourceID", res.ActionID)
				}
			}
			if err != nil {
				_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, err.Error(), true)
				return apiErr
			}

			// Explicitly type rsc as *pklRes.Resource
			rsc, ok := resPkl.(pklRes.Resource)
			if !ok {
				_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("invalid resource type returned: %T", resPkl), true)
				return apiErr
			}

			// Process the resource
			if shouldContinue, err := dr.ProcessRunBlockFn(res, rsc, processID, false); err != nil {
				return err
			} else if !shouldContinue {
				return nil
			}

			// Handle APIResponse if present (in API server mode) - MEMORY-ONLY APPROACH
			if dr.APIServerMode && rsc.GetRun() != nil && rsc.GetRun().APIResponse != nil {
				// Build response directly in memory - NO TEMPORARY FILES
				jsonResponse, err := dr.BuildResponseInMemory(*rsc.GetRun().APIResponse)
				if err != nil {
					_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Failed to build response in memory for resource %s: %v", res.ActionID, err), true)
					return apiErr
				}

				// Store all responses in memory for potential future use
				dr.storedAPIResponses[res.ActionID] = jsonResponse
				dr.Logger.Info("APIResponse processed in memory-only",
					"actionID", res.ActionID,
					"responseLength", len(jsonResponse),
					"totalStoredResponses", len(dr.storedAPIResponses))

				// Check if this is the target action - if so, we can terminate
				targetActionID := dr.Workflow.GetTargetActionID()
				if res.ActionID == targetActionID {
					// Only write final response file when we reach the target (required for API consumers)
					if err := afero.WriteFile(dr.Fs, dr.ResponseTargetFile, []byte(jsonResponse), pkg.DefaultOctalFilePerms); err != nil {
						_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("Failed to write final response for target action %s: %v", res.ActionID, err), true)
						return apiErr
					}

					dr.Logger.Info("Target action reached - workflow complete",
						"targetActionID", targetActionID,
						"currentActionID", res.ActionID,
						"finalResponseFile", dr.ResponseTargetFile,
						"intermediateResponsesStored", len(dr.storedAPIResponses)-1)
					return nil // Terminate the workflow as we've reached the target
				} else {
					dr.Logger.Info("Intermediate APIResponse stored in memory, continuing to target",
						"currentActionID", res.ActionID,
						"targetActionID", targetActionID,
						"memoryOnlyPolicy", "no-temp-files")
					// Continue processing - store in memory but don't create files or terminate
				}
			}

			break
		} // close if res, ok := resInterface.(ResourceNodeEntry); ok
	}

	if !resourceFound {
		_, apiErr := dr.HandleAPIErrorResponse(statusInternalServerError, fmt.Sprintf("resource with actionID '%s' not found in loaded resources", nodeActionID), true)
		return apiErr
	}

	return nil
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Removed sequential processing fallback to avoid compilation errors.
// The parallel processing approach is now the primary method.

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

// GetStoredAPIResponses returns all stored API responses (exported for testing and monitoring)
func (dr *DependencyResolver) GetStoredAPIResponses() map[string]string {
	if dr.storedAPIResponses == nil {
		return make(map[string]string)
	}
	// Return a copy to prevent external modification
	result := make(map[string]string)
	for k, v := range dr.storedAPIResponses {
		result[k] = v
	}
	return result
}

// GetStoredAPIResponse returns a specific stored API response by actionID
func (dr *DependencyResolver) GetStoredAPIResponse(actionID string) (string, bool) {
	if dr.storedAPIResponses == nil {
		return "", false
	}
	response, exists := dr.storedAPIResponses[actionID]
	return response, exists
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
	for _, resInterface := range dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
			if res.ActionID == actionID {
				// Create a copy since we can't take address of map value
				resCopy := res
				resourceEntry = &resCopy
				break
			}
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
	for _, resInterface := range dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
			if res.ActionID == canonicalActionID {
				return true
			}
		}
	}

	return false
}

// PKL Evaluation Cache Methods

// initializePklCache initializes the PKL evaluation cache with default settings
func (dr *DependencyResolver) initializePklCache() {
	dr.pklEvaluationCache = make(map[string]interface{})
	dr.pklCacheEnabled = true // Enable by default for performance
	dr.pklCacheMaxSize = 100  // Limit cache size to prevent memory issues
	dr.pklCacheHitCount = 0
	dr.pklCacheMissCount = 0

	dr.Logger.Debug("PKL evaluation cache initialized",
		"enabled", dr.pklCacheEnabled,
		"maxSize", dr.pklCacheMaxSize)
}

// getPklCacheKey generates a cache key for PKL evaluation based on file path and resource type
func (dr *DependencyResolver) getPklCacheKey(resourceFile string, resourceType ResourceType, contextSuffix string) string {
	// Include file modification time to invalidate cache if file changes
	fileInfo, err := dr.Fs.Stat(resourceFile)
	var modTime int64
	if err == nil {
		modTime = fileInfo.ModTime().Unix()
	}

	return fmt.Sprintf("%s:%s:%s:%d", resourceFile, resourceType, contextSuffix, modTime)
}

// getCachedPklEvaluation retrieves a cached PKL evaluation result
func (dr *DependencyResolver) getCachedPklEvaluation(cacheKey string) (interface{}, bool) {
	if !dr.pklCacheEnabled {
		return nil, false
	}

	result, exists := dr.pklEvaluationCache[cacheKey]
	if exists {
		dr.pklCacheHitCount++
		dr.Logger.Debug("PKL cache hit", "key", cacheKey, "hitCount", dr.pklCacheHitCount)
		return result, true
	}

	dr.pklCacheMissCount++
	dr.Logger.Debug("PKL cache miss", "key", cacheKey, "missCount", dr.pklCacheMissCount)
	return nil, false
}

// setCachedPklEvaluation stores a PKL evaluation result in the cache
func (dr *DependencyResolver) setCachedPklEvaluation(cacheKey string, result interface{}) {
	if !dr.pklCacheEnabled {
		return
	}

	// Implement simple LRU eviction if cache is full
	if len(dr.pklEvaluationCache) >= dr.pklCacheMaxSize {
		// Remove one random entry to make space (simple eviction strategy)
		for key := range dr.pklEvaluationCache {
			delete(dr.pklEvaluationCache, key)
			dr.Logger.Debug("PKL cache evicted entry", "evictedKey", key)
			break
		}
	}

	dr.pklEvaluationCache[cacheKey] = result
	dr.Logger.Debug("PKL cache stored", "key", cacheKey, "cacheSize", len(dr.pklEvaluationCache))
}

// clearPklCache clears the entire PKL evaluation cache
func (dr *DependencyResolver) clearPklCache() {
	dr.pklEvaluationCache = make(map[string]interface{})
	dr.pklCacheHitCount = 0
	dr.pklCacheMissCount = 0
	dr.Logger.Debug("PKL cache cleared")
}

// getPklCacheStats returns cache performance statistics
func (dr *DependencyResolver) getPklCacheStats() map[string]interface{} {
	totalRequests := dr.pklCacheHitCount + dr.pklCacheMissCount
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(dr.pklCacheHitCount) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"enabled":       dr.pklCacheEnabled,
		"size":          len(dr.pklEvaluationCache),
		"maxSize":       dr.pklCacheMaxSize,
		"hits":          dr.pklCacheHitCount,
		"misses":        dr.pklCacheMissCount,
		"hitRate":       fmt.Sprintf("%.2f%%", hitRate),
		"totalRequests": totalRequests,
	}
}
