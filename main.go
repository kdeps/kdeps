package main

import (
	"context"
	"fmt"
	"kdeps/cmd"
	"kdeps/pkg/cfg"
	"kdeps/pkg/docker"
	"kdeps/pkg/environment"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"kdeps/pkg/utils"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func main() {
	// Initialize filesystem and context
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	logger := logging.GetLogger()

	// Setup environment
	env, err := setupEnvironment(fs)
	if err != nil {
		logger.Error("Failed to set up environment", "error", err)
		os.Exit(1)
	}

	if env.DockerMode == "1" {
		// Initialize Docker system
		apiServerMode, err := docker.BootstrapDockerSystem(fs, ctx, env, logger)
		if err != nil {
			logger.Error("Error during Docker bootstrap", "error", err)
			utils.SendSigterm(logger)
		}

		// Setup graceful shutdown handling
		setupSignalHandler(cancel, fs, env, apiServerMode, logger)

		// Run workflow or wait for shutdown
		if !apiServerMode {
			err = runGraphResolver(fs, ctx, env, apiServerMode, logger)
			if err != nil {
				logger.Error("Error running graph resolver", "error", err)
				utils.SendSigterm(logger)
			}
		}

		// Wait for shutdown signal
		<-ctx.Done()
		logger.Debug("Context canceled, shutting down gracefully...")
		cleanup(fs, env, apiServerMode, logger)
	} else {
		var cfgFile string

		cfgFile, err = cfg.FindConfiguration(fs, env, logger)
		if err != nil {
			logger.Error("Error occurred finding configuration")
		}

		if cfgFile == "" {
			cfgFile, err = cfg.GenerateConfiguration(fs, env, logger)
			if err != nil {
				logger.Fatal("Error occurred generating configuration", "error", err)
				os.Exit(1)
			}

			cfgFile, err = cfg.EditConfiguration(fs, env, logger)
			if err != nil {
				logger.Error("Error occurred editing configuration")
			}
		}

		cfgFile, err = cfg.ValidateConfiguration(fs, env, logger)
		if err != nil {
			logger.Fatal("Error occurred validating configuration", "error", err)
			os.Exit(1)
		}

		systemCfg, err := cfg.LoadConfiguration(fs, cfgFile, logger)
		if err != nil {
			logger.Error("Error occurred loading configuration")
			os.Exit(1)
		}

		kdepsDir, err := cfg.GetKdepsPath(*systemCfg)
		if err != nil {
			logger.Error("Error occurred while getting Kdeps system path")
			os.Exit(1)
		}

		rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

		if err := rootCmd.Execute(); err != nil {
			log.Fatal(err)
		}
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
func setupSignalHandler(cancelFunc context.CancelFunc, fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *log.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanup(fs, env, apiServerMode, logger)
		utils.WaitForFileReady(fs, "/.dockercleanup", logger)
		os.Exit(0)
	}()
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool, logger *log.Logger) error {
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
func cleanup(fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *log.Logger) {
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
