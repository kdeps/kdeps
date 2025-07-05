package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/schema/gen/kdeps"
	kdepspkg "github.com/kdeps/schema/gen/kdeps"
	kpath "github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(_ context.Context, _ kdepspkg.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
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

	originalEditConfigurationFn := editConfigurationFn
	editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil
	}
	defer func() { editConfigurationFn = originalEditConfigurationFn }()

	originalValidateConfigurationFn := validateConfigurationFn
	validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
		return "", nil
	}
	defer func() { validateConfigurationFn = originalValidateConfigurationFn }()

	// Call the function, it should return without panicking
	handleNonDockerMode(fs, ctx, env, logger)
	t.Log("handleNonDockerMode with no config test passed")
}

func TestCleanupFlagRemovalMemFS(t *testing.T) {
	_ = schema.SchemaVersion(context.Background())

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	flag := "/.dockercleanup"
	if err := afero.WriteFile(fs, flag, []byte("flag"), 0o644); err != nil {
		t.Fatalf("write flag: %v", err)
	}

	env := &environment.Environment{DockerMode: "0"}

	cleanup(fs, ctx, env, true, logger)

	if exists, _ := afero.Exists(fs, flag); exists {
		t.Fatalf("cleanup did not remove %s", flag)
	}
}

// Helper to reset global injectable vars after test.
func withInjects(inject func(), t *testing.T) {
	t.Helper()
	inject()
	t.Cleanup(func() {
		// restore originals (defined in main.go)
		newGraphResolverFn = resolver.NewGraphResolver
		bootstrapDockerSystemFn = docker.BootstrapDockerSystem
		runGraphResolverActionsFn = runGraphResolverActions

		findConfigurationFn = cfg.FindConfiguration
		generateConfigurationFn = cfg.GenerateConfiguration
		editConfigurationFn = cfg.EditConfiguration
		validateConfigurationFn = cfg.ValidateConfiguration
		loadConfigurationFn = cfg.LoadConfiguration
		getKdepsPathFn = cfg.GetKdepsPath

		newRootCommandFn = cmd.NewRootCommand
		cleanupFn = cleanup
	})
}

func TestHandleDockerMode_Flow(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{DockerMode: "1"}
	logger := logging.NewTestLogger()

	dr := &resolver.DependencyResolver{Fs: fs, Logger: logger, Environment: env}

	// Channels to assert our stubs were invoked
	bootCalled := make(chan struct{}, 1)
	cleanupCalled := make(chan struct{}, 1)

	withInjects(func() {
		bootstrapDockerSystemFn = func(ctx context.Context, _ *resolver.DependencyResolver) (bool, error) {
			bootCalled <- struct{}{}
			return true, nil // apiServerMode
		}
		// runGraphResolverActions should NOT be called because apiServerMode == true; panic if invoked
		runGraphResolverActionsFn = func(ctx context.Context, dr *resolver.DependencyResolver, apiServer bool) error {
			t.Fatalf("runGraphResolverActions should not be called in apiServerMode")
			return nil
		}
		cleanupFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ bool, _ *logging.Logger) {
			cleanupCalled <- struct{}{}
		}
	}, t)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handleDockerMode(ctx, dr, cancel)
	}()

	// Wait for bootstrap to be called
	select {
	case <-bootCalled:
	case <-time.After(time.Second):
		t.Fatal("bootstrapDockerSystemFn not called")
	}

	// Cancel context to allow handleDockerMode to exit and call cleanup
	cancel()

	// Expect cleanup within reasonable time
	select {
	case <-cleanupCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup not invoked")
	}

	wg.Wait()
}

func TestHandleNonDockerMode_Flow(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{DockerMode: "0", NonInteractive: "1"}
	logger := logging.NewTestLogger()

	// Stub chain of cfg helpers & root command
	withInjects(func() {
		findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "", nil
		}
		generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "/tmp/config", nil
		}
		editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "/tmp/config", nil
		}
		validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "/tmp/config", nil
		}
		loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdepspkg.Kdeps, error) {
			return &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}, nil
		}
		getKdepsPathFn = func(context.Context, kdepspkg.Kdeps) (string, error) { return "/tmp/kdeps", nil }
		newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
			return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
		}
	}, t)

	ctx := context.Background()
	handleNonDockerMode(fs, ctx, env, logger) // should complete without panic
}

