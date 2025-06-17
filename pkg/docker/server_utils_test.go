package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/charmbracelet/log"
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
