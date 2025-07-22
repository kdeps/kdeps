package resolver

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/kdeps/kdeps/pkg/reactive"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// NewReactiveDependencyResolver creates a new reactive-first dependency resolver
func NewReactiveDependencyResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, req *gin.Context, logger *logging.Logger, eval pkl.Evaluator) (*DependencyResolver, error) {
	// Extract context values
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

	// Initialize reactive system first
	reactiveResolver := reactive.NewReactiveResolver(logger)

	// Emit initialization started
	reactiveResolver.StartAsyncOperation("resolver-init", "initialization", func() (interface{}, error) {
		return "Dependency resolver initialized", nil
	})

	// Determine workflow path
	var pklWfFile, projectDir string
	if env.DockerMode == "1" {
		pklWfFile = findWorkflowInRun(fs)
		if pklWfFile == "" {
			reactiveResolver.EmitError("workflow.pkl not found in /agents directory structure", "resolver-init", nil, "critical")
			return nil, fmt.Errorf("workflow.pkl not found in /agents directory structure")
		}
		projectDir = filepath.Dir(pklWfFile)
	} else {
		projectDir = filepath.Join(agentDir, "/project/")
		pklWfFile = filepath.Join(projectDir, "workflow.pkl")
		exists, err := afero.Exists(fs, pklWfFile)
		if err != nil || !exists {
			reactiveResolver.EmitError(fmt.Sprintf("error checking %s", pklWfFile), "resolver-init", nil, "critical")
			return nil, fmt.Errorf("error checking %s: %w", pklWfFile, err)
		}
	}

	// Create directories reactively
	dataDir := filepath.Join(projectDir, "/data/")
	filesDir := filepath.Join(actionDir, "/files/")
	directories := []string{projectDir, actionDir, filesDir}

	reactiveResolver.StartAsyncOperation("create-directories", "filesystem", func() (interface{}, error) {
		if err := utils.CreateDirectories(ctx, fs, directories); err != nil {
			return nil, fmt.Errorf("error creating directory: %w", err)
		}
		return fmt.Sprintf("Created %d directories", len(directories)), nil
	})

	// Create stamp files
	files := []string{filepath.Join(actionDir, graphID)}
	reactiveResolver.StartAsyncOperation("create-files", "filesystem", func() (interface{}, error) {
		if err := utils.CreateFiles(ctx, fs, files); err != nil {
			return nil, fmt.Errorf("error creating file: %w", err)
		}
		return fmt.Sprintf("Created %d files", len(files)), nil
	})

	// Load workflow configuration reactively
	var workflowConfiguration pklWf.Workflow
	reactiveResolver.StartAsyncOperation("load-workflow", "configuration", func() (interface{}, error) {
		wf, err := pklWf.LoadFromPath(ctx, pklWfFile)
		if err != nil {
			return nil, err
		}
		workflowConfiguration = wf
		return "Workflow configuration loaded", nil
	})

	// Calculate timeout
	defaultTimeoutSec := func() int {
		if v, ok := os.LookupEnv("TIMEOUT"); ok {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return -1
	}()

	// Set up paths
	requestPklFile := filepath.Join(actionDir, "/api/"+graphID+"__request.pkl")
	responsePklFile := filepath.Join(actionDir, "/api/"+graphID+"__response.pkl")
	responseTargetFile := filepath.Join(actionDir, "/api/"+graphID+"__response.json")

	// Wait for workflow loading to complete (placeholder - would use proper async coordination)
	time.Sleep(100 * time.Millisecond)

	// Extract configuration
	var agentName string
	var apiServerMode, installAnaconda bool

	if workflowConfiguration != nil {
		if workflowConfiguration.GetAgentID() != "" {
			agentName = workflowConfiguration.GetAgentID()

			// Set environment variables
			os.Setenv("KDEPS_CURRENT_AGENT", agentName)
			os.Setenv("KDEPS_CURRENT_VERSION", workflowConfiguration.GetVersion())

			reactiveResolver.EmitLog("info", fmt.Sprintf("Agent configured: %s", agentName), "resolver-init", nil)
		}

		if workflowConfiguration.GetSettings() != nil {
			apiServerMode = workflowConfiguration.GetSettings().APIServerMode != nil && *workflowConfiguration.GetSettings().APIServerMode
			agentSettings := workflowConfiguration.GetSettings().AgentSettings
			if agentSettings != nil {
				installAnaconda = agentSettings.InstallAnaconda != nil && *agentSettings.InstallAnaconda
			}

			reactiveResolver.EmitLog("info", fmt.Sprintf("API Server Mode: %t, Anaconda: %t", apiServerMode, installAnaconda), "resolver-init", nil)
		}
	}

	// Create the dependency resolver with reactive core
	resolver := &DependencyResolver{
		ReactiveResolver:   reactiveResolver,
		Fs:                 fs,
		Logger:             logger,
		Context:            ctx,
		Environment:        env,
		Workflow:           workflowConfiguration,
		Request:            req,
		Evaluator:          eval,
		ProjectDir:         projectDir,
		AgentDir:           agentDir,
		ActionDir:          actionDir,
		FilesDir:           filesDir,
		DataDir:            dataDir,
		RequestPklFile:     requestPklFile,
		ResponsePklFile:    responsePklFile,
		ResponseTargetFile: responseTargetFile,
		AgentName:          agentName,
		RequestID:          graphID,
		APIServerMode:      apiServerMode,
		AnacondaInstalled:  installAnaconda,
		DefaultTimeoutSec:  defaultTimeoutSec,
	}

	// Initialize PklresReader
	kdepsBase := os.Getenv("KDEPS_SHARED_VOLUME_PATH")
	if kdepsBase == "" {
		kdepsBase = "/.kdeps/"
	}

	pklresReader, err := pklres.InitializePklResource(graphID, agentName, workflowConfiguration.GetVersion(), kdepsBase, fs)
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize PklresReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize PklresReader: %w", err)
	}
	resolver.PklresReader = pklresReader

	// Initialize AgentReader
	agentReader, err := agent.InitializeAgent(fs, kdepsBase, agentName, workflowConfiguration.GetVersion(), logger)
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize AgentReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize AgentReader: %w", err)
	}
	resolver.AgentReader = agentReader

	// Initialize database readers for backward compatibility
	memoryDBPath := filepath.Join(kdepsBase, agentName+"_memory.db")
	memoryReader, err := memory.InitializeMemory(memoryDBPath)
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize MemoryReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize MemoryReader: %w", err)
	}
	resolver.MemoryReader = memoryReader

	sessionReader, err := session.InitializeSession(":memory:")
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize SessionReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize SessionReader: %w", err)
	}
	resolver.SessionReader = sessionReader

	toolReader, err := tool.InitializeTool(":memory:")
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize ToolReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize ToolReader: %w", err)
	}
	resolver.ToolReader = toolReader

	itemReader, err := item.InitializeItem(":memory:", nil)
	if err != nil {
		reactiveResolver.EmitError("Failed to initialize ItemReader", "resolver-init", nil, "critical")
		return nil, fmt.Errorf("failed to initialize ItemReader: %w", err)
	}
	resolver.ItemReader = itemReader

	// Initialize PklresHelper
	resolver.PklresHelper = NewPklresHelper(resolver)

	// Initialize legacy fields for backward compatibility
	resolver.Resources = make(map[string]interface{})
	resolver.FileRunCounter = make(map[string]int)
	resolver.ResourceDependencies = make(map[string][]string)

	// Initialize legacy database paths
	resolver.SessionDBPath = ":memory:"
	resolver.MemoryDBPath = filepath.Join(kdepsBase, agentName+"_memory.db")
	resolver.ToolDBPath = ":memory:"
	resolver.ItemDBPath = ":memory:"

	// Initialize legacy function fields
	resolver.GetCurrentTimestampFn = func(resourceID, actionID string) (int64, error) {
		return time.Now().Unix(), nil
	}
	resolver.WaitForTimestampChangeFn = func(resourceID string, timestamp interface{}, duration time.Duration, actionID string) error {
		return nil
	}
	resolver.HandleAPIErrorResponseFn = func(code int, message string, critical bool) (interface{}, error) {
		return nil, fmt.Errorf("API error %d: %s", code, message)
	}
	resolver.WalkFn = afero.Walk
	resolver.LoadResourceEntriesFn = func(file string) ([]ResourceNodeEntry, error) {
		// Call the actual LoadResourceEntries method instead of returning empty
		if err := resolver.LoadResourceEntries(); err != nil {
			return []ResourceNodeEntry{}, err
		}
		// Convert the loaded Resources map to ResourceNodeEntry slice
		var entries []ResourceNodeEntry
		for _, resInterface := range resolver.Resources {
			if res, ok := resInterface.(ResourceNodeEntry); ok {
				entries = append(entries, res)
			}
		}
		return entries, nil
	}
	resolver.BuildDependencyStackFn = func(deps []string, context interface{}) ([]interface{}, error) {
		return []interface{}{}, nil
	}
	resolver.LoadResourceWithRequestContextFn = func(ctx context.Context, file string, resourceType interface{}) (interface{}, error) {
		// Call the actual LoadResourceWithRequestContext method
		if rt, ok := resourceType.(ResourceType); ok {
			return resolver.LoadResourceWithRequestContext(ctx, file, rt)
		}
		return resolver.LoadResourceWithRequestContext(ctx, file, Resource) // Default resource type
	}
	resolver.LoadResourceFn = func(ctx context.Context, file string, resourceType interface{}) (interface{}, error) {
		// Call the actual LoadResource method
		if rt, ok := resourceType.(ResourceType); ok {
			return resolver.LoadResource(ctx, file, rt)
		}
		return resolver.LoadResource(ctx, file, Resource) // Default resource type
	}
	resolver.ProcessRunBlockFn = func(res ResourceNodeEntry, rsc interface{}, actionID string, hasItems bool) (bool, error) {
		return false, nil
	}
	resolver.NewLLMFn = func(model string) (interface{}, error) {
		return nil, nil
	}
	resolver.GenerateChatResponseFn = func(ctx context.Context, fs afero.Fs, llm interface{}, chatBlock interface{}, toolReader interface{}, logger *logging.Logger) (string, error) {
		return "", nil
	}
	resolver.ExecTaskRunnerFn = func(ctx context.Context, task interface{}) (string, string, error) {
		return "", "", nil
	}
	resolver.DoRequestFn = func(req interface{}) error {
		return nil
	}
	resolver.storedAPIResponses = make(map[string]string)

	// Initialize legacy async processing
	resolver.processedResources = make(map[string]bool)
	resolver.resourceReloadQueue = make(chan string, 100)

	// Initialize legacy PKL caching
	resolver.pklEvaluationCache = make(map[string]interface{})
	resolver.pklCacheEnabled = true
	resolver.pklCacheMaxSize = 1000
	resolver.pklCacheHitCount = 0
	resolver.pklCacheMissCount = 0

	// Initialize legacy database connections with actual DBs
	resolver.DBs = []*sql.DB{
		memoryReader.DB,
		sessionReader.DB,
		toolReader.DB,
		itemReader.DB,
		agentReader.DB,
		// Note: pklresReader.DB is global and managed separately
	}

	// Mark initialization as complete
	resolver.UpdateOperationProgress("resolver-init", 1.0)
	resolver.EmitLog("info", "Dependency resolver fully initialized", "resolver-init", nil)

	return resolver, nil
}

