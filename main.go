package main

import (
	"context"
	"fmt"
	"kdeps/pkg/docker"
	"kdeps/pkg/environment"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/afero"
)

func main() {
	// Initialize filesystem and context
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	// Setup environment
	env, err := setupEnvironment(fs)
	if err != nil {
		logging.Error("Failed to set up environment", "error", err)
		os.Exit(1)
	}

	// Initialize Docker system
	apiServerMode, err := docker.BootstrapDockerSystem(fs, ctx, env)
	if err != nil {
		logging.Error("Error during Docker bootstrap", "error", err)
		os.Exit(1)
	}

	// Setup graceful shutdown handling
	setupSignalHandler(cancel, fs, env, apiServerMode)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		err = runGraphResolver(fs, ctx, env, apiServerMode)
		if err != nil {
			logging.Error("Error running graph resolver", "error", err)
			os.Exit(1)
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logging.Info("Context canceled, shutting down gracefully...")
	cleanup(fs, env, apiServerMode)
}

// setupEnvironment initializes the environment using the filesystem.
func setupEnvironment(fs afero.Fs) (*environment.Environment, error) {
	env := &environment.Environment{}
	environ, err := environment.NewEnvironment(fs, env)
	if err != nil {
		return nil, err
	}
	return environ, nil
}

// setupSignalHandler sets up a goroutine to handle OS signals for graceful shutdown.
func setupSignalHandler(cancelFunc context.CancelFunc, fs afero.Fs, env *environment.Environment, apiServerMode bool) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logging.Info(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanup(fs, env, apiServerMode)
		resolver.WaitForFile(fs, "/.dockercleanup")
		os.Exit(0)
	}()
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolver(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool) error {
	dr, err := resolver.NewGraphResolver(fs, nil, ctx, env, "/agent")
	if err != nil {
		return fmt.Errorf("failed to create graph resolver: %w", err)
	}

	// Prepare workflow directory
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	// Handle run action
	if err := dr.HandleRunAction(); err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	cleanup(fs, env, apiServerMode)

	resolver.WaitForFile(fs, "/.dockercleanup")

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(fs afero.Fs, env *environment.Environment, apiServerMode bool) {
	logging.Info("Performing cleanup tasks...")

	// Remove any old cleanup flags
	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("/.dockercleanup"); err != nil {
			logging.Error("Unable to delete cleanup flag file", "cleanup-file", "/.dockercleanup", "error", err)
		}
	}

	// Perform Docker cleanup
	docker.Cleanup(fs, env)

	logging.Info("Cleanup complete.")

	if !apiServerMode {
		os.Exit(0)
	}
}
