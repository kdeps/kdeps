package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	kdSchema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerModeMinimal exercises the happy path of handleNonDockerMode
// using stubbed helpers. It ensures the internal control flow executes without
// touching the real filesystem or starting Docker.
func TestHandleNonDockerModeMinimal(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"}
	logger := logging.NewTestLogger()

	// ---- stub helper fns ----
	findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil // trigger generation path
	}
	generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return tmp + "/cfg.pkl", nil
	}
	editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return tmp + "/cfg.pkl", nil
	}
	validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return tmp + "/cfg.pkl", nil
	}
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdSchema.Kdeps, error) {
		return &kdSchema.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdSchema.Kdeps) (string, error) { return tmp, nil }

	newRootCommandFn = func(afero.Fs, context.Context, string, *kdSchema.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
		c := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
		return c
	}

	// execute function under test; should not panic
	handleNonDockerMode(fs, ctx, env, logger)
}
