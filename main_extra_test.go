package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestRunGraphResolverActions_PrepareWorkflowDirError verifies that an error in
// PrepareWorkflowDir is propagated by runGraphResolverActions. This provides
// coverage over the early-exit failure path without bootstrapping a full
// resolver workflow.
func TestRunGraphResolverActions_PrepareWorkflowDirError(t *testing.T) {
	t.Parallel()

	// Use an in-memory filesystem with *no* project directory so that
	// PrepareWorkflowDir fails when walking the source path.
	fs := afero.NewMemMapFs()

	env := &environment.Environment{DockerMode: "0"}
	logger := logging.NewTestLogger()

	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		ProjectDir:  "/nonexistent/project", // source dir intentionally missing
		WorkflowDir: "/tmp/workflow",
		Environment: env,
		Context:     context.Background(),
	}

	err := runGraphResolverActions(dr.Context, dr, false)
	if err == nil {
		t.Fatal("expected error due to missing project directory, got nil")
	}
}

// TestHandleNonDockerModeExercise exercises the happy-path configuration flow using stubbed functions.
func TestHandleNonDockerModeExercise(t *testing.T) {
	// Save original function pointers to restore after test
	origFind := findConfigurationFn
	origGen := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn
	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGen
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	}()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"}
	logger := logging.NewTestLogger()

	// Stub behaviour chain
	findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil // trigger generation path
	}
	generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "config.yml", nil
	}
	editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "config.yml", nil
	}
	validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "config.yml", nil
	}
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdeps.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdeps.Kdeps) (string, error) {
		return "/tmp/kdeps", nil
	}

	executed := false
	newRootCommandFn = func(afero.Fs, context.Context, string, *kdeps.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
		return &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { executed = true; return nil }}
	}

	handleNonDockerMode(fs, ctx, env, logger)
	require.True(t, executed, "root command Execute should be called")
}

// TestCleanupFlagRemoval verifies cleanup deletes the /.dockercleanup flag file.
func TestCleanupFlagRemoval(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"} // skip docker specific logic
	logger := logging.NewTestLogger()

	// Create flag file
	require.NoError(t, afero.WriteFile(fs, "/.dockercleanup", []byte("flag"), 0644))

	cleanup(fs, ctx, env, true, logger)

	exists, _ := afero.Exists(fs, "/.dockercleanup")
	require.False(t, exists, "cleanup should remove /.dockercleanup")
}
