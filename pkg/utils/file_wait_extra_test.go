package utils

import (
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestWaitForFileReadySuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	fname := "/tmp/ready.txt"

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

	start := time.Now()
	err := WaitForFileReady(fs, "/nonexistent", logger)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if time.Since(start) < 990*time.Millisecond {
		t.Fatalf("function returned too early, did not wait full timeout")
	}
}
