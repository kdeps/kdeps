package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""

	// Function variables for dependency injection during tests.
	newGraphResolverFn        = resolver.NewGraphResolver
	bootstrapDockerSystemFn   = docker.BootstrapDockerSystem
	runGraphResolverActionsFn = runGraphResolverActions

	findConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.FindConfiguration
	generateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.GenerateConfiguration
	editConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.EditConfiguration
	validateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.ValidateConfiguration
	loadConfigurationFn     func(context.Context, afero.Fs, string, *logging.Logger) (*kdeps.Kdeps, error)             = cfg.LoadConfiguration
	getKdepsPathFn          func(context.Context, kdeps.Kdeps) (string, error)                                         = cfg.GetKdepsPath

	newRootCommandFn func(context.Context, afero.Fs, string, *kdeps.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command = cmd.NewRootCommand

	cleanupFn = cleanup
)

func main() {
	v.Version = version
	v.Commit = commit

	logger := logging.GetLogger()
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

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
		handleNonDockerMode(ctx, fs, env, logger)
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
	setupSignalHandler(ctx, dr.Fs, cancel, dr.Environment, apiServerMode, dr.Logger)

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
	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)
}

func handleNonDockerMode(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) {
	cfgFile, err := findConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = generateConfigurationFn(ctx, fs, env, logger)
		if err != nil {
			logger.Fatal("error occurred generating configuration", "error", err)
			return
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

	cfgFile, err = validateConfigurationFn(ctx, fs, env, logger)
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
	// Prepare workflow directory
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

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

	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)

	if err := utils.WaitForFileReady(dr.Fs, "/.dockercleanup", dr.Logger); err != nil {
		return fmt.Errorf("failed to wait for file to be ready: %w", err)
	}

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(ctx context.Context, fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("performing cleanup tasks...")

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
