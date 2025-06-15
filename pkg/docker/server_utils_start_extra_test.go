package docker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
)

// TestStartOllamaServerReturn ensures the helper returns immediately even when the underlying command is missing.
func TestStartOllamaServerReturn(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	startOllamaServer(ctx, logger)
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("startOllamaServer took too long to return")
	}
}

// TestStartOllamaServer_NoBinary ensures the helper returns immediately even when the
// underlying binary is not present on the host machine. It simply exercises the
// code path to boost coverage without making assumptions about the external
// environment.
func TestStartOllamaServer_NoBinary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	startOllamaServer(ctx, logging.NewTestLogger())
	elapsed := time.Since(start)

	// The function should return almost instantly because it only launches the
	// background goroutine. Use a generous threshold to avoid flakes.
	if elapsed > 100*time.Millisecond {
		t.Fatalf("startOllamaServer took too long: %v", elapsed)
	}
}

// TestStartOllamaServerBackground verifies that the helper kicks off the background task and logs as expected.
func TestStartOllamaServerBackground(t *testing.T) {
	// Create a temporary directory that will hold a dummy `ollama` executable.
	tmpDir := t.TempDir()
	dummy := filepath.Join(tmpDir, "ollama")
	if err := os.WriteFile(dummy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write dummy executable: %v", err)
	}

	// Prepend the temp dir to PATH so it's discovered by exec.LookPath.
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	logger := logging.NewTestLogger()

	// Call the function under test; it should return immediately.
	startOllamaServer(context.Background(), logger)

	// Allow some time for the goroutine in KdepsExec to start and finish.
	time.Sleep(150 * time.Millisecond)

	output := logger.GetOutput()
	if !strings.Contains(output, messages.MsgStartOllamaBackground) {
		t.Errorf("expected log %q not found. logs: %s", messages.MsgStartOllamaBackground, output)
	}
	if !strings.Contains(output, "background command started") {
		t.Errorf("expected background start log not found. logs: %s", output)
	}
}