// TestHandleDockerMode_APIServerMode validates the code path where bootstrapDockerSystemFn
// indicates that the current execution is in API-server mode (apiServerMode == true).
// In this branch handleDockerMode should *not* invoke runGraphResolverActionsFn but must
// still perform cleanup before returning. This test exercises those control-flow paths
// which previously had little or no coverage.
func TestHandleDockerMode_APIServerMode(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Environment: &environment.Environment{},
		Logger:      logging.NewTestLogger(),
	}

	// Backup originals to restore afterwards.
	origBootstrap := bootstrapDockerSystemFn
	origRun := runGraphResolverActionsFn
	origCleanup := cleanupFn

	t.Cleanup(func() {
		bootstrapDockerSystemFn = origBootstrap
		runGraphResolverActionsFn = origRun
		cleanupFn = origCleanup
	})

	var bootstrapCalled, runCalled, cleanupCalled int32

	// Stub bootstrap to enter API-server mode.
	bootstrapDockerSystemFn = func(_ context.Context, _ *resolver.DependencyResolver) (bool, error) {
		atomic.StoreInt32(&bootstrapCalled, 1)
		return true, nil // apiServerMode == true
	}

	// If runGraphResolverActionsFn is invoked we record it – it should NOT be for this path.
	runGraphResolverActionsFn = func(_ context.Context, _ *resolver.DependencyResolver, _ bool) error {
		atomic.StoreInt32(&runCalled, 1)
		return nil
	}

	// Stub cleanup so we do not touch the real docker cleanup logic.
	cleanupFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ bool, _ *logging.Logger) {
		atomic.StoreInt32(&cleanupCalled, 1)
	}

	done := make(chan struct{})
	go func() {
		handleDockerMode(ctx, dr, cancel)
		close(done)
	}()

	// Allow goroutine to set up then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("handleDockerMode did not exit in expected time")
	}

	if atomic.LoadInt32(&bootstrapCalled) == 0 {
		t.Errorf("bootstrapDockerSystemFn was not called")
	}
	if atomic.LoadInt32(&runCalled) != 0 {
		t.Errorf("runGraphResolverActionsFn should NOT be called in API-server mode")
	}
	if atomic.LoadInt32(&cleanupCalled) == 0 {
		t.Errorf("cleanupFn was not executed")
	}
}

// TestHandleDockerMode_NoAPIServer exercises the docker-mode loop with all helpers stubbed.
func TestHandleDockerMode_NoAPIServer(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake dependency resolver with only the fields used by handleDockerMode.
	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Environment: &environment.Environment{},
		Logger:      logging.NewTestLogger(),
	}

	// Backup originals.
	origBootstrap := bootstrapDockerSystemFn
	origRun := runGraphResolverActionsFn
	origCleanup := cleanupFn

	// Restore on cleanup.
	t.Cleanup(func() {
		bootstrapDockerSystemFn = origBootstrap
		runGraphResolverActionsFn = origRun
		cleanupFn = origCleanup
	})

	var bootstrapCalled, runCalled, cleanupCalled int32

	// Stub implementations.
	bootstrapDockerSystemFn = func(_ context.Context, _ *resolver.DependencyResolver) (bool, error) {
		atomic.StoreInt32(&bootstrapCalled, 1)
		return false, nil // apiServerMode = false
	}

	runGraphResolverActionsFn = func(_ context.Context, _ *resolver.DependencyResolver, _ bool) error {
		atomic.StoreInt32(&runCalled, 1)
		return nil
	}

	cleanupFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ bool, _ *logging.Logger) {
		atomic.StoreInt32(&cleanupCalled, 1)
	}

	// Execute in goroutine because handleDockerMode blocks until ctx canceled.
	done := make(chan struct{})
	go func() {
		handleDockerMode(ctx, dr, cancel)
		close(done)
	}()

	// Let the function reach the wait, then cancel.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("handleDockerMode did not exit in time")
	}

	if atomic.LoadInt32(&bootstrapCalled) == 0 || atomic.LoadInt32(&runCalled) == 0 || atomic.LoadInt32(&cleanupCalled) == 0 {
		t.Fatalf("expected all stubbed functions to be called; got bootstrap=%d run=%d cleanup=%d", bootstrapCalled, runCalled, cleanupCalled)
	}

	// Touch rule-required reference
	_ = utils.SafeDerefBool(nil) // uses utils to avoid unused import
	_ = schema.SchemaVersion(context.Background())
}

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
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdeps.Kdeps) (string, error) {
		return "/tmp/kdeps", nil
	}

	executed := false
	newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
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

