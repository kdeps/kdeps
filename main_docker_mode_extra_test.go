package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

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
