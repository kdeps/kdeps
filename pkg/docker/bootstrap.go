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
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func BootstrapDockerSystem(fs afero.Fs, ctx context.Context, environ *environment.Environment, actionDir string, dr *resolver.DependencyResolver, logger *logging.Logger) (bool, error) {
	if environ.DockerMode != "1" {
		logger.Debug("docker system bootstrap completed.")
		return false, nil
	}

	logger.Debug("inside Docker environment\ninitializing Docker system")

	apiServerMode, err := setupDockerEnvironment(fs, ctx, environ, actionDir, dr, logger)
	if err != nil {
		return apiServerMode, err
	}

	logger.Debug("docker system bootstrap completed.")
	return apiServerMode, nil
}

func setupDockerEnvironment(fs afero.Fs, ctx context.Context, environ *environment.Environment, actionDir string, dr *resolver.DependencyResolver, logger *logging.Logger) (bool, error) {
	const agentDir = "/agent"
	apiServerPath := filepath.Join(actionDir, "/api")
	agentWorkflow := filepath.Join(agentDir, "workflow/workflow.pkl")

	if err := ensureWorkflowExists(fs, dr, agentWorkflow, logger); err != nil {
		return false, err
	}

	wfCfg, err := workflow.LoadWorkflow(ctx, agentWorkflow, logger)
	if err != nil {
		logger.Error("error loading workflow", "error", err)
		return false, err
	}

	host, port, err := parseOLLAMAHost(logger)
	if err != nil {
		return false, err
	}

	if err := startAndWaitForOllama(ctx, host, port, logger); err != nil {
		return false, err
	}

	wfSettings := wfCfg.GetSettings()
	if err := pullModels(ctx, wfSettings.AgentSettings.Models, logger); err != nil {
		return wfSettings.APIServerMode, err
	}

	if err := fs.MkdirAll(apiServerPath, 0o777); err != nil {
		return wfSettings.APIServerMode, err
	}

	return wfSettings.APIServerMode, startAPIServer(fs, ctx, wfCfg, environ, agentDir, apiServerPath, actionDir, dr, logger)
}

func ensureWorkflowExists(fs afero.Fs, dr *resolver.DependencyResolver, path string, logger *logging.Logger) error {
	exists, err := afero.Exists(fs, path)
	if err != nil || exists {
		return err
	}

	logger.Debug("preparing workflow directory")
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}
	return nil
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

func startAPIServer(fs afero.Fs, ctx context.Context, wfCfg pklWf.Workflow, environ *environment.Environment, agentDir, apiServerPath, actionDir string, dr *resolver.DependencyResolver, logger *logging.Logger) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- StartAPIServerMode(fs, ctx, wfCfg, environ, agentDir, apiServerPath, actionDir, dr, logger)
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
