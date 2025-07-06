package utils

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestWaitForFileReadySuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	fname := filepath.Join(tmpDir, "ready.txt")

	// create file after 100ms in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = afero.WriteFile(fs, fname, []byte("ok"), 0o644)
	}()

	if err := WaitForFileReady(fs, fname, logger); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestWaitForFileReadyTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent")

	start := time.Now()
	err := WaitForFileReady(fs, nonexistentPath, logger)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if time.Since(start) < 990*time.Millisecond {
		t.Fatalf("function returned too early, did not wait full timeout")
	}
}

func TestGenerateResourceIDFilenameAdditional(t *testing.T) {
	cases := []struct {
		input string
		reqID string
		want  string
	}{
		{"@foo/bar:baz", "req", "req_foo_bar_baz"},
		{"hello/world", "id", "idhello_world"},
		{"simple", "", "simple"},
	}

	for _, c := range cases {
		got := GenerateResourceIDFilename(c.input, c.reqID)
		if got != c.want {
			t.Fatalf("GenerateResourceIDFilename(%q,%q) = %q; want %q", c.input, c.reqID, got, c.want)
		}
	}
}

func TestSanitizeArchivePathAdditional(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "safe", "root")

	// Good path
	if _, err := SanitizeArchivePath(base, "folder/file.txt"); err != nil {
		t.Fatalf("unexpected error for safe path: %v", err)
	}

	// Attempt path traversal should error
	if _, err := SanitizeArchivePath(base, "../evil.txt"); err == nil {
		t.Fatalf("expected error for tainted path")
	}
}
