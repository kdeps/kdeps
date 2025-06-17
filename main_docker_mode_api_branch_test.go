package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

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
