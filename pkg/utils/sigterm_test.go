package utils_test

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestSendSigterm(t *testing.T) {
	// Create a logger that outputs to os.Stderr for visibility in tests
	logging.CreateLogger()

	// Create a channel to intercept the SIGTERM signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Run SendSigterm in a goroutine to avoid blocking
	go utils.SendSigterm()

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

func TestSendSigterm_Subprocess(t *testing.T) {
	if os.Getenv("SIGTERM_HELPER") == "1" {
		// Child process: intercept SIGTERM so default action doesn't kill us.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM)
		go func() {
			<-sigCh
			os.Exit(0) // graceful exit when signal received
		}()
		utils.SendSigterm()
		// If SendSigterm failed to deliver, exit non-zero after timeout.
		time.Sleep(500 * time.Millisecond)
		os.Exit(2)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSendSigterm_Subprocess")
	cmd.Env = append(os.Environ(), "SIGTERM_HELPER=1")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("child exited with code %d: %v", exitErr.ExitCode(), err)
		}
		t.Fatalf("failed to run child process: %v", err)
	}
}
