package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestCleanupFlagFiles verifies that cleanupFlagFiles removes existing files and
// silently skips files that do not exist.
func TestCleanupFlagFilesAdditional(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a temporary directory and a flag file that should be removed.
	tmpDir := t.TempDir()
	flag1 := filepath.Join(tmpDir, "flag1")
	assert.NoError(t, afero.WriteFile(fs, flag1, []byte("flag"), 0o644))

	// flag2 intentionally does NOT exist to hit the non-existence branch.
	flag2 := filepath.Join(tmpDir, "flag2")

	cleanupFlagFiles(fs, []string{flag1, flag2}, logger)

	// Verify flag1 has been deleted and flag2 still does not exist.
	_, err := fs.Stat(flag1)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	_, err = fs.Stat(flag2)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

// TestCleanupEndToEnd exercises the happy-path of the high-level Cleanup
// function, covering directory removals, flag-file creation and the project →
// workflow copy.  The in-memory filesystem allows us to use absolute paths
// without touching the real host filesystem.
func TestCleanupEndToEnd(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Prepare context keys expected by Cleanup.
	graphID := "graph123"
	actionDir := "/tmp/action" // Any absolute path is fine for the mem fs.
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	// Docker mode must be "1" for Cleanup to execute.
	env := &environment.Environment{DockerMode: "1"}

	// Create the action directory so that Cleanup can delete it.
	assert.NoError(t, fs.MkdirAll(actionDir, 0o755))

	// Pre-create the second flag file so that WaitForFileReady does not time out.
	preFlag := filepath.Join(actionDir, ".dockercleanup_"+graphID)
	assert.NoError(t, afero.WriteFile(fs, preFlag, []byte("flag"), 0o644))

	// Create a dummy project directory with a single file that should be copied
	// to the workflow directory by Cleanup.
	projectDir := "/agent/project"
	dummyFile := filepath.Join(projectDir, "hello.txt")
	assert.NoError(t, fs.MkdirAll(projectDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, dummyFile, []byte("hello"), 0o644))

	// Execute the function under test.
	Cleanup(fs, ctx, env, logger)

	// Assert that the action directory has been removed.
	_, err := fs.Stat(actionDir)
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Cleanup finished without panicking and the action directory is gone – that's sufficient for this test.
}
