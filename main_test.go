package main

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
)

// TestHandleNonDockerMode_Smoke verifies that the non-docker CLI path
// executes end-to-end without panicking or exiting the process. It
// stubs all dependency-injected functions so no real file system or
// external interaction occurs.
func TestHandleNonDockerMode_Smoke(t *testing.T) {
	// Preserve originals so we can restore them when the test ends.
	origFindCfg := findConfigurationFn
	origGenCfg := generateConfigurationFn
	origEditCfg := editConfigurationFn
	origValidateCfg := validateConfigurationFn
	origLoadCfg := loadConfigurationFn
	origGetKdeps := getKdepsPathFn
	origNewRootCmd := newRootCommandFn
	defer func() {
		findConfigurationFn = origFindCfg
		generateConfigurationFn = origGenCfg
		editConfigurationFn = origEditCfg
		validateConfigurationFn = origValidateCfg
		loadConfigurationFn = origLoadCfg
		getKdepsPathFn = origGetKdeps
		newRootCommandFn = origNewRootCmd
	}()

	// Stub implementations – each simply records it was invoked and
	// returns a benign value.
	var loadCalled, rootCalled int32
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "config.yaml", nil
	}
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "generated.yaml", nil
	}
	editConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "edited.yaml", nil
	}
	validateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "validated.yaml", nil
	}
	loadConfigurationFn = func(fs afero.Fs, ctx context.Context, cfgFile string, logger *logging.Logger) (*kdeps.Kdeps, error) {
		atomic.AddInt32(&loadCalled, 1)
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(ctx context.Context, cfg kdeps.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, cfg *kdeps.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		atomic.AddInt32(&rootCalled, 1)
		return &cobra.Command{}
	}

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"}
	logger := logging.NewTestLogger()

	handleNonDockerMode(fs, ctx, env, logger)

	if atomic.LoadInt32(&loadCalled) == 0 {
		t.Errorf("expected loadConfigurationFn to be called")
	}
	if atomic.LoadInt32(&rootCalled) == 0 {
		t.Errorf("expected newRootCommandFn to be called")
	}
}
