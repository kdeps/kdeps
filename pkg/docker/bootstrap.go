package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/spf13/afero"
)

// BootstrapDockerSystem initializes the Docker system and pulls models after ollama server is ready.
func BootstrapDockerSystem(fs afero.Fs, ctx context.Context, environ *environment.Environment, actionDir string, dr *resolver.DependencyResolver, logger *logging.Logger) (bool, error) {
	var apiServerMode bool

	if environ.DockerMode == "1" {
		logger.Debug("inside Docker environment")
		logger.Debug("initializing Docker system")

		agentDir := "/agent"
		apiServerPath := filepath.Join(actionDir, "/api")
		agentWorkflow := filepath.Join(agentDir, "workflow/workflow.pkl")

		exists, err := afero.Exists(fs, agentWorkflow)
		if err != nil {
			return false, err
		}

		if !exists {
			// Prepare workflow directory
			if err := dr.PrepareWorkflowDir(); err != nil {
				return false, fmt.Errorf("failed to prepare workflow directory: %w", err)
			}
		}

		wfCfg, err := workflow.LoadWorkflow(ctx, agentWorkflow, logger)
		if err != nil {
			logger.Error("error loading", "workflow", err)
			return apiServerMode, err
		}

		// Parse OLLAMA_HOST to get the host and port
		host, port, err := parseOLLAMAHost(logger)
		if err != nil {
			return apiServerMode, err
		}

		// Start ollama server in the background
		go startOllamaServer(ctx, logger)

		// Wait for ollama server to be fully ready (using the parsed host and port)
		err = waitForServer(host, port, 60*time.Second, logger)
		if err != nil {
			return apiServerMode, err
		}

		// Once ollama server is ready, proceed with pulling models
		wfSettings := wfCfg.GetSettings()
		apiServerMode = wfSettings.APIServerMode

		dockerSettings := *wfSettings.AgentSettings
		modelList := dockerSettings.Models
		for _, value := range modelList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			logger.Debug("pulling", "model", value)
			stdout, stderr, exitCode, err := KdepsExec(ctx, "ollama", []string{"pull", value}, logger)
			if err != nil {
				logger.Error("error pulling model: ", value, " stdout: ", stdout, " stderr: ", stderr, " exitCode: ", exitCode, " err: ", err)
				return apiServerMode, fmt.Errorf("error pulling model %s: %s %s %d %w", value, stdout, stderr, exitCode, err)
			}
		}

		if err := fs.MkdirAll(apiServerPath, 0o777); err != nil {
			return apiServerMode, err
		}

		errChan := make(chan error, 1) // Channel to capture the error

		go func() {
			if err := StartAPIServerMode(fs, ctx, wfCfg, environ, agentDir, apiServerPath, actionDir, dr, logger); err != nil {
				errChan <- err // Send the error to the channel
				return
			}
			errChan <- nil // Send a nil if no error occurred
		}()

		// Wait for the result from the goroutine
		err = <-errChan
		if err != nil {
			// Return the error to the caller
			return apiServerMode, err
		}
	}

	logger.Debug("docker system bootstrap completed.")

	return apiServerMode, nil
}

func CreateFlagFile(fs afero.Fs, ctx context.Context, filename string) error {
	// Check if file exists
	if exists, err := afero.Exists(fs, filename); err != nil {
		return err
	} else if !exists {
		// Create the file if it doesn't exist
		file, err := fs.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		// If the file exists, update its modification time to the current time
		currentTime := time.Now().UTC()
		if err := fs.Chtimes(filename, currentTime, currentTime); err != nil {
			return err
		}
	}
	return nil
}
