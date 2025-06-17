package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	schema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerModeExistingConfig exercises the code path where a
// configuration file is found immediately (the happy-path) thereby covering
// several lines that were previously unexecuted.
func TestHandleNonDockerModeExistingConfig(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Backup originals.
	origFind := findConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGet := getKdepsPathFn
	origRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGet
		newRootCommandFn = origRoot
	}()

	// Stub functions.
	cfgPath := "/home/user/.kdeps/config.pkl"
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return cfgPath, nil
	}

	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return cfgPath, nil
	}

	dummyCfg := &schema.Kdeps{KdepsDir: ".kdeps"}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*schema.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ schema.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *schema.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// Execute.
	handleNonDockerMode(fs, ctx, env, logger)
}
