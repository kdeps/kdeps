package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/cfg"
	kdepsctx "github.com/kdeps/kdeps/pkg/core"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/spf13/afero"
)

var (
	version = "dev"
	commit  = "unknown"

	// Function variables for dependency injection during tests.
	newGraphResolverFn        func(afero.Fs, context.Context, *environment.Environment, *gin.Context, *logging.Logger, pkl.Evaluator) (*resolver.DependencyResolver, error) = resolver.NewGraphResolver
	bootstrapDockerSystemFn                                                                                                                                                 = docker.BootstrapDockerSystem
	runGraphResolverActionsFn                                                                                                                                               = runGraphResolverActions

	// Configuration functions with different signatures
	findConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error)
	generateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger, pkl.Evaluator) (string, error)
	editConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error)
	validateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger, pkl.Evaluator) (string, error)
	loadConfigurationFn     = cfg.LoadConfiguration
	getKdepsPathFn          = cfg.GetKdepsPath

	newRootCommandFn = cmd.NewRootCommand

	cleanupFn = cleanup
)

func main() {
	v.SetVersionInfo(version, commit)

	logger := logging.GetLogger()
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	// Initialize PKL evaluator with all available resource readers
	// Use in-memory databases for all readers except memory which uses persistent storage

	// Use configurable shared volume path for tests or default to appropriate location
	sharedVolumePath := os.Getenv("KDEPS_SHARED_VOLUME_PATH")
	if sharedVolumePath == "" {
		// For non-Docker mode, use a local directory instead of shared volume
		sharedVolumePath = filepath.Join(os.TempDir(), ".kdeps")
	}

	// Ensure shared volume directory exists
	if err := utils.CreateDirectories(ctx, fs, []string{sharedVolumePath}); err != nil {
		logger.Fatalf("failed to create shared volume directory: %v", err)
	}

	memoryDBPath := filepath.Join(sharedVolumePath, "memory.db") // Persistent storage (shared volume in Docker mode, temp dir in non-Docker mode)
	sessionDBPath := ":memory:"                                  // In-memory tied to operation
	toolDBPath := ":memory:"                                     // In-memory tied to operation
	itemDBPath := ":memory:"                                     // In-memory tied to operation

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

	// Initialize unified context system
	err = kdepsctx.InitializeContext(fs, "global", "", "", sharedVolumePath, logger)
	if err != nil {
		logger.Fatalf("failed to initialize unified context: %v", err)
	}

	// Get readers from unified context
	kdepsCtx := kdepsctx.GetContext()
	if kdepsCtx == nil {
		logger.Fatalf("failed to get unified context")
	}

	pklresReader := kdepsCtx.PklresReader
	agentReader := kdepsCtx.AgentReader

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
	evaluatorManager, err := evaluator.InitializeEvaluator(ctx, evaluatorConfig)
	if err != nil {
		logger.Fatalf("failed to initialize PKL evaluator: %v", err)
	}
	pklEvaluator, err := evaluatorManager.GetEvaluator()
	if err != nil {
		logger.Fatalf("failed to get PKL evaluator: %v", err)
	}
	_ = evaluatorManager // If not used directly, suppress unused warning

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

	// Initialize configuration function variables
	findConfigurationFn = cfg.FindConfiguration
	generateConfigurationFn = cfg.GenerateConfiguration
	editConfigurationFn = cfg.EditConfiguration
	validateConfigurationFn = cfg.ValidateConfiguration

	if env.DockerMode == "1" {
		// In Docker mode, check if we need to extract a .kdeps file first
		logger.Debug("Docker mode detected, checking for agent extraction", "args", os.Args)

		// Check if --agent parameter was provided by looking at command line args
		agentFile := ""
		args := os.Args[1:]
		for i, arg := range args {
			if arg == "--agent" && i+1 < len(args) {
				agentFile = args[i+1]
				break
			}
		}

		logger.Debug("Agent file parameter check", "agentFile", agentFile, "argsCount", len(args))

		if agentFile != "" {
			logger.Info("Found --agent parameter, extracting to /run", "agentFile", agentFile)

			// Verify the agent file exists before extraction
			if exists, err := afero.Exists(fs, agentFile); err != nil {
				logger.Fatalf("failed to check agent file: %v", err)
			} else if !exists {
				logger.Fatalf("agent file does not exist: %s", agentFile)
			}

			// Extract the .kdeps package to /run structure
			_, err := archiver.ExtractPackage(fs, ctx, "/", agentFile, logger)
			if err != nil {
				logger.Fatalf("failed to extract agent package: %v", err)
			}

			logger.Info("Successfully extracted agent package to /run structure")

			// Debug: Check if /run directory was created
			if exists, err := afero.Exists(fs, "/run"); err != nil {
				logger.Warn("Error checking /run directory", "error", err)
			} else if exists {
				logger.Debug("/run directory exists after extraction")
			} else {
				logger.Warn("/run directory does not exist after extraction")
			}
		} else {
			logger.Debug("No --agent parameter found, proceeding without extraction")
		}

		dr, err := newGraphResolverFn(fs, ctx, env, nil, logger.With("requestID", graphID), pklEvaluator)
		if err != nil {
			logger.Fatalf("failed to create graph resolver: %v", err)
		}

		handleDockerMode(ctx, dr, cancel)
	} else {
		handleNonDockerMode(ctx, fs, env, logger, pklEvaluator)
	}
}

