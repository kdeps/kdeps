package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyFilesToRunDirSuccess verifies that files in the download cache
// are copied into the run directory cache.
func TestCopyFilesToRunDirSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "download")
	runDir := filepath.Join(dir, "run")

	_ = fs.MkdirAll(downloadDir, 0o755)
	// create two mock files
	_ = afero.WriteFile(fs, filepath.Join(downloadDir, "a.bin"), []byte("A"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(downloadDir, "b.bin"), []byte("B"), 0o600)

	logger := logging.NewTestLogger()
	if err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify they exist in runDir/cache with same names
	for _, name := range []string{"a.bin", "b.bin"} {
		data, err := afero.ReadFile(fs, filepath.Join(runDir, "cache", name))
		if err != nil {
			t.Fatalf("copied file missing: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("copied file empty")
		}
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFilesToRunDirMissingSource ensures a descriptive error when the
// download directory does not exist.
func TestCopyFilesToRunDirMissingSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "no_such")
	runDir := filepath.Join(dir, "run")

	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error for missing download dir, got nil")
	}

	schema.SchemaVersion(context.Background())
}
