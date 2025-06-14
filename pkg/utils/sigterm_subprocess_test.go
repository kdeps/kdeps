package utils

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"testing"
	"time"

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