func handleDockerMode(ctx context.Context, dr *resolver.DependencyResolver, cancel context.CancelFunc) {
	// Initialize Docker system
	apiServerMode, err := bootstrapDockerSystemFn(ctx, dr)
	if err != nil {
		dr.Logger.Error("error during Docker bootstrap", "error", err)
		utils.SendSigterm()
		return
	}
	// Setup graceful shutdown handler
	setupSignalHandler(ctx, dr.Fs, cancel, dr.Environment, apiServerMode, dr.Logger)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		if err := runGraphResolverActionsFn(ctx, dr, apiServerMode); err != nil {
			dr.Logger.Error("error running graph resolver", "error", err)
			utils.SendSigterm()
			return
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	dr.Logger.Debug("context canceled, shutting down gracefully...")
	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)
}

func handleNonDockerMode(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger, pklEvaluator pkl.Evaluator) {
	cfgFile, err := findConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = generateConfigurationFn(ctx, fs, env, logger, pklEvaluator)
		if err != nil {
			logger.Fatal("error occurred generating configuration", "error", err)
		}

		logger.Info("configuration file generated", "file", cfgFile)

		cfgFile, err = editConfigurationFn(ctx, fs, env, logger)
		if err != nil {
			logger.Error("error occurred editing configuration")
		}
	}

	if cfgFile == "" {
		return
	}

	logger.Info("configuration file ready", "file", cfgFile)

	cfgFile, err = validateConfigurationFn(ctx, fs, env, logger, pklEvaluator)
	if err != nil {
		logger.Fatal("error occurred validating configuration", "error", err)
		return
	}

	systemCfg, err := loadConfigurationFn(ctx, fs, cfgFile, logger)
	if err != nil {
		logger.Error("error occurred loading configuration")
		return
	}

	if systemCfg == nil {
		logger.Error("system configuration is nil")
		return
	}

	kdepsDir, err := getKdepsPathFn(ctx, *systemCfg)
	if err != nil {
		logger.Error("error occurred while getting Kdeps system path")
		return
	}

	rootCmd := newRootCommandFn(ctx, fs, kdepsDir, systemCfg, env, logger)
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
func setupSignalHandler(ctx context.Context, fs afero.Fs, cancelFunc context.CancelFunc, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanupFn(ctx, fs, env, apiServerMode, logger)

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

	// Note: Removed async pklres polling system - now running purely synchronous execution
	// The pklres system will handle dependency waiting internally during PKL template evaluation

	// Handle run action

	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	// In certain error cases, Ollama needs to be restarted
	if fatal {
		dr.Logger.Fatal("fatal error occurred")
		utils.SendSigterm()
	}

	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)

	if err := utils.WaitForFileReady(dr.Fs, "/.dockercleanup", dr.Logger); err != nil {
		return fmt.Errorf("failed to wait for file to be ready: %w", err)
	}

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(ctx context.Context, fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("performing cleanup tasks...")

	// Close the unified context
	if err := kdepsctx.CloseContext(); err != nil {
		logger.Error("failed to close unified context", "error", err)
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
