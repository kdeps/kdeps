package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	kdepspkg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

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
			return &kdepspkg.Kdeps{KdepsDir: "."}, nil
		}
		getKdepsPathFn = func(context.Context, kdepspkg.Kdeps) (string, error) { return "/tmp/kdeps", nil }
		newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
			return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
		}
	}, t)

	ctx := context.Background()
	handleNonDockerMode(fs, ctx, env, logger) // should complete without panic
}
