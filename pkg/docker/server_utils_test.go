package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

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
	logger := logging.GetLogger()
	ctx := context.Background()

	// This test is more of a smoke test since we can't easily mock the command execution
	startOllamaServer(ctx, logger)
	// No assertions, just ensure it doesn't panic
}
