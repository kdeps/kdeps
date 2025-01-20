package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	env, err := setupEnvironment(fs, ctx)
	if err != nil {
		logger.Error("Failed to set up environment", "error", err)
		return
	}

	if env.DockerMode == "1" {
		handleDockerMode(fs, ctx, env, logger, cancel)
	} else {
		handleNonDockerMode(fs, ctx, env, logger)
	}
}

func handleDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger, cancel context.CancelFunc) {
	// Initialize Docker system
	apiServerMode, err := docker.BootstrapDockerSystem(fs, ctx, env, logger)
	if err != nil {
		logger.Error("Error during Docker bootstrap", "error", err)
		utils.SendSigterm(logger)
		return
	}

	// Setup graceful shutdown handling
	setupSignalHandler(cancel, fs, env, apiServerMode, logger)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		if err := runGraphResolver(fs, ctx, env, apiServerMode, logger); err != nil {
			logger.Error("Error running graph resolver", "error", err)
			utils.SendSigterm(logger)
			return
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Debug("Context canceled, shutting down gracefully...")
	cleanup(fs, env, apiServerMode, logger)
}

func handleNonDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) {
	cfgFile, err := cfg.FindConfiguration(fs, ctx, env, logger)
	if err != nil {
		logger.Error("Error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = cfg.GenerateConfiguration(fs, ctx, env, logger)
		if err != nil {
			logger.Fatal("Error occurred generating configuration", "error", err)
			return
		}

		logger.Info("Configuration file generated", "file", cfgFile)

		cfgFile, err = cfg.EditConfiguration(fs, ctx, env, logger)
		if err != nil {
			logger.Error("Error occurred editing configuration")
		}
	}

	if cfgFile == "" {
		return
	}

	logger.Info("Configuration file ready", "file", cfgFile)

	cfgFile, err = cfg.ValidateConfiguration(fs, ctx, env, logger)
	if err != nil {
		logger.Fatal("Error occurred validating configuration", "error", err)
		return
	}

	systemCfg, err := cfg.LoadConfiguration(fs, ctx, cfgFile, logger)
	if err != nil {
		logger.Error("Error occurred loading configuration")
		return
	}

	kdepsDir, err := cfg.GetKdepsPath(ctx, *systemCfg)
	if err != nil {
		logger.Error("Error occurred while getting Kdeps system path")
		return
	}

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

// setupEnvironment initializes the environment using the filesystem.
func setupEnvironment(fs afero.Fs, ctx context.Context) (*environment.Environment, error) {
	environ, err := environment.NewEnvironment(fs, ctx, nil)
	if err != nil {
		return nil, err
	}
	return environ, nil
}

// setupSignalHandler sets up a goroutine to handle OS signals for graceful shutdown.
func setupSignalHandler(cancelFunc context.CancelFunc, fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanup(fs, env, apiServerMode, logger)
		if err := utils.WaitForFileReady(fs, "/.dockercleanup", logger); err != nil {
			logger.Error("Error occurred while waiting for file to be ready", "file", "/.dockercleanup")

			return
		}
		os.Exit(0)
	}()
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool, logger *logging.Logger) error {
	dr, err := resolver.NewGraphResolver(fs, logger, ctx, env, "/agent")
	if err != nil {
		return fmt.Errorf("failed to create graph resolver: %w", err)
	}

	// Prepare workflow directory
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	// Handle run action
	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	// In certain error cases, Ollama needs to be restarted
	if fatal {
		dr.Logger.Fatal("Fatal error occurred")
		utils.SendSigterm(logger)
	}

	cleanup(fs, env, apiServerMode, logger)

	utils.WaitForFileReady(fs, "/.dockercleanup", logger)

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("Performing cleanup tasks...")

	// Remove any old cleanup flags
	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("/.dockercleanup"); err != nil {
			logger.Error("Unable to delete cleanup flag file", "cleanup-file", "/.dockercleanup", "error", err)
		}
	}

	// Perform Docker cleanup
	docker.Cleanup(fs, env, logger)

	logger.Debug("Cleanup complete.")

	if !apiServerMode {
		os.Exit(0)
	}
}
