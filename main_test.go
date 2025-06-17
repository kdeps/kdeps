package main

import (
	"context"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	// The following imports are required for stubbing the functions used in handleNonDockerMode
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/cobra"
)

func TestSetupEnvironment(t *testing.T) {
	// Test case 1: Basic environment setup with in-memory FS
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if env == nil {
		t.Errorf("Expected non-nil environment, got nil")
	}
	t.Log("setupEnvironment basic test passed")
}

func TestSetupEnvironmentError(t *testing.T) {
	// Test with a filesystem that will cause an error
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())

	env, err := setupEnvironment(fs)
	// The function should still return an environment even if there are minor issues
	// This depends on the actual implementation of environment.NewEnvironment
	if err != nil {
		assert.Nil(t, env)
	} else {
		assert.NotNil(t, env)
	}
}

func TestSetupSignalHandler(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Test that setupSignalHandler doesn't panic
	assert.NotPanics(t, func() {
		setupSignalHandler(fs, ctx, cancel, env, false, logger)
	})

	// Cancel the context to clean up the goroutine
	cancel()
}

func TestCleanup(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a cleanup flag file to test removal
	fs.Create("/.dockercleanup")

	// Test that cleanup doesn't panic
	assert.NotPanics(t, func() {
		cleanup(fs, ctx, env, true, logger) // Use apiServerMode=true to avoid os.Exit
	})

	// Check that the cleanup flag file was removed
	_, err := fs.Stat("/.dockercleanup")
	assert.True(t, os.IsNotExist(err))
}

// TestHandleNonDockerMode_Stubbed exercises the main.handleNonDockerMode logic using stubbed dependency
// functions so that we avoid any heavy external interactions while still executing most of the
// code paths. This substantially increases coverage for the main package.
func TestHandleNonDockerMode_Stubbed(t *testing.T) {
	// Prepare a memory backed filesystem and minimal context / environment
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "test-graph")
	env := &environment.Environment{Home: "/home", Pwd: "/pwd"}
	logger := logging.NewTestLogger()

	// Save originals to restore after the test to avoid side-effects on other tests
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

	// Stub all external dependency functions so that they succeed quickly.
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", nil // trigger configuration generation path
	}
	generateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/home/.kdeps.pkl", nil
	}
	editConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/home/.kdeps.pkl", nil
	}
	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/home/.kdeps.pkl", nil
	}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdeps.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(_ context.Context, _ kdeps.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdeps.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
	}

	// Execute the function under test – if any of our stubs return an unexpected error the
	// function itself will log.Fatal / log.Error. The absence of panics or fatal exits is our
	// success criteria here.
	handleNonDockerMode(fs, ctx, env, logger)
}

func TestHandleNonDockerMode_NoConfig(t *testing.T) {
	// Test case: No configuration file found, should not panic
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"}
	logger := logging.GetLogger()

	// Mock functions to avoid actual file operations
	originalFindConfigurationFn := findConfigurationFn
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil
	}
	defer func() { findConfigurationFn = originalFindConfigurationFn }()

	originalGenerateConfigurationFn := generateConfigurationFn
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil
	}
	defer func() { generateConfigurationFn = originalGenerateConfigurationFn }()

	// Call the function, it should return without panicking
	handleNonDockerMode(fs, ctx, env, logger)
	t.Log("handleNonDockerMode with no config test passed")
}
