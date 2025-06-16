package docker

import (
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
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
