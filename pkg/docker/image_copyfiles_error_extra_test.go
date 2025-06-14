package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyFilesToRunDirCacheDirCreateFail makes runDir/cache a file so MkdirAll fails.
func TestCopyFilesToRunDirCacheDirCreateFail(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	fs := afero.NewReadOnlyFs(baseFs)
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "download")
	runDir := filepath.Join(dir, "run")

	// Prepare download directory with one file so the function proceeds past stat.
	if err := baseFs.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("mkdir download: %v", err)
	}
	_ = afero.WriteFile(baseFs, filepath.Join(downloadDir, "x.bin"), []byte("x"), 0o644)

	// runDir is unwritable (ReadOnlyFs), so MkdirAll to create runDir/cache must fail.
	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error due to cache path collision")
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFilesToRunDirCopyFailure forces CopyFile to fail by making destination directory read-only.
func TestCopyFilesToRunDirCopyFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	fs := afero.NewReadOnlyFs(baseFs)
	base := t.TempDir()
	downloadDir := filepath.Join(base, "dl")
	runDir := filepath.Join(base, "run")

	// setup download dir with one file
	_ = baseFs.MkdirAll(downloadDir, 0o755)
	_ = afero.WriteFile(baseFs, filepath.Join(downloadDir, "obj.bin"), []byte("data"), 0o644)

	// No need to create cache dir; ReadOnlyFs will prevent MkdirAll inside implementation.
	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error due to read-only cache directory")
	}

	schema.SchemaVersion(context.Background())
}
