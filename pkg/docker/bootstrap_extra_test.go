package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestStartAndWaitForOllamaReady(t *testing.T) {
	// Spin up dummy listener to simulate Ollama server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := logging.NewTestLogger()
	if err := startAndWaitForOllama(ctx, "127.0.0.1", portStr, logger); err != nil {
		t.Errorf("expected nil error when server already ready, got %v", err)
	}
}
