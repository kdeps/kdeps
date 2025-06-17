package docker

import (
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

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
