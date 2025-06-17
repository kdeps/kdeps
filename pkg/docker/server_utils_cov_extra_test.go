package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// TestIsServerReadyAndWaitForServer covers both positive and timeout scenarios
// for the helper functions in server_utils.go.
func TestIsServerReadyAndWaitForServerExtra(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP server listening on an available port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	host, port, _ := net.SplitHostPort(ln.Addr().String())

	// Expect server to be reported as ready.
	if !isServerReady(host, port, logger) {
		t.Fatalf("expected server to be ready")
	}

	// waitForServer should return quickly for an already-ready server.
	if err := waitForServer(host, port, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}

	// Close listener to test timeout path.
	ln.Close()

	start := time.Now()
	err = waitForServer(host, port, 1*time.Second, logger)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if time.Since(start) < 1*time.Second {
		t.Fatalf("waitForServer returned too quickly, expected it to wait for timeout")
	}

	// Context compile-time check to ensure startOllamaServer callable without panic.
	// We cannot execute it because it would attempt to run an external binary, but we
	// can at least ensure it does not panic when invoked with a canceled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	startOllamaServer(ctx, logger)
}
