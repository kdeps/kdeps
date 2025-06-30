package main

import (
	"context"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaK "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"testing"
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
	findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil // trigger generation path
	}
	generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "/generated/config.yml", nil
	}
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*schemaK.Kdeps, error) {
		return &schemaK.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(afero.Fs, context.Context, string, *schemaK.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
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
	findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "/existing/config.yml", nil
	}
	validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "/existing/config.yml", nil
	}
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*schemaK.Kdeps, error) {
		return &schemaK.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(afero.Fs, context.Context, string, *schemaK.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
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
