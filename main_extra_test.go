package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

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
