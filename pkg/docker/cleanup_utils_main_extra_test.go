package docker

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCleanupDockerMode_Timeout ensures that Cleanup enters DockerMode branch,
// removes the actionDir, and returns after WaitForFileReady timeout without panic.
func TestCleanupDockerMode_Timeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	graphID := "gid123"
	actionDir := "/action"

	// Create the directories and dummy files that will be deleted during cleanup.
	if err := fs.MkdirAll(actionDir, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	_ = afero.WriteFile(fs, actionDir+"/dummy.txt", []byte("x"), 0o644)

	// Also create project directory with file, though copy step may not be reached.
	if err := fs.MkdirAll("/agent/project", 0o755); err != nil {
		t.Fatalf("setup project dir: %v", err)
	}
	_ = afero.WriteFile(fs, "/agent/project/hello.txt", []byte("hi"), 0o644)

	// Prepare context with graphID and actionDir.
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	env := &environment.Environment{DockerMode: "1"}

	start := time.Now()
	Cleanup(fs, ctx, env, logger) // should block ~1s due to WaitForFileReady timeout
	elapsed := time.Since(start)
	if elapsed < time.Second {
		t.Fatalf("expected at least 1s wait, got %v", elapsed)
	}

	// Verify actionDir has been removed.
	if exists, _ := afero.DirExists(fs, actionDir); exists {
		t.Fatalf("expected actionDir to be removed")
	}
}
