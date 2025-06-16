package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	schema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestHandleNonDockerModeFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// backup original function vars and restore after test
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

	// stub behaviours
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", nil // ensure we go through generation path
	}

	genPath := "/tmp/system.pkl"
	generateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	editConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return genPath, nil
	}

	dummyCfg := &schema.Kdeps{}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*schema.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ schema.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *schema.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// execute function
	handleNonDockerMode(fs, ctx, env, logger)

	// if we reach here, function executed without fatal panic.
	assert.True(t, true)
}
