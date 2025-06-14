package docker

import (
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

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
