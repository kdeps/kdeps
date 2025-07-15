package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/apple/pkl-go/pkl"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/spf13/afero"
)

var (
	version = "dev"
	commit  = ""

	// Function variables for dependency injection during tests.
	newGraphResolverFn        = resolver.NewGraphResolver
	bootstrapDockerSystemFn   = docker.BootstrapDockerSystem
	runGraphResolverActionsFn = runGraphResolverActions

	findConfigurationFn     = cfg.FindConfiguration
	generateConfigurationFn = cfg.GenerateConfiguration
	editConfigurationFn     = cfg.EditConfiguration
	validateConfigurationFn = cfg.ValidateConfiguration
	loadConfigurationFn     = cfg.LoadConfiguration
	getKdepsPathFn          = cfg.GetKdepsPath

	newRootCommandFn = cmd.NewRootCommand

	cleanupFn = cleanup
)

func main() {
	v.Version = version
	v.Commit = commit

	logger := logging.GetLogger()
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	// Initialize PKL evaluator with all available resource readers
	// Create temporary database paths for resource readers
	tmpDir := os.TempDir()
	memoryDBPath := filepath.Join(tmpDir, "memory.db")
	sessionDBPath := filepath.Join(tmpDir, "session.db")
	toolDBPath := filepath.Join(tmpDir, "tool.db")
	itemDBPath := filepath.Join(tmpDir, "item.db")
	pklresDBPath := filepath.Join(tmpDir, "pklres.db")

	// Initialize all resource readers
	memoryReader, err := memory.InitializeMemory(memoryDBPath)
	if err != nil {
		logger.Fatalf("failed to initialize memory reader: %v", err)
	}

	sessionReader, err := session.InitializeSession(sessionDBPath)
	if err != nil {
		logger.Fatalf("failed to initialize session reader: %v", err)
	}

	toolReader, err := tool.InitializeTool(toolDBPath)
	if err != nil {
		logger.Fatalf("failed to initialize tool reader: %v", err)
	}

	itemReader, err := item.InitializeItem(itemDBPath, []string{})
	if err != nil {
		logger.Fatalf("failed to initialize item reader: %v", err)
	}

	pklresReader, err := pklres.InitializePklResource(pklresDBPath)
	if err != nil {
		logger.Fatalf("failed to initialize pklres reader: %v", err)
	}

	// Initialize agent reader (requires additional parameters)
	agentReader, err := agent.InitializeAgent(fs, "/tmp", "default", "latest", logger)
	if err != nil {
		logger.Fatalf("failed to initialize agent reader: %v", err)
	}

	evaluatorConfig := &evaluator.EvaluatorConfig{
		ResourceReaders: []pkl.ResourceReader{
			memoryReader,
			sessionReader,
			toolReader,
			itemReader,
			agentReader,
			pklresReader,
		},
		Logger: logger,
	}
	if err := evaluator.InitializeEvaluator(ctx, evaluatorConfig); err != nil {
		logger.Fatalf("failed to initialize PKL evaluator: %v", err)
	}

	graphID := uuid.New().String()
	actionDir := filepath.Join(os.TempDir(), "action")
	agentDir := filepath.Join("/", "agent")

	// Setup environment
	env, err := setupEnvironment(fs)
	if err != nil {
		logger.Fatalf("failed to set up environment: %v", err)
	}

	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)

	if env.DockerMode == "1" {
		dr, err := newGraphResolverFn(fs, ctx, env, nil, logger.With("requestID", graphID))
		if err != nil {
			logger.Fatalf("failed to create graph resolver: %v", err)
		}

		handleDockerMode(ctx, dr, cancel)
	} else {
		handleNonDockerMode(fs, ctx, env, logger)
	}
}

