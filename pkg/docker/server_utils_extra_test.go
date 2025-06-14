package docker

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestIsServerReadyAndWaitForServer(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a dummy TCP listener on a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String() // e.g. 127.0.0.1:54321
	host, port, _ := strings.Cut(addr, ":")

	// isServerReady should return true.
	if ready := isServerReady(host, port, logger); !ready {
		t.Errorf("expected server to be ready on %s:%s", host, port)
	}

	// waitForServer should return quickly because it's already ready.
	if err := waitForServer(host, port, 3*time.Second, logger); err != nil {
		t.Errorf("waitForServer returned error: %v", err)
	}

	// Close the listener to test negative case quickly with isServerReady
	ln.Close()
	// Choose a port unlikely to be in use (listener just closed)
	pInt, _ := strconv.Atoi(port)
	unavailablePort := strconv.Itoa(pInt)
	if ready := isServerReady(host, unavailablePort, logger); ready {
		t.Errorf("expected server NOT to be ready on closed port %s", unavailablePort)
	}
}

func TestWaitForServerTimeout(t *testing.T) {
	logger := logging.NewTestLogger()

	// Use an unlikely port to be open
	host := "127.0.0.1"
	port := "65000"

	start := time.Now()
	err := waitForServer(host, port, 1500*time.Millisecond, logger)
	duration := time.Since(start)

	if err == nil {
		t.Errorf("expected timeout error for unopened port")
	}
	// Ensure it respected the timeout (±500ms)
	if duration < time.Second || duration > 3*time.Second {
		t.Errorf("waitForServer duration out of expected bounds: %v", duration)
	}
}

func TestIsServerReadyListener(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP listener to simulate ready server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	portStr := strconv.Itoa(addr.Port)

	if !isServerReady("127.0.0.1", portStr, logger) {
		t.Fatalf("expected server to be ready on open port")
	}
	ln.Close()

	// After closing listener, readiness should fail
	if isServerReady("127.0.0.1", portStr, logger) {
		t.Fatalf("expected server NOT ready after listener closed")
	}
}

func TestWaitForServerTimeoutShort(t *testing.T) {
	logger := logging.NewTestLogger()
	port := "65534" // unlikely to be in use
	start := time.Now()
	err := waitForServer("127.0.0.1", port, 1500*time.Millisecond, logger)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if time.Since(start) < 1500*time.Millisecond {
		t.Fatalf("waitForServer returned too early")
	}
}
