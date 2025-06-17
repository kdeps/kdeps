package docker

import (
	"net"
	"testing"
	"time"

	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestServerReadyHelpers(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a TCP listener on an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	t.Run("isServerReady_true", func(t *testing.T) {
		assert.True(t, isServerReady(host, port, logger))
	})

	t.Run("waitForServer_success", func(t *testing.T) {
		assert.NoError(t, waitForServer(host, port, 2*time.Second, logger))
	})

	// close listener to make port unavailable
	_ = ln.Close()

	t.Run("isServerReady_false", func(t *testing.T) {
		assert.False(t, isServerReady(host, port, logger))
	})

	t.Run("waitForServer_timeout", func(t *testing.T) {
		err := waitForServer(host, port, 1500*time.Millisecond, logger)
		assert.Error(t, err)
	})
}

// TestIsServerReady_Extra checks that the helper correctly detects
// an open TCP port and a closed one.
func TestIsServerReady_Extra(t *testing.T) {
	logger := logging.NewTestLogger()
	// Listen on a random available port on localhost.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	if !isServerReady("127.0.0.1", port, logger) {
		t.Fatalf("server should be reported as ready on open port")
	}

	// pick an arbitrary high port unlikely to be used (and different)
	if isServerReady("127.0.0.1", "65535", logger) {
		t.Fatalf("server should not be ready on closed port")
	}

	schema.SchemaVersion(context.Background()) // maintain convention
}

// TestWaitForServerQuickSuccess ensures waitForServer returns quickly when the
// port is already open.
func TestWaitForServerQuickSuccess(t *testing.T) {
	logger := logging.NewTestLogger()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	start := time.Now()
	if err := waitForServer("127.0.0.1", port, 500*time.Millisecond, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("waitForServer took too long: %v", elapsed)
	}

	schema.SchemaVersion(context.Background())
}

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

func TestIsServerReadyVariants(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a real TCP listener on an ephemeral port to simulate ready server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	if ok := isServerReady(host, port, logger); !ok {
		t.Fatalf("expected server to be ready")
	}

	// Close listener to make port unavailable.
	ln.Close()

	if ok := isServerReady(host, port, logger); ok {
		t.Fatalf("expected server to be NOT ready after close")
	}
}

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

func TestStartOllamaServerSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.NewTestLogger()

	// Call function under test; it should return immediately and not panic.
	startOllamaServer(ctx, logger)

	// Give the background goroutine a brief moment to run and fail gracefully.
	time.Sleep(10 * time.Millisecond)
}

func TestCheckDevBuildModeVariants(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Case 1: file missing -> expect false
	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected false when file absent")
	}

	// Case 2: file present -> expect true
	cacheFile := filepath.Join(kdepsDir, "cache", "kdeps")
	_ = fs.MkdirAll(filepath.Dir(cacheFile), 0o755)
	_ = afero.WriteFile(fs, cacheFile, []byte("bin"), 0o755)

	ok, err = checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when file present")
	}
}

func TestStartOllamaServerStubbed(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Function should return immediately and not panic.
	startOllamaServer(ctx, logger)
}

func TestIsServerReady(t *testing.T) {
	logger := logging.GetLogger()

	t.Run("ServerReady", func(t *testing.T) {
		// Start a test TCP server
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		defer listener.Close()

		host, port, _ := net.SplitHostPort(listener.Addr().String())
		ready := isServerReady(host, port, logger)
		assert.True(t, ready)
	})

	t.Run("ServerNotReady", func(t *testing.T) {
		ready := isServerReady("127.0.0.1", "99999", logger)
		assert.False(t, ready)
	})
}

func TestWaitForServer(t *testing.T) {
	logger := logging.GetLogger()

	t.Run("ServerReady", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		defer listener.Close()

		host, port, _ := net.SplitHostPort(listener.Addr().String())
		err = waitForServer(host, port, 2*time.Second, logger)
		assert.NoError(t, err)
	})

	t.Run("Timeout", func(t *testing.T) {
		err := waitForServer("127.0.0.1", "99999", 1*time.Second, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestStartOllamaServer(t *testing.T) {
	ctx := context.Background()
	// Initialize a proper logger to avoid nil pointer dereference
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}

	// Simply call the function to ensure it doesn't panic
	// Since it runs in background, we can't easily check the result
	startOllamaServer(ctx, logger)

	// If we reach here without panic, the test passes
	t.Log("startOllamaServer called without panic")
}

func TestWaitForServerSuccess(t *testing.T) {
	logger := logging.NewTestLogger()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	host, port, _ := net.SplitHostPort(ln.Addr().String())

	if err := waitForServer(host, port, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}
}

func TestWaitForServerReadyAndTimeout(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a real TCP listener on an ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())

	// Ready case: should return quickly.
	start := time.Now()
	if err := waitForServer(host, portStr, 2*time.Second, logger); err != nil {
		t.Fatalf("expected server to be ready, got error: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("waitForServer took too long for ready case")
	}

	// Timeout case: use a different unused port.
	unusedPort := strconv.Itoa(60000)
	start = time.Now()
	err = waitForServer(host, unusedPort, 1*time.Second, logger)
	if err == nil {
		t.Fatalf("expected timeout error for unopened port")
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Fatalf("waitForServer returned too quickly on timeout path")
	}
}

func TestParseOLLAMAHostVariants(t *testing.T) {
	logger := logging.NewTestLogger()

	// Success path.
	_ = os.Setenv("OLLAMA_HOST", "0.0.0.0:12345")
	host, port, err := parseOLLAMAHost(logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "0.0.0.0" || port != "12345" {
		t.Fatalf("incorrect parse result: %s %s", host, port)
	}

	// Invalid format path.
	_ = os.Setenv("OLLAMA_HOST", "badformat")
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Missing var path.
	_ = os.Unsetenv("OLLAMA_HOST")
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when OLLAMA_HOST unset")
	}
}
