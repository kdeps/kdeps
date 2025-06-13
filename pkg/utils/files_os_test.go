package utils

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCreateFilesErrorOsFs validates the error branch when using a read-only filesystem
// backed by the real OS and a temporary directory.
func TestCreateFilesErrorOsFs(t *testing.T) {
	tmpDir := t.TempDir()
	// The read-only wrapper simulates permission failure.
	roFs := afero.NewReadOnlyFs(afero.NewOsFs())

	files := []string{filepath.Join(tmpDir, "should_fail.txt")}
	err := CreateFiles(roFs, context.Background(), files)
	if err == nil {
		t.Fatalf("expected error when creating files on read-only fs, got nil")
	}
}

// TestWaitForFileReadyOsFs uses a real tmpfile on the OS FS.
func TestWaitForFileReadyOsFs(t *testing.T) {
	osFs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "ready.txt")

	// create file after delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		f, _ := osFs.Create(filePath)
		f.Close()
	}()

	if err := WaitForFileReady(osFs, filePath, logger); err != nil {
		t.Fatalf("WaitForFileReady returned error: %v", err)
	}
}

// TestCreateDirectoriesErrorOsFs validates failure path of CreateDirectories on read-only fs.
func TestCreateDirectoriesErrorOsFs(t *testing.T) {
	tmpDir := t.TempDir()
	roFs := afero.NewReadOnlyFs(afero.NewOsFs())

	dirs := []string{filepath.Join(tmpDir, "subdir")}
	if err := CreateDirectories(roFs, context.Background(), dirs); err == nil {
		t.Fatalf("expected error when creating directory on read-only fs, got nil")
	}
}
