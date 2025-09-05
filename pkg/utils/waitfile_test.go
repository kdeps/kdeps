package utils

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestWaitForFileReady_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	file := filepath.Join(dir, "flag")

	logger := logging.NewTestLogger()

	// Create the file after 200ms
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = afero.WriteFile(fs, file, []byte("done"), 0o644)
	}()

	start := time.Now()
	if err := WaitForFileReady(fs, file, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("WaitForFileReady took too long: %v", elapsed)
	}
}

func TestWaitForFileReady_Timeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	file := filepath.Join(dir, "never")

	err := WaitForFileReady(fs, file, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestGenerateResourceIDFilenameInWaitfileContext(t *testing.T) {
	got := GenerateResourceIDFilename("@foo/bar:baz", "req-")
	expected := "req-_foo_bar_baz"
	if got != expected {
		t.Fatalf("unexpected filename: %s", got)
	}
}