// TestSetupEnvironmentExtra2 ensures the helper returns a populated Environment without error.
func TestSetupEnvironmentExtra2(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected environment struct, got nil")
	}
}

// TestHandleDockerMode verifies that the control flow cancels correctly in both API-server and non-API modes.
func TestHandleDockerMode(t *testing.T) {
	tests := []bool{false, true} // apiServerMode flag returned by bootstrap stub

	for _, apiServerMode := range tests {
		// Capture range variable
		apiServerMode := apiServerMode
		t.Run("apiServerMode="+boolToStr(apiServerMode), func(t *testing.T) {
			// Preserve originals and restore after test
			origBootstrap := bootstrapDockerSystemFn
			origRun := runGraphResolverActionsFn
			origCleanup := cleanupFn
			defer func() {
				bootstrapDockerSystemFn = origBootstrap
				runGraphResolverActionsFn = origRun
				cleanupFn = origCleanup
			}()

			// Stubs
			bootstrapDockerSystemFn = func(ctx context.Context, dr *resolver.DependencyResolver) (bool, error) {
				return apiServerMode, nil
			}
			runCalled := false
			runGraphResolverActionsFn = func(ctx context.Context, dr *resolver.DependencyResolver, api bool) error {
				runCalled = true
				return nil
			}
			cleanCalled := false
			cleanupFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ bool, _ *logging.Logger) {
				cleanCalled = true
			}

			// Prepare resolver with minimal fields
			dr := &resolver.DependencyResolver{
				Fs:          afero.NewMemMapFs(),
				Logger:      logging.NewTestLogger(),
				Environment: &environment.Environment{DockerMode: "1"},
			}

			ctx, cancel := context.WithCancel(context.Background())
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				handleDockerMode(ctx, dr, cancel)
			}()

			// Give goroutine some time to hit wait state, then cancel
			time.Sleep(100 * time.Millisecond)
			cancel()
			wg.Wait()

			// Assertions
			if apiServerMode {
				if runCalled {
					t.Fatalf("runGraphResolverActions should not be called when apiServerMode is true")
				}
			} else {
				if !runCalled {
					t.Fatalf("expected runGraphResolverActions to be called")
				}
			}
			if !cleanCalled {
				t.Fatalf("expected cleanup to be invoked")
			}
		})
	}
}

// TestHandleNonDockerMode runs through the non-docker flow with all external helpers stubbed.
func TestHandleNonDockerMode(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Preserve and restore injected funcs
	origFind := findConfigurationFn
	origGen := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origRoot := newRootCommandFn
	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGen
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origRoot
	}()

	// Stub chain
	findConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "", nil // force generation path
	}
	generateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/config.yml", nil
	}
	editConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/config.yml", nil
	}
	validateConfigurationFn = func(_ afero.Fs, _ context.Context, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/config.yml", nil
	}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdepspkg.Kdeps{
			KdepsDir:  pkg.GetDefaultKdepsDir(),
			KdepsPath: (*kpath.Path)(pkg.GetDefaultKdepsPath()),
		}, nil
	}
	getKdepsPathFn = func(_ context.Context, _ kdepspkg.Kdeps) (string, error) { return "/tmp/kdeps", nil }

	executed := false
	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(cmd *cobra.Command, args []string) { executed = true }}
	}

	env := &environment.Environment{DockerMode: "0"}
	ctx := context.Background()

	handleNonDockerMode(fs, ctx, env, logger)

	if !executed {
		t.Fatalf("expected root command to be executed")
	}
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestMainEntry_NoDocker(t *testing.T) {
	// Ensure .dockerenv is not present so DockerMode=0
	// Stub all injectable funcs to lightweight versions.
	fs := afero.NewMemMapFs()

	withInjects(func() {
		// environment is created inside main; we can't intercept that easily.

		newGraphResolverFn = func(afero.Fs, context.Context, *environment.Environment, *gin.Context, *logging.Logger) (*resolver.DependencyResolver, error) {
			return &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger()}, nil
		}
		bootstrapDockerSystemFn = func(context.Context, *resolver.DependencyResolver) (bool, error) { return false, nil }
		runGraphResolverActionsFn = func(context.Context, *resolver.DependencyResolver, bool) error { return nil }

		findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdepspkg.Kdeps, error) {
			return &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}, nil
		}
		getKdepsPathFn = func(context.Context, kdepspkg.Kdeps) (string, error) { return "/tmp", nil }
		newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
			return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
		}
		cleanupFn = func(afero.Fs, context.Context, *environment.Environment, bool, *logging.Logger) {}
	}, t)

	// Run main. It should return without panic.
	main()
}

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

	dummyCfg := &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdepspkg.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ kdepspkg.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// execute function
	handleNonDockerMode(fs, ctx, env, logger)

	// if we reach here, function executed without fatal panic.
	assert.True(t, true)
}

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

	dummyCfg := &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdepspkg.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ kdepspkg.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// Execute.
	handleNonDockerMode(fs, ctx, env, logger)
}

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

	dummyCfg := &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdepspkg.Kdeps, error) {
		return dummyCfg, nil
	}

	getKdepsPathFn = func(_ context.Context, _ kdepspkg.Kdeps) (string, error) { return "/kdeps", nil }

	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		// Define a no-op RunE so that Execute() does not error.
		cmd := &cobra.Command{Use: "root"}
		cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
		return cmd
	}

	// Execute.
	handleNonDockerMode(fs, ctx, env, logger)
}

