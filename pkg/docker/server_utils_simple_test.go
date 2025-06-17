package docker

import (
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestIsServerReadyAndWaitForServerSimple(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP listener to act as a fake Ollama server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())

	// Positive case for isServerReady
	if !isServerReady(host, portStr, logger) {
		t.Fatalf("expected server to be ready on open port")
	}

	// Positive case for waitForServer with short timeout
	if err := waitForServer(host, portStr, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer unexpectedly failed: %v", err)
	}

	// Close listener to test negative path
	ln.Close()

	// Now port should be closed; isServerReady should return false
	if isServerReady(host, portStr, logger) {
		t.Fatalf("expected server not ready after listener closed")
	}

	// waitForServer should timeout quickly
	timeout := 1500 * time.Millisecond
	start := time.Now()
	err = waitForServer(host, portStr, timeout, logger)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	elapsed := time.Since(start)
	// Ensure we waited at least 'timeout' but not excessively more (allow 1s margin)
	if elapsed < timeout || elapsed > timeout+time.Second {
		t.Fatalf("waitForServer elapsed time unexpected: %s (timeout %s)", elapsed, timeout)
	}
}
