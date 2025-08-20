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
		logger.Debug("pulling model", "model", model)

		stdout, stderr, exitCode, err := KdepsExec(
			ctx,
			"sh",
			[]string{"-c", "OLLAMA_MODELS=${OLLAMA_MODELS:-/root/.ollama} ollama pull " + model},
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

	// Copy blobs if present to /root/.ollama/models/blobs
	if _, _, _, err := KdepsExec(ctx, "test", []string{"-d", modelsSourceDir + "/blobs"}, "", false, false, logger); err == nil {
		if out, errStr, ec, errM := KdepsExec(ctx, "mkdir", []string{"-p", modelsTargetRoot + "/blobs"}, "", false, false, logger); errM != nil {
			logger.Error("failed to create ollama models/blobs", "stdout", out, "stderr", errStr, "exitCode", ec, "error", errM)
			return fmt.Errorf("failed to create ollama models/blobs: %w", errM)
		}
		if out, errStr, ec, errC := KdepsExec(ctx, "sh", []string{"-c", "cp -a /models/blobs/. /root/.ollama/models/blobs/"}, "", false, false, logger); errC != nil {
			logger.Error("failed to copy blobs", "stdout", out, "stderr", errStr, "exitCode", ec, "error", errC)
			return fmt.Errorf("failed to copy blobs: %w", errC)
		}
	}

	// Copy manifests if present to /root/.ollama/models/manifests
	if _, _, _, err := KdepsExec(ctx, "test", []string{"-d", modelsSourceDir + "/manifests"}, "", false, false, logger); err == nil {
		if out, errStr, ec, errM := KdepsExec(ctx, "mkdir", []string{"-p", modelsTargetRoot + "/manifests"}, "", false, false, logger); errM != nil {
			logger.Error("failed to create ollama models/manifests", "stdout", out, "stderr", errStr, "exitCode", ec, "error", errM)
			return fmt.Errorf("failed to create ollama models/manifests: %w", errM)
		}
		if out, errStr, ec, errC := KdepsExec(ctx, "sh", []string{"-c", "cp -a /models/manifests/. /root/.ollama/models/manifests/"}, "", false, false, logger); errC != nil {
			logger.Error("failed to copy manifests", "stdout", out, "stderr", errStr, "exitCode", ec, "error", errC)
			return fmt.Errorf("failed to copy manifests: %w", errC)
		}
	}

	// If neither blobs nor manifests were found, copy entire /models root into /root/.ollama/models
	if _, _, _, errB := KdepsExec(ctx, "test", []string{"-d", modelsSourceDir + "/blobs"}, "", false, false, logger); errB != nil {
		if _, _, _, errM := KdepsExec(ctx, "test", []string{"-d", modelsSourceDir + "/manifests"}, "", false, false, logger); errM != nil {
			stdout, stderr, exitCode, err = KdepsExec(
				ctx,
				"sh",
				[]string{"-c", "cp -a /models/. /root/.ollama/models/"},
				"",
				false,
				false,
				logger,
			)
			if err != nil {
				logger.Error("failed to copy offline models root to models/", "stdout", stdout, "stderr", stderr, "exitCode", exitCode, "error", err)
				return fmt.Errorf("failed to copy offline models root to models/: %w", err)
			}
		}
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
