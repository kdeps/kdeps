package utils_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

// errFS wraps an afero.Fs but forces Stat to return an error to exercise the error branch in WaitForFileReady.
type errFS struct{ afero.Fs }

func (e errFS) Stat(name string) (os.FileInfo, error) {
	return nil, errors.New("stat failure")
}

func TestWaitForFileReadyHelper(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	fname := "/tmp/ready.txt"

	// Create the file after a short delay to test the polling loop.
	go func() {
		time.Sleep(100 * time.Millisecond)
		f, _ := fs.Create(fname)
		f.Close()
	}()

	if err := utils.WaitForFileReady(fs, fname, logger); err != nil {
		t.Fatalf("WaitForFileReady returned error: %v", err)
	}

	// Ensure timeout branch returns error when file never appears.
	if err := utils.WaitForFileReady(fs, "/tmp/missing.txt", logger); err == nil {
		t.Errorf("expected timeout error but got nil")
	}
}

func TestCreateDirectoriesAndFilesHelper(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	dirs := []string{"/a/b", "/c/d/e"}
	if err := utils.CreateDirectories(fs, ctx, dirs); err != nil {
		t.Fatalf("CreateDirectories error: %v", err)
	}
	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		if !exists {
			t.Errorf("directory %s not created", d)
		}
	}

	files := []string{"/a/b/file.txt", "/c/d/e/other.txt"}
	if err := utils.CreateFiles(fs, ctx, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		if !exists {
			t.Errorf("file %s not created", f)
		}
	}
}

func TestGenerateResourceIDFilenameAndSanitizeArchivePathHelper(t *testing.T) {
	id := "abc/def:ghi@jkl"
	got := utils.GenerateResourceIDFilename(id, "req-")
	want := "req-abc_def_ghi_jkl"
	if filepath.Base(got) != want {
		t.Errorf("GenerateResourceIDFilename = %s, want %s", got, want)
	}

	good, err := utils.SanitizeArchivePath("/base", "sub/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedGood := filepath.Join("/base", "sub/file.txt")
	if good != expectedGood {
		t.Errorf("SanitizeArchivePath = %s, want %s", good, expectedGood)
	}

	if _, err := utils.SanitizeArchivePath("/base", "../escape.txt"); err == nil {
		t.Errorf("expected error for path escape, got nil")
	}
}

func TestWaitForFileReadyError(t *testing.T) {
	fs := errFS{afero.NewMemMapFs()}
	logger := logging.NewTestLogger()
	if err := utils.WaitForFileReady(fs, "/any", logger); err == nil {
		t.Errorf("expected error due to Stat failure, got nil")
	}
}
