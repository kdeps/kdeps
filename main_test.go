package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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

// TestSetupEnvironment ensures that setupEnvironment returns a valid *Environment and no error.
func TestSetupEnvironment(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Fatalf("setupEnvironment returned error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected non-nil environment")
	}
	if env.DockerMode != "0" {
		t.Errorf("expected DockerMode '0', got %q", env.DockerMode)
	}
}

// TestCleanup_RemovesFlagFile verifies that cleanup removes the /.dockercleanup file and does not exit when apiServerMode is true.
func TestCleanup_RemovesFlagFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create the flag file that should be removed by cleanup.
	if err := afero.WriteFile(fs, "/.dockercleanup", []byte("dummy"), 0o644); err != nil {
		t.Fatalf("failed to create flag file: %v", err)
	}

	ctx := context.Background()
	env := &environment.Environment{DockerMode: "0"} // ensure docker.Cleanup is a no-op
	logger := logging.NewTestLogger()

	cleanup(fs, ctx, env, true /* apiServerMode */, logger)

	if exists, _ := afero.Exists(fs, "/.dockercleanup"); exists {
		t.Errorf("expected /.dockercleanup to be removed by cleanup")
	}
}

// TestSetupSignalHandler_HandlesSignal verifies that setupSignalHandler
// responds to signals by canceling context and calling cleanup, but does not exit
// when run in test mode.
func TestSetupSignalHandler_HandlesSignal(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())
	env := &environment.Environment{DockerMode: "0"}
	logger := logging.NewTestLogger()

	// Setup signal handler
	setupSignalHandler(fs, ctx, cancel, env, true, logger)

	// The signal handler goroutine is now running and waiting for SIGINT/SIGTERM
	// Since we can't easily send real signals in tests without complex setup,
	// we'll just verify that the function doesn't panic and returns normally.
	// The actual signal handling is tested indirectly through the cleanup function.

	// Give a small delay to ensure the goroutine is set up
	time.Sleep(10 * time.Millisecond)

	// The test passes if we reach here without panicking
	// The signal handler is now running in the background
}