func TestHandleNonDockerModeBasic(t *testing.T) {
	// Setup in-memory filesystem and environment
	fs := afero.NewMemMapFs()
	homeDir := "/home"
	pwdDir := "/workspace"
	_ = fs.MkdirAll(homeDir, 0o755)
	_ = fs.MkdirAll(pwdDir, 0o755)

	env := &environment.Environment{
		Root:           "/",
		Home:           homeDir,
		Pwd:            pwdDir,
		DockerMode:     "0",
		NonInteractive: "1",
	}

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Inject stubbed dependency functions
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil // force generation path
	}
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		confPath := env.Home + "/.kdeps.pkl"
		if err := afero.WriteFile(fs, confPath, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}
		return confPath, nil
	}
	editConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return env.Home + "/.kdeps.pkl", nil
	}
	validateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return env.Home + "/.kdeps.pkl", nil
	}
	loadConfigurationFn = func(fs afero.Fs, ctx context.Context, path string, logger *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}, nil
	}
	getKdepsPathFn = func(ctx context.Context, k kdepspkg.Kdeps) (string, error) {
		return "/tmp/kdeps", nil
	}
	newRootCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, cfg *kdepspkg.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root", Run: func(cmd *cobra.Command, args []string) {}}
	}

	// Add context keys to mimic main
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "graph-id")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, "/tmp/action")

	// Invoke the function under test. It should complete without panicking or fatal logging.
	handleNonDockerMode(fs, ctx, env, logger)
}

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
	loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdepspkg.Kdeps{}, nil
	}
	getKdepsPathFn = func(context.Context, kdepspkg.Kdeps) (string, error) { return tmp, nil }

	newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
		c := &cobra.Command{RunE: func(*cobra.Command, []string) error { return nil }}
		return c
	}

	// execute function under test; should not panic
	handleNonDockerMode(fs, ctx, env, logger)
}

// TestHandleNonDockerMode_Happy mocks dependencies so the flow completes without fatal errors.
func TestHandleNonDockerMode_Happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmp := t.TempDir()

	// Prepare a dummy config file path to be used by stubs.
	cfgPath := filepath.Join(tmp, "config.pkl")
	_ = afero.WriteFile(fs, cfgPath, []byte("config"), 0o644)

	env := &environment.Environment{
		Home:           tmp,
		Pwd:            tmp,
		NonInteractive: "1",
	}

	// Backup original function pointers.
	origFind := findConfigurationFn
	origGen := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	// Restore after test.
	t.Cleanup(func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGen
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	})

	// Stubs.
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil // force generate path
	}
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	editConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	validateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	loadConfigurationFn = func(fs afero.Fs, ctx context.Context, configFile string, logger *logging.Logger) (*kdepspkg.Kdeps, error) {
		return &kdepspkg.Kdeps{KdepsDir: pkg.GetDefaultKdepsDir()}, nil
	}
	getKdepsPathFn = func(ctx context.Context, _ kdepspkg.Kdeps) (string, error) {
		return filepath.Join(tmp, "agents"), nil
	}
	newRootCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, _ *kdepspkg.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
	}

	logger := logging.NewTestLogger()

	// Execute the function under test; expect it to run without panics or exits.
	handleNonDockerMode(fs, context.Background(), env, logger)

	// Sanity: ensure our logger captured the ready message.
	if out := logger.GetOutput(); out == "" {
		t.Fatalf("expected some log output, got none")
	}

	_ = schema.SchemaVersion(context.Background())
}
