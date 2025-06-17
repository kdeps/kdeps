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

// TestHandleNonDockerModeGenerateFlow covers the branch where no existing
// configuration is found so the code generates, edits, validates and loads a
// new configuration. This executes the previously uncovered paths inside
// handleNonDockerMode.
func TestHandleNonDockerModeGenerateFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Back up originals.
	origFind := findConfigurationFn
	origGenerate := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGet := getKdepsPathFn
	origRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGenerate
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGet
		newRootCommandFn = origRoot
	}()

	// Stub behaviour: initial find returns empty string triggering generation.
	genPath := "/tmp/generated-config.pkl"
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", nil
	}

	generateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	editConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	dummyCfg := &schema.Kdeps{KdepsDir: ".kdeps"}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*schema.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ schema.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *schema.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		// Define a no-op RunE so that Execute() does not error.
		cmd := &cobra.Command{Use: "root"}
		cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
		return cmd
	}

	// Execute.
	handleNonDockerMode(fs, ctx, env, logger)
}
