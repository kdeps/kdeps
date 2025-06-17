package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestHandleNonDockerModeEditError triggers the branch where editing the
// generated configuration fails, exercising the previously uncovered
// logger.Error path and early return when cfgFile remains empty.
func TestHandleNonDockerModeEditError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// backup originals
	origFind := findConfigurationFn
	origGenerate := generateConfigurationFn
	origEdit := editConfigurationFn

	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGenerate
		editConfigurationFn = origEdit
	}()

	// No existing config
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", nil
	}
	// Generation succeeds
	generated := "/tmp/generated.pkl"
	generateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return generated, nil
	}
	// Editing fails
	editConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", fmt.Errorf("edit failed")
	}

	// Other functions should not be called; keep minimal safe stubs.
	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		t.Fatalf("validateConfigurationFn should not be called when cfgFile is empty after edit")
		return "", nil
	}

	// Execute – should not panic or fatal.
	handleNonDockerMode(fs, ctx, env, logger)
}
