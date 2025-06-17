package docker

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCheckDevBuildModeDir verifies that the helper treats a directory named
// "cache/kdeps" as non-dev build mode, exercising the !info.Mode().IsRegular()
// branch for additional coverage.
func TestCheckDevBuildModeDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Create a directory at cache/kdeps instead of a file.
	cacheDir := filepath.Join(kdepsDir, "cache")
	if err := fs.MkdirAll(filepath.Join(cacheDir, "kdeps"), 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected dev mode to be false when path is a directory")
	}
}