func handleDockerMode(ctx context.Context, dr *resolver.DependencyResolver, cancel context.CancelFunc) {
	// Initialize Docker system
	apiServerMode, err := bootstrapDockerSystemFn(ctx, dr)
	if err != nil {
		dr.Logger.Error("error during Docker bootstrap", "error", err)
		utils.SendSigterm(dr.Logger)
		return
	}
	// Setup graceful shutdown handler
	setupSignalHandler(dr.Fs, ctx, cancel, dr.Environment, apiServerMode, dr.Logger)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		if err := runGraphResolverActionsFn(ctx, dr, apiServerMode); err != nil {
			dr.Logger.Error("error running graph resolver", "error", err)
			utils.SendSigterm(dr.Logger)
			return
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	dr.Logger.Debug("context canceled, shutting down gracefully...")
	cleanupFn(dr.Fs, ctx, dr.Environment, apiServerMode, dr.Logger)
}

func handleNonDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) {
	cfgFile, err := findConfigurationFn(fs, ctx, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = generateConfigurationFn(fs, ctx, env, logger)
		if err != nil {
			logger.Fatal("error occurred generating configuration", "error", err)
			return
		}

		logger.Info("configuration file generated", "file", cfgFile)

		cfgFile, err = editConfigurationFn(fs, ctx, env, logger)
		if err != nil {
			logger.Error("error occurred editing configuration")
		}
	}

	if cfgFile == "" {
		return
	}

	logger.Info("configuration file ready", "file", cfgFile)

	cfgFile, err = validateConfigurationFn(fs, ctx, env, logger)
	if err != nil {
		logger.Fatal("error occurred validating configuration", "error", err)
		return
	}

	systemCfg, err := loadConfigurationFn(fs, ctx, cfgFile, logger)
	if err != nil {
		logger.Error("error occurred loading configuration")
		return
	}

	kdepsDir, err := getKdepsPathFn(ctx, *systemCfg)
	if err != nil {
		logger.Error("error occurred while getting Kdeps system path")
		return
	}

	rootCmd := newRootCommandFn(fs, ctx, kdepsDir, systemCfg, env, logger)
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

// setupEnvironment initializes the environment using the filesystem.
func setupEnvironment(fs afero.Fs) (*environment.Environment, error) {
	environ, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		return nil, err
	}
	return environ, nil
}

// setupSignalHandler sets up a goroutine to handle OS signals for graceful shutdown.
func setupSignalHandler(fs afero.Fs, ctx context.Context, cancelFunc context.CancelFunc, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanupFn(fs, ctx, env, apiServerMode, logger)

		var graphID, actionDir string

		contextKeys := map[*string]ktx.ContextKey{
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

		stampFile := filepath.Join(actionDir, ".dockercleanup_"+graphID)

		if err := utils.WaitForFileReady(fs, stampFile, logger); err != nil {
			logger.Error("error occurred while waiting for file to be ready", "file", stampFile)

			return
		}
		os.Exit(0)
	}()
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolverActions(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) error {
	// Workflow directory preparation no longer needed - using project directory directly

	if err := dr.PrepareImportFiles(); err != nil {
		return fmt.Errorf("failed to prepare import files: %w", err)
	}

	// Handle run action

	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	// In certain error cases, Ollama needs to be restarted
	if fatal {
		dr.Logger.Fatal("fatal error occurred")
		utils.SendSigterm(dr.Logger)
	}

	cleanupFn(dr.Fs, ctx, dr.Environment, apiServerMode, dr.Logger)

	if err := utils.WaitForFileReady(dr.Fs, "/.dockercleanup", dr.Logger); err != nil {
		return fmt.Errorf("failed to wait for file to be ready: %w", err)
	}

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("performing cleanup tasks...")

	// Close the global agent reader
	if err := agent.CloseGlobalAgentReader(); err != nil {
		logger.Error("failed to close global agent reader", "error", err)
	}

	// Close the singleton evaluator
	if evaluatorMgr, err := evaluator.GetEvaluatorManager(); err == nil {
		if err := evaluatorMgr.Close(); err != nil {
			logger.Error("failed to close PKL evaluator", "error", err)
		}
	}

	// Remove any old cleanup flags
	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("/.dockercleanup"); err != nil {
			logger.Error("unable to delete cleanup flag file", "cleanup-file", "/.dockercleanup", "error", err)
		}
	}

	// Perform Docker cleanup
	docker.Cleanup(fs, ctx, env, logger)

	logger.Debug("cleanup complete.")

	if !apiServerMode {
		os.Exit(0)
	}
}
