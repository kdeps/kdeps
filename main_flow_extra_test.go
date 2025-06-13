package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/schema/gen/kdeps"
	kpath "github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

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
	loadConfigurationFn = func(_ afero.Fs, _ context.Context, _ string, _ *logging.Logger) (*kdeps.Kdeps, error) {
		return &kdeps.Kdeps{
			KdepsDir:  ".kdeps",
			KdepsPath: kpath.User,
		}, nil
	}
	getKdepsPathFn = func(_ context.Context, _ kdeps.Kdeps) (string, error) { return "/tmp/kdeps", nil }

	executed := false
	newRootCommandFn = func(_ afero.Fs, _ context.Context, _ string, _ *kdeps.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
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
