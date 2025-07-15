package docker

import (
	"context"
	"errors"
	"fmt"
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

	apiServerMode, err := SetupDockerEnvironment(ctx, dr)
	if err != nil {
		return apiServerMode, err
	}

	dr.Logger.Debug("docker system bootstrap completed.")
	return apiServerMode, nil
}

// SetupDockerEnvironment sets up the Docker environment.
func SetupDockerEnvironment(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
	apiServerPath := filepath.Join(dr.ActionDir, "api") // fixed path

	// Workflow directory preparation no longer needed - using project directory directly
	dr.Logger.Debug("using project directory directly")

	host, port, err := ParseOLLAMAHost(dr.Logger)
	if err != nil {
		return false, fmt.Errorf("failed to parse OLLAMA host: %w", err)
	}

	if err := StartAndWaitForOllama(ctx, host, port, dr.Logger); err != nil {
		return false, fmt.Errorf("OLLAMA service startup failed: %w", err)
	}

	wfSettings := dr.Workflow.GetSettings()
	if err := PullModels(ctx, wfSettings.AgentSettings.Models, dr.Logger); err != nil {
		apiServerMode := wfSettings.APIServerMode != nil && *wfSettings.APIServerMode
		webServerMode := wfSettings.WebServerMode != nil && *wfSettings.WebServerMode
		return apiServerMode || webServerMode, fmt.Errorf("failed to pull models: %w", err)
	}

	if err := dr.Fs.MkdirAll(apiServerPath, 0o777); err != nil {
		apiServerMode := wfSettings.APIServerMode != nil && *wfSettings.APIServerMode
		webServerMode := wfSettings.WebServerMode != nil && *wfSettings.WebServerMode
		return apiServerMode || webServerMode, fmt.Errorf("failed to create API server path: %w", err)
	}

	apiServerMode := wfSettings.APIServerMode != nil && *wfSettings.APIServerMode
	webServerMode := wfSettings.WebServerMode != nil && *wfSettings.WebServerMode
	anyMode := apiServerMode || webServerMode
	errChan := make(chan error, 2)

	// Start API server
	if apiServerMode {
		go func() {
			dr.Logger.Info("starting API server")
			errChan <- StartAPIServer(ctx, dr)
		}()
	}

	// Start Web server
	if webServerMode {
		go func() {
			dr.Logger.Info("starting Web server")
			errChan <- StartWebServer(ctx, dr)
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

// StartAndWaitForOllama starts and waits for Ollama to be ready.
func StartAndWaitForOllama(ctx context.Context, host, port string, logger *logging.Logger) error {
	go StartOllamaServer(ctx, logger)
	return WaitForServer(host, port, 60*time.Second, logger)
}

// PullModels pulls the required models.
func PullModels(ctx context.Context, models []string, logger *logging.Logger) error {
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

// StartAPIServer starts the API server.
func StartAPIServer(ctx context.Context, dr *resolver.DependencyResolver) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartAPIServerMode(ctx, dr)
	}()

	return <-errChan
}

// StartWebServer starts the web server.
func StartWebServer(ctx context.Context, dr *resolver.DependencyResolver) error {
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