// ProcessWorkflowReactive processes the workflow using reactive patterns
func (dr *DependencyResolver) ProcessWorkflowReactive() error {
	// Start workflow processing
	dr.StartAsyncOperation("workflow-processing", "workflow", func() (interface{}, error) {
		// Emit workflow started event
		workflowResource := reactive.Resource{
			ID:     "workflow",
			Type:   "workflow",
			Status: "processing",
			Data:   dr.Workflow,
		}
		dr.EmitResourceCreated(workflowResource, nil)

		// Process workflow steps reactively
		if err := dr.processWorkflowSteps(); err != nil {
			workflowResource.Status = "error"
			dr.EmitResourceUpdated(workflowResource, map[string]interface{}{
				"error": err.Error(),
			})
			return nil, err
		}

		// Mark workflow as completed
		workflowResource.Status = "completed"
		dr.EmitResourceUpdated(workflowResource, nil)

		return "Workflow processing completed", nil
	})

	return nil
}

// processWorkflowSteps processes individual workflow steps
func (dr *DependencyResolver) processWorkflowSteps() error {
	// This would contain the actual workflow processing logic
	// For now, simulate some workflow steps

	steps := []struct {
		id   string
		name string
		work func() error
	}{
		{
			id:   "prepare-resources",
			name: "Prepare Resources",
			work: func() error {
				dr.EmitLog("info", "Preparing resources", "workflow", nil)
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		},
		{
			id:   "load-dependencies",
			name: "Load Dependencies",
			work: func() error {
				dr.EmitLog("info", "Loading dependencies", "workflow", nil)
				time.Sleep(150 * time.Millisecond)
				return nil
			},
		},
		{
			id:   "execute-workflow",
			name: "Execute Workflow",
			work: func() error {
				dr.EmitLog("info", "Executing workflow", "workflow", nil)
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
	}

	for i, step := range steps {
		progress := float64(i) / float64(len(steps))
		dr.UpdateOperationProgress("workflow-processing", progress)

		dr.StartAsyncOperation(step.id, "workflow-step", func() (interface{}, error) {
			if err := step.work(); err != nil {
				return nil, err
			}
			return fmt.Sprintf("%s completed", step.name), nil
		})
	}

	return nil
}

// HandleRunActionReactive handles run actions reactively
func (dr *DependencyResolver) HandleRunActionReactive() error {
	dr.StartAsyncOperation("run-action", "execution", func() (interface{}, error) {
		dr.EmitLog("info", "Starting run action", "execution", nil)

		// Simulate run action processing
		for i := 0; i <= 10; i++ {
			dr.UpdateOperationProgress("run-action", float64(i)/10.0)
			time.Sleep(50 * time.Millisecond)
		}

		dr.EmitLog("info", "Run action completed", "execution", nil)
		return "Run action completed successfully", nil
	})
	return nil
}

// PrepareImportFilesReactive prepares import files using reactive patterns
func (dr *DependencyResolver) PrepareImportFilesReactive() error {
	dr.StartAsyncOperation("prepare-imports", "imports", func() (interface{}, error) {
		dr.EmitLog("info", "Preparing import files", "imports", nil)

		// Simulate import preparation
		time.Sleep(100 * time.Millisecond)

		return "Import files prepared", nil
	})
	return nil
}

// CloseReactive closes the reactive resolver properly
func (dr *DependencyResolver) CloseReactive() {
	if dr.ReactiveResolver != nil {
		dr.ReactiveResolver.Close()
	}
}

// Helper function - would be moved from old resolver
func findWorkflowInRun(fs afero.Fs) string {
	// Walk through the /agents directory structure to find workflow.pkl
	agentsDir := "/agents"
	var workflowPath string

	afero.Walk(fs, agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		if !info.IsDir() && info.Name() == "workflow.pkl" {
			workflowPath = path
			return filepath.SkipDir // Found it, stop walking
		}

		return nil
	})

	return workflowPath
}
