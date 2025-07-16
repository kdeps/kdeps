package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaK "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerMode_GenerateFlow exercises the path where no config exists and it must be generated.
func TestHandleNonDockerMode_GenerateFlow(_ *testing.T) {
	// Prepare filesystem and env
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env, _ := environment.NewEnvironment(fs, nil)
	logger := logging.GetLogger()

	// Backup original fns
	origFind := findConfigurationFn
	origGenerate := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGenerate
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	}()

	// Mock functions with correct parameter order
	findConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	generateConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	editConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	validateConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	loadConfigurationFn = func(_ context.Context, _ afero.Fs, _ string, _ *logging.Logger) (*schemaK.Kdeps, error) {
		return &schemaK.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ context.Context, _ afero.Fs, _ string, systemCfg *schemaK.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(_ *cobra.Command, _ []string) {}}
	}

	// Call the function; expecting graceful completion without panic.
	handleNonDockerMode(ctx, fs, env, logger)
}

// TestHandleNonDockerMode_ExistingConfig exercises the flow when a configuration already exists.
func TestHandleNonDockerMode_ExistingConfig(_ *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env, _ := environment.NewEnvironment(fs, nil)
	logger := logging.GetLogger()

	// Backup originals
	origFind := findConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	}()

	// Stubs
	findConfigurationFn = func(_ context.Context, _ afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil
	}
	generateConfigurationFn = func(_ context.Context, _ afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "/test/existing.pkl", nil
	}
	validateConfigurationFn = func(_ context.Context, _ afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "/existing/config.yml", nil
	}
	loadConfigurationFn = func(_ context.Context, _ afero.Fs, _ string, logger *logging.Logger) (*schemaK.Kdeps, error) {
		return &schemaK.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ context.Context, _ afero.Fs, _ string, systemCfg *schemaK.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// Execute
	handleNonDockerMode(ctx, fs, env, logger)
}

func TestSetupEnvironmentSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected non-nil environment")
	}
}
