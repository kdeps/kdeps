package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	kdeps "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerMode_GenerateFlow exercises the path where no config exists and it must be generated.
func TestHandleNonDockerMode_GenerateFlow(t *testing.T) {
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

	// Stubbed behaviours
	findConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil // trigger generation path
	}
	generateConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	editConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	validateConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	loadConfigurationFn = func(context.Context, afero.Fs, string, *logging.Logger) (*kdeps.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdeps.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(context.Context, afero.Fs, string, *kdeps.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
		return &cobra.Command{
			Use: "root",
			Run: func(cmd *cobra.Command, args []string) {},
		}
	}

	// Call the function; expecting graceful completion without panic.
	handleNonDockerMode(fs, ctx, env, logger)
}

// TestHandleNonDockerMode_ExistingConfig exercises the flow when a configuration already exists.
func TestHandleNonDockerMode_ExistingConfig(t *testing.T) {
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
	findConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "/existing/config.yml", nil
	}
	validateConfigurationFn = func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) {
		return "/existing/config.yml", nil
	}
	loadConfigurationFn = func(context.Context, afero.Fs, string, *logging.Logger) (*kdeps.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdeps.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(context.Context, afero.Fs, string, *kdeps.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// Execute
	handleNonDockerMode(fs, ctx, env, logger)
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
