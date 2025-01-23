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
func BootstrapDockerSystem(fs afero.Fs, ctx context.Context, environ *environment.Environment, logger *logging.Logger) (bool, error) {
	var APIServerMode bool

	if environ.DockerMode == "1" {
		logger.Debug("Inside Docker environment. Proceeding with bootstrap.")
		logger.Debug("Initializing Docker system")

		agentDir := "/agent"
		APIServerPath := filepath.Join(agentDir, "/actions/api")
		agentWorkflow := filepath.Join(agentDir, "workflow/workflow.pkl")

		exists, err := afero.Exists(fs, agentWorkflow)
		if !exists {
			env, err := environment.NewEnvironment(fs, ctx, nil)
			if err != nil {
				return false, err
			}

			dr, err := resolver.NewGraphResolver(fs, ctx, env, "/agent", logger)
			if err != nil {
				return false, fmt.Errorf("failed to create graph resolver: %w", err)
			}

			// Prepare workflow directory
			if err := dr.PrepareWorkflowDir(); err != nil {
				return false, fmt.Errorf("failed to prepare workflow directory: %w", err)
			}
		}

		wfCfg, err := workflow.LoadWorkflow(ctx, agentWorkflow, logger)
		if err != nil {
			logger.Error("Error loading", "workflow", err)
			return APIServerMode, err
		}

		// Parse OLLAMA_HOST to get the host and port
		host, port, err := parseOLLAMAHost(ctx, logger)
		if err != nil {
			return APIServerMode, err
		}

		// Start ollama server in the background
		if err := startOllamaServer(logger); err != nil {
			return APIServerMode, fmt.Errorf("Failed to start ollama server: %w", err)
		}

		// Wait for ollama server to be fully ready (using the parsed host and port)
		err = waitForServer(host, port, 60*time.Second, logger)
		if err != nil {
			return APIServerMode, err
		}

		// Once ollama server is ready, proceed with pulling models
		wfSettings := wfCfg.GetSettings()
		APIServerMode = wfSettings.APIServerMode

		dockerSettings := *wfSettings.AgentSettings
		modelList := dockerSettings.Models
		for _, value := range modelList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			logger.Debug("Pulling", "model", value)
			stdout, stderr, exitCode, err := KdepsExec("ollama", []string{"pull", value}, logger)
			if err != nil {
				logger.Error("Error pulling model: ", value, " stdout: ", stdout, " stderr: ", stderr, " exitCode: ", exitCode, " err: ", err)
				return APIServerMode, fmt.Errorf("Error pulling model %s: %s %s %d %w", value, stdout, stderr, exitCode, err)
			}
		}

		if err := fs.MkdirAll(APIServerPath, 0o777); err != nil {
			return true, err
		}

		go func() error {
			if err := StartAPIServerMode(fs, ctx, wfCfg, environ, agentDir, APIServerPath, logger); err != nil {
				return err
			}

			return nil
		}()
	}

	logger.Debug("Docker system bootstrap completed.")

	return APIServerMode, nil
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
		currentTime := time.Now().Local()
		if err := fs.Chtimes(filename, currentTime, currentTime); err != nil {
			return err
		}
	}
	return nil
}
