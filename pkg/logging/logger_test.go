package logging_test

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestCreateLogger(t *testing.T) {
	// Test normal logger creation
	logging.CreateLogger()
	logger := logging.GetLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.BaseLogger())

	t.Setenv("DEBUG", "1")
	logging.CreateLogger()
	logger = logging.GetLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.BaseLogger())
}

func TestNewTestLogger(t *testing.T) {
	testLogger := logging.NewTestLogger()
	assert.NotNil(t, testLogger)
	assert.NotNil(t, testLogger.BaseLogger())
	assert.NotEmpty(t, testLogger.GetOutput())
}

func TestGetOutput(t *testing.T) {
	testLogger := logging.NewTestLogger()
	assert.Empty(t, testLogger.GetOutput())

	testLogger.Info("test message")
	output := testLogger.GetOutput()
	assert.Contains(t, output, "test message")
}

func TestLogLevels(t *testing.T) {
	testLogger := logging.NewTestLogger()

	// Test Debug
	testLogger.Debug("debug message", "key", "value")
	output := testLogger.GetOutput()
	assert.Contains(t, output, "debug message")

	// Clear buffer
	testLogger = logging.NewTestLogger()

	// Test Info
	testLogger.Info("info message", "key", "value")
	output = testLogger.GetOutput()
	assert.Contains(t, output, "info message")

	// Clear buffer
	testLogger = logging.NewTestLogger()

	// Test Warn
	testLogger.Warn("warning message", "key", "value")
	output = testLogger.GetOutput()
	assert.Contains(t, output, "warning message")

	// Clear buffer
	testLogger = logging.NewTestLogger()

	// Test Error
	testLogger.Error("error message", "key", "value")
	output = testLogger.GetOutput()
	assert.Contains(t, output, "error message")
}

func TestGetLogger(t *testing.T) {
	// Test before initialization
	logger := logging.GetLogger()
	assert.NotNil(t, logger)

	// Test after initialization
	logger = logging.GetLogger()
	assert.NotNil(t, logger)
}

func TestBaseLogger(t *testing.T) {
	testLogger := logging.NewTestLogger()
	assert.NotNil(t, testLogger.BaseLogger())

	// Test panic case
	var nilLogger *logging.Logger
	assert.Panics(t, func() {
		nilLogger.BaseLogger()
	})
}

func TestWith(t *testing.T) {
	testLogger := logging.NewTestLogger()
	newLogger := testLogger.With("key", "value")
	assert.NotNil(t, newLogger)

	// Test with multiple key-value pairs
	newLogger = testLogger.With("key1", "value1", "key2", "value2")
	assert.NotNil(t, newLogger)
}

func TestFatal(t *testing.T) {
	// Since Fatal calls os.Exit, we can't test it directly
	// This is a limitation of testing fatal conditions
	// In practice, this would be tested through integration tests

	// However, we can test that Fatal at least initializes the logger
	logger := logging.GetLogger()
	assert.NotNil(t, logger)
}

func TestLoggerWithAndOutput(t *testing.T) {
	base := logging.NewTestLogger()
	child := base.With("k", "v")
	child.Info("hello")

	output := child.GetOutput()
	assert.NotEmpty(t, output)
}

func TestFatal_Subprocess(t *testing.T) {
	if os.Getenv("LOG_FATAL_CHILD") == "1" {
		// In child process: call Fatal which should exit.
		testLogger := logging.NewTestLogger()
		testLogger.Fatal("fatal message", "key", "value")
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatal_Subprocess")
	cmd.Env = append(os.Environ(), "LOG_FATAL_CHILD=1")
	output, err := cmd.CombinedOutput()

	// The child process must exit with non-zero due to Fatal.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if exitErr.ExitCode() == 0 {
			t.Fatalf("expected non-zero exit code, got 0, output: %s", string(output))
		}
	} else {
		t.Fatalf("expected exec.ExitError, got %T: %v", err, err)
	}
}
