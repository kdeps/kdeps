package logging

import (
	"os"
	"os/exec"
	"testing"
)

func TestFatal_Subprocess(t *testing.T) {
	if os.Getenv("LOG_FATAL_CHILD") == "1" {
		// In child process: call Fatal which should exit.
		testLogger := NewTestLogger()
		logger = testLogger
		Fatal("fatal message", "key", "value")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_Subprocess")
	cmd.Env = append(os.Environ(), "LOG_FATAL_CHILD=1")
	output, err := cmd.CombinedOutput()

	// The child process must exit with non-zero due to Fatal.
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", string(output))
		}
	} else {
		t.Fatalf("expected exec.ExitError, got %v, output: %s", err, string(output))
	}

	// The buffer used by Fatal may not flush to combined output, so we skip
	// validating exact message content.
}
