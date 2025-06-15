package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
)

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
