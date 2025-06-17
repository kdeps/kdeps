package docker

import (
	"net"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

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
