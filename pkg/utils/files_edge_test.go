package utils

import (
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestWaitForFileReadyEdgeSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	path := "/tmp/file.txt"

	// create file after short delay in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = afero.WriteFile(fs, path, []byte("ok"), 0o644)
	}()

	if err := WaitForFileReady(fs, path, logger); err != nil {
		t.Fatalf("expected file ready, got error %v", err)
	}
}

func TestWaitForFileReadyEdgeTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	start := time.Now()
	err := WaitForFileReady(fs, "/nonexistent", logger)
	duration := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if duration < 1*time.Second {
		t.Fatalf("function returned too early, expected ~1s wait")
	}
}
