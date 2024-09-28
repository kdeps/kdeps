package main

import (
	"context"
	"fmt"
	"kdeps/pkg/docker"
	"kdeps/pkg/environment"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/afero"
)

func main() {
	// Create an afero filesystem (you can use afero.NewOsFs() for the real filesystem)
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	env := &environment.Environment{}
	environ, err := environment.NewEnvironment(fs, env)
	if err != nil {
		logging.Error(err)

		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logging.Info(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancel()

		cleanup(fs, environ)

		resolver.WaitForFile(fs, "/.dockercleanup")

		os.Exit(0)
	}()

	// Call BootstrapDockerSystem to initialize Docker and pull models
	apiServerMode, err := docker.BootstrapDockerSystem(fs, ctx, environ)
	if err != nil {
		fmt.Printf("Error during bootstrap: %v\n", err)
		os.Exit(1) // Exit with a non-zero status on failure
	}

	if !apiServerMode {
		dr, err := resolver.NewGraphResolver(fs, nil, ctx, environ, "/agent")
		if err != nil {
			log.Fatal(err)
		}

		if err := dr.PrepareWorkflowDir(); err != nil {
			log.Fatal(err)
		}

		if err := dr.HandleRunAction(); err != nil {
			log.Fatal(err)
		}

		cleanup(fs, environ)
		resolver.WaitForFile(fs, "/.dockercleanup")
		os.Exit(0)
	}

	// Block the main routine, but respond to the context cancellation
	<-ctx.Done()
	logging.Info("Shutting down gracefully...")
}

// cleanup performs any necessary cleanup tasks before shutting down
func cleanup(fs afero.Fs, environ *environment.Environment) {
	// Perform file cleanups or other shutdown tasks here
	logging.Info("Performing cleanup tasks...")

	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("./dockercleanup"); err != nil {
			logging.Error("Unable to delete old cleanup flag file", "cleanup-file", "/.dockercleanup")
		}
	}

	docker.Cleanup(fs, environ)

	logging.Info("Cleanup complete.")
}
