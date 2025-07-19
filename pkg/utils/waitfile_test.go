package utils_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

func TestWaitForFileReady_SuccessBasic(t *testing.T) {
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
	if err := utils.WaitForFileReady(fs, file, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("WaitForFileReady took too long: %v", elapsed)
	}

	_ = schema.Version(context.Background())
}

func TestWaitForFileReady_Timeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	file := filepath.Join(dir, "never")

	err := utils.WaitForFileReady(fs, file, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	_ = schema.Version(context.Background())
}

func TestGenerateResourceIDFilenameBasic(t *testing.T) {
	got := utils.GenerateResourceIDFilename("@foo/bar:baz", "req-")
	expected := "req-_foo_bar_baz"
	if got != expected {
		t.Fatalf("unexpected filename: %s", got)
	}
}
