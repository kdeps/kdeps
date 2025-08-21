package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

func BootstrapDockerSystem(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
	if dr.Logger == nil {
		return false, errors.New("Bootstrapping Docker system failed")
	}

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

	// Handle model initialization based on offline mode
	if wfSettings.AgentSettings.OfflineMode {
		if err := copyOfflineModels(ctx, wfSettings.AgentSettings.Models, dr.Logger); err != nil {
			return wfSettings.APIServerMode || wfSettings.WebServerMode, fmt.Errorf("failed to copy offline models: %w", err)
		}
	} else {
		if err := pullModels(ctx, wfSettings.AgentSettings.Models, dr.Logger); err != nil {
			return wfSettings.APIServerMode || wfSettings.WebServerMode, fmt.Errorf("failed to pull models: %w", err)
		}
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
		if model == "" {
			continue
		}
		logger.Debug("pulling model", "model", model)

		// Apply a per-model timeout so we don't hang indefinitely when offline
		tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		stdout, stderr, exitCode, err := KdepsExec(
			tctx,
			"sh",
			[]string{"-c", "OLLAMA_MODELS=${OLLAMA_MODELS:-/root/.ollama} ollama pull " + model},
			"",
			false,
			false,
			logger,
		)
		cancel()

		if err != nil || exitCode != 0 {
			// Check if this is likely a "binary not found" error vs network/registry issues
			if strings.Contains(stderr, "command not found") || strings.Contains(stderr, "not found") || 
			   strings.Contains(stdout, "could not find ollama app") ||
			   strings.Contains(stderr, "could not find ollama app") ||
			   strings.Contains(err.Error(), "executable file not found") {
				return fmt.Errorf("ollama binary not found: %w", err)
			}
			// For other errors (network, registry unavailable, etc.), warn and continue
			logger.Warn("model pull skipped or failed (continuing)", "model", model, "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
			continue
		}
	}
	return nil
}

func copyOfflineModels(ctx context.Context, models []string, logger *logging.Logger) error {
	// Copy models from build-time location to ollama's shared volume location
	modelsSourceDir := "/models"
	modelsTargetDir := os.Getenv("OLLAMA_MODELS")
	if strings.TrimSpace(modelsTargetDir) == "" {
		modelsTargetDir = "/root/.ollama"
	}
	modelsTargetRoot := modelsTargetDir + "/models"

	// Check if source models directory exists
	stdout, stderr, exitCode, err := KdepsExec(
		ctx,
		"test",
		[]string{"-d", modelsSourceDir},
		"",
		false,
		false,
		logger,
	)
	if err != nil {
		logger.Warn("offline models directory not found, skipping offline model setup", "path", modelsSourceDir)
		return nil
	}

	// Create target root directory if it doesn't exist
	stdout, stderr, exitCode, err = KdepsExec(
		ctx,
		"mkdir",
		[]string{"-p", modelsTargetRoot},
		"",
		false,
		false,
		logger,
	)
	if err != nil {
		logger.Error("failed to create ollama models root directory", "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
		return fmt.Errorf("failed to create ollama models root directory: %w", err)
	}

	// Sync /models into ${OLLAMA_MODELS}/models using rsync (preserves attrs, handles dots, shows progress)
	cmd := fmt.Sprintf("mkdir -p %s && rsync -avrPtz --human-readable %s/. %s/", modelsTargetRoot, modelsSourceDir, modelsTargetRoot)
	stdout, stderr, exitCode, err = KdepsExec(ctx, "sh", []string{"-c", cmd}, "", false, false, logger)
	if err != nil {
		logger.Error("failed to sync offline models via rsync", "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
		return fmt.Errorf("failed to sync offline models via rsync: %w", err)
	}
	if err != nil {
		logger.Error("failed to copy offline models", "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
		return fmt.Errorf("failed to copy offline models: %w", err)
	}

	logger.Info("offline models copied successfully", "source", modelsSourceDir, "target", modelsTargetDir)
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
