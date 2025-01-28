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
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/spf13/afero"
)

var (
	version = "dev"
	commit  = ""
)

func main() {
	v.Version = version
	v.Commit = commit

	// Initialize filesystem and context
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	logger := logging.GetLogger()

	// Setup environment
	env, err := setupEnvironment(fs)
	if err != nil {
		logger.Fatalf("failed to set up environment: %v", err)
	}

	// Get the system's temporary directory (cross-platform)
	baseTempDir := os.TempDir()

	// Define the desired subdirectory for actions
	actionDir := filepath.Join(baseTempDir, "action")

	// Ensure the action directory exists (creating it if necessary)
	if err := fs.MkdirAll(actionDir, 0o777); err != nil {
		logger.Fatalf("failed to create action directory: %s", err)
	}

	if env.DockerMode == "1" {
		graphID := uuid.New().String()

		dr, err := resolver.NewGraphResolver(fs, ctx, env, "/agent", actionDir, graphID, logger)
		if err != nil {
			logger.Fatalf("failed to create graph resolver: %v", err)
		}

		handleDockerMode(fs, ctx, env, actionDir, dr, logger, cancel)
	} else {
		handleNonDockerMode(fs, ctx, env, logger)
	}
}

func handleDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, actionDir string, dr *resolver.DependencyResolver, logger *logging.Logger, cancel context.CancelFunc) {
	// Initialize Docker system
	apiServerMode, err := docker.BootstrapDockerSystem(fs, ctx, env, actionDir, dr, logger)
	if err != nil {
		logger.Error("error during Docker bootstrap", "error", err)
		utils.SendSigterm(logger)
		return
	}

	// Setup graceful shutdown handling
	setupSignalHandler(fs, ctx, cancel, env, apiServerMode, logger)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		if err := runGraphResolverActions(fs, ctx, env, dr, apiServerMode, logger); err != nil {
			logger.Error("error running graph resolver", "error", err)
			utils.SendSigterm(logger)
			return
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Debug("context canceled, shutting down gracefully...")
	cleanup(fs, ctx, env, apiServerMode, logger)
}

func handleNonDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) {
	cfgFile, err := cfg.FindConfiguration(fs, ctx, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = cfg.GenerateConfiguration(fs, ctx, env, logger)
		if err != nil {
			logger.Fatal("error occurred generating configuration", "error", err)
			return
		}

		logger.Info("configuration file generated", "file", cfgFile)

		cfgFile, err = cfg.EditConfiguration(fs, ctx, env, logger)
		if err != nil {
			logger.Error("error occurred editing configuration")
		}
	}

	if cfgFile == "" {
		return
	}

	logger.Info("configuration file ready", "file", cfgFile)

	cfgFile, err = cfg.ValidateConfiguration(fs, ctx, env, logger)
	if err != nil {
		logger.Fatal("error occurred validating configuration", "error", err)
		return
	}

	systemCfg, err := cfg.LoadConfiguration(fs, ctx, cfgFile, logger)
	if err != nil {
		logger.Error("error occurred loading configuration")
		return
	}

	kdepsDir, err := cfg.GetKdepsPath(ctx, *systemCfg)
	if err != nil {
		logger.Error("error occurred while getting Kdeps system path")
		return
	}

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)
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
		cleanup(fs, ctx, env, apiServerMode, logger)
		if err := utils.WaitForFileReady(fs, "/.dockercleanup", logger); err != nil {
			logger.Error("error occurred while waiting for file to be ready", "file", "/.dockercleanup")

			return
		}
		os.Exit(0)
	}()
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolverActions(fs afero.Fs, ctx context.Context, env *environment.Environment, dr *resolver.DependencyResolver, apiServerMode bool, logger *logging.Logger) error {
	// Prepare workflow directory
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	// Handle run action
	//nolint:contextcheck
	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	// In certain error cases, Ollama needs to be restarted
	if fatal {
		dr.Logger.Fatal("fatal error occurred")
		utils.SendSigterm(logger)
	}

	cleanup(fs, ctx, env, apiServerMode, logger)

	if err := utils.WaitForFileReady(fs, "/.dockercleanup", logger); err != nil {
		return fmt.Errorf("failed to wait for file to be ready: %w", err)
	}

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
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
