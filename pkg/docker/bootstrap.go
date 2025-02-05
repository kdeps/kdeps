package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

func BootstrapDockerSystem(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
	if dr.Environment.DockerMode != "1" {
		dr.Logger.Debug("docker system bootstrap completed.")
		return false, nil
	}

	dr.Logger.Debug("inside Docker environment\ninitializing Docker system")

	apiServerMode, err := setupDockerEnvironment(ctx, dr)
	if err != nil {
		return apiServerMode, err
	}

	dr.Logger.Debug("docker system bootstrap completed.")
	return apiServerMode, nil
}

func setupDockerEnvironment(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
	apiServerPath := filepath.Join(dr.ActionDir, "/api/")

	dr.Logger.Debug("preparing workflow directory")
	if err := dr.PrepareWorkflowDir(); err != nil {
		return false, fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	host, port, err := parseOLLAMAHost(dr.Logger)
	if err != nil {
		return false, err
	}

	if err := startAndWaitForOllama(ctx, host, port, dr.Logger); err != nil {
		return false, err
	}

	wfSettings := dr.Workflow.GetSettings()
	if err := pullModels(ctx, wfSettings.AgentSettings.Models, dr.Logger); err != nil {
		return wfSettings.APIServerMode, err
	}

	if err := dr.Fs.MkdirAll(apiServerPath, 0o777); err != nil {
		return wfSettings.APIServerMode, err
	}

	return wfSettings.APIServerMode, startAPIServer(ctx, dr)
}

func startAndWaitForOllama(ctx context.Context, host, port string, logger *logging.Logger) error {
	go startOllamaServer(ctx, logger)
	return waitForServer(host, port, 60*time.Second, logger)
}

func pullModels(ctx context.Context, models []string, logger *logging.Logger) error {
	for _, model := range models {
		model = strings.TrimSpace(model)
		logger.Debug("pulling model", "model", model)

		stdout, stderr, exitCode, err := KdepsExec(ctx, "ollama", []string{"pull", model}, logger)
		if err != nil {
			logger.Error("model pull failed", "model", model, "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
			return fmt.Errorf("failed to pull model %s: %w", model, err)
		}
	}
	return nil
}

func startAPIServer(ctx context.Context, dr *resolver.DependencyResolver) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartAPIServerMode(ctx, dr)
	}()

	return <-errChan
}

func CreateFlagFile(fs afero.Fs, ctx context.Context, filename string) error {
	if exists, err := afero.Exists(fs, filename); err != nil || exists {
		return err
	}

	file, err := fs.Create(filename)
	if err != nil {
		return err
	}
	file.Close()

	currentTime := time.Now().UTC()
	return fs.Chtimes(filename, currentTime, currentTime)
}
