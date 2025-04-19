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
	apiServerPath := filepath.Join(dr.ActionDir, "api") // fixed path

	dr.Logger.Debug("preparing workflow directory")
	if err := dr.PrepareWorkflowDir(); err != nil {
		return false, fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	host, port, err := parseOLLAMAHost(dr.Logger)
	if err != nil {
		return false, fmt.Errorf("failed to parse OLLAMA host: %w", err)
	}

	if err := startAndWaitForOllama(ctx, host, port, dr.Logger); err != nil {
		return false, fmt.Errorf("OLLAMA service startup failed: %w", err)
	}

	wfSettings := dr.Workflow.GetSettings()
	if err := pullModels(ctx, wfSettings.AgentSettings.Models, dr.Logger); err != nil {
		return wfSettings.APIServerMode || wfSettings.WebServerMode, fmt.Errorf("failed to pull models: %w", err)
	}

	if err := dr.Fs.MkdirAll(apiServerPath, 0o777); err != nil {
		return wfSettings.APIServerMode || wfSettings.WebServerMode, fmt.Errorf("failed to create API server path: %w", err)
	}

	anyMode := wfSettings.APIServerMode || wfSettings.WebServerMode
	errChan := make(chan error, 2)

	// Start API server
	if wfSettings.APIServerMode {
		go func() {
			dr.Logger.Info("starting API server")
			errChan <- startAPIServer(ctx, dr)
		}()
	}

	// Start Web server
	if wfSettings.WebServerMode {
		go func() {
			dr.Logger.Info("starting Web server")
			errChan <- startWebServer(ctx, dr)
		}()
	}

	// Wait for one to fail (or both to return nil)
	for range cap(errChan) {
		if err := <-errChan; err != nil {
			return anyMode, fmt.Errorf("server startup error: %w", err)
		}
	}

	return anyMode, nil
}

func startAndWaitForOllama(ctx context.Context, host, port string, logger *logging.Logger) error {
	go startOllamaServer(ctx, logger)
	return waitForServer(host, port, 60*time.Second, logger)
}

func pullModels(ctx context.Context, models []string, logger *logging.Logger) error {
	for _, model := range models {
		model = strings.TrimSpace(model)
		logger.Debug("pulling model", "model", model)

		stdout, stderr, exitCode, err := KdepsExec(
			ctx,
			"ollama",
			[]string{"pull", model},
			"",
			false,
			false,
			logger,
		)
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

func startWebServer(ctx context.Context, dr *resolver.DependencyResolver) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartWebServerMode(ctx, dr)
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
