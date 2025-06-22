package utils_test

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

	. "github.com/kdeps/kdeps/pkg/utils"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestSendSigterm_Subprocess(t *testing.T) {
	if os.Getenv("SIGTERM_HELPER") == "1" {
		// Child process: intercept SIGTERM so default action doesn't kill us.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM)
		go func() {
			<-sigCh
			os.Exit(0) // graceful exit when signal received
		}()
		SendSigterm(logging.NewTestLogger())
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

// TestSendSigterm_InjectableFunctions tests error paths using injectable functions
func TestSendSigterm_InjectableFunctions(t *testing.T) {
	t.Run("FindProcessError", func(t *testing.T) {
		if os.Getenv("TEST_FIND_PROCESS_ERROR") == "1" {
			// Save original functions
			originalOsFindProcess := OsFindProcessFunc
			defer func() { OsFindProcessFunc = originalOsFindProcess }()

			// Mock OsFindProcessFunc to return an error
			OsFindProcessFunc = func(pid int) (Process, error) {
				return nil, errors.New("process not found")
			}

			logger := logging.NewTestLogger()
			SendSigterm(logger) // This will call logger.Fatal and exit
			return
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestSendSigterm_InjectableFunctions/FindProcessError")
		cmd.Env = append(os.Environ(), "TEST_FIND_PROCESS_ERROR=1")
		err := cmd.Run()

		// Expect non-zero exit due to logger.Fatal
		if err == nil {
			t.Fatal("expected subprocess to exit with error due to logger.Fatal")
		}
	})

	t.Run("SignalError", func(t *testing.T) {
		if os.Getenv("TEST_SIGNAL_ERROR") == "1" {
			// Save original functions
			originalOsFindProcess := OsFindProcessFunc
			defer func() { OsFindProcessFunc = originalOsFindProcess }()

			// Mock OsFindProcessFunc to return a mock process that fails to signal
			OsFindProcessFunc = func(pid int) (Process, error) {
				return &mockProcess{signalErr: errors.New("signal failed")}, nil
			}

			logger := logging.NewTestLogger()
			SendSigterm(logger) // This will call logger.Fatal and exit
			return
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestSendSigterm_InjectableFunctions/SignalError")
		cmd.Env = append(os.Environ(), "TEST_SIGNAL_ERROR=1")
		err := cmd.Run()

		// Expect non-zero exit due to logger.Fatal
		if err == nil {
			t.Fatal("expected subprocess to exit with error due to logger.Fatal")
		}
	})

	t.Run("SignalSuccess", func(t *testing.T) {
		if os.Getenv("TEST_SIGNAL_SUCCESS") == "1" {
			// Save original functions
			originalOsFindProcess := OsFindProcessFunc
			defer func() { OsFindProcessFunc = originalOsFindProcess }()

			// Mock OsFindProcessFunc to return a mock process that succeeds
			OsFindProcessFunc = func(pid int) (Process, error) {
				return &mockProcess{signalErr: nil}, nil
			}

			logger := logging.NewTestLogger()
			SendSigterm(logger) // This should succeed and log success
			return
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestSendSigterm_InjectableFunctions/SignalSuccess")
		cmd.Env = append(os.Environ(), "TEST_SIGNAL_SUCCESS=1")
		err := cmd.Run()

		// Expect zero exit for successful case
		if err != nil {
			t.Fatalf("expected subprocess to exit successfully, got: %v", err)
		}
	})
}

// mockProcess implements a mock Process for testing signal errors
type mockProcess struct {
	signalErr error
}

func (m *mockProcess) Signal(sig os.Signal) error { return m.signalErr }
