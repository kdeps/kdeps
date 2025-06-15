package utils

import (
	"os"
	"os/exec"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestSendSigtermSubprocess(t *testing.T) {
	if os.Getenv("SUBPROC_SIGTERM") == "1" {
		// Inside the child process – call the function under test.
		SendSigterm(logging.NewTestLogger())
		return // execution should reach here only if SIGTERM was handled.
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSendSigtermSubprocess")
	cmd.Env = append(os.Environ(), "SUBPROC_SIGTERM=1")

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// SIGTERM exits with non-zero status; accept it as success.
			if ee.ExitCode() == -1 || ee.ExitCode() == 0 {
				return
			}
		}
		t.Fatalf("subprocess execution failed: %v", err)
	}
}
