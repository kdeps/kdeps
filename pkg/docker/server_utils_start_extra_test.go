package docker

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
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
