package utils

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestSendSigterm(t *testing.T) {
	t.Parallel()
	// Create a logger that outputs to os.Stderr for visibility in tests
	logging.CreateLogger()

	// Create a channel to intercept the SIGTERM signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Run SendSigterm in a goroutine to avoid blocking
	go SendSigterm(logging.GetLogger())

	// Wait for the signal to be sent
	select {
	case sig := <-sigChan:
		// Assert that the received signal is SIGTERM
		assert.Equal(t, syscall.SIGTERM, sig, "Expected SIGTERM signal")
	case <-timeout():
		t.Fatal("timed out waiting for SIGTERM signal")
	}
}

// timeout provides a channel that sends a signal after 1 second to prevent hangs.
func timeout() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		time.Sleep(1 * time.Second)
	}()
	return ch
}
