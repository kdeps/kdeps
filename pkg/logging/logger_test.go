package logging_test

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

// resetLoggerState resets the logger for testing using the provided helper.
func resetLoggerState() {
	logging.ResetForTest()
}

func TestCreateLogger(t *testing.T) {
	resetLoggerState()
	// Test normal logger creation
	logging.CreateLogger()
	assert.NotNil(t, logging.GetLogger())

	resetLoggerState()
	t.Setenv("DEBUG", "1")
	logging.CreateLogger()
	assert.NotNil(t, logging.GetLogger())
}

func TestNewTestLogger(t *testing.T) {
	testLogger := logging.NewTestLogger()
	assert.NotNil(t, testLogger)
	assert.NotNil(t, testLogger.Logger)
	assert.NotNil(t, testLogger.Buffer)
}

func TestGetOutput(t *testing.T) {
	testLogger := logging.NewTestLogger()
	assert.Equal(t, "", testLogger.GetOutput())

	testLogger.Info("test message")
	output := testLogger.GetOutput()
	assert.Contains(t, output, "test message")

	// Test GetOutput with nil buffer
	loggerWithNilBuffer := &logging.Logger{
		Logger: testLogger.Logger,
		Buffer: nil,
	}
	assert.Equal(t, "", loggerWithNilBuffer.GetOutput())
}

func TestLogLevels(t *testing.T) {
	testLogger := logging.NewTestLogger()
	// Override global logger in package for test
	logging.ResetForTest()
	// Use reflection not needed; rely on TestLogger via package-level function
	// but we can set it through CreateLogger? simpler: set global variable via function not exported; instead call package private variable? can't.
	// We'll skip capturing internal logger; instead call package functions that use GetLogger's underlying logger.
	logging.SetTestLogger(testLogger)

	// Test Debug
	logging.Debug("debug message", "key", "value")
	output := testLogger.GetOutput()
	t.Logf("Debug output: %q", output)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.Buffer.Reset()
	testLogger = logging.NewTestLogger()
	logging.SetTestLogger(testLogger)

	// Test Info
	logging.Info("info message", "key", "value")
	output = testLogger.GetOutput()
	t.Logf("Info output: %q", output)
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.Buffer.Reset()
	testLogger = logging.NewTestLogger()
	logging.SetTestLogger(testLogger)

	// Test Warn
	logging.Warn("warning message", "key", "value")
	output = testLogger.GetOutput()
	t.Logf("Warn output: %q", output)
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.Buffer.Reset()
	testLogger = logging.NewTestLogger()
	logging.SetTestLogger(testLogger)

	// Test Error
	logging.Error("error message", "key", "value")
	output = testLogger.GetOutput()
	t.Logf("Error output: %q", output)
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestGetLogger(t *testing.T) {
	// Don't run in parallel due to global state manipulation
	resetLoggerState()
	// Test before initialization
	assert.NotNil(t, logging.GetLogger()) // This should create a new logger

	// Test after initialization
	assert.NotNil(t, logging.GetLogger())

	resetLoggerState()
	// Test with nil logger
	assert.NotNil(t, logging.GetLogger())
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
	assert.Equal(t, testLogger.Buffer, newLogger.Buffer)

	// Test with multiple key-value pairs
	newLogger = testLogger.With("key1", "value1", "key2", "value2")
	assert.NotNil(t, newLogger)
	assert.Equal(t, testLogger.Buffer, newLogger.Buffer)
}

func TestFatal(t *testing.T) {
	// Since Fatal calls os.Exit, we can't test it directly
	// This is a limitation of testing fatal conditions
	// In practice, this would be tested through integration tests

	// However, we can test that Fatal at least initializes the logger
	testLogger := logging.NewTestLogger()
	logging.SetTestLogger(testLogger)

	// We can't actually call Fatal() because it will exit the test
	// But we can verify the function exists and the logger is set up
	assert.NotNil(t, logging.GetLogger())
}

func TestEnsureInitialized(t *testing.T) {
	// Don't run in parallel due to global state manipulation
	resetLoggerState()
	// Test initialization
	logging.EnsureInitialized()
	assert.NotNil(t, logging.GetLogger())

	// Test that subsequent calls don't change the logger
	originalLogger := logging.GetLogger()
	logging.EnsureInitialized()
	assert.Equal(t, originalLogger, logging.GetLogger())
}

func TestLoggerWithAndOutput(t *testing.T) {
	base := logging.NewTestLogger()
	child := base.With("k", "v")
	child.Info("hello")

	if out := child.GetOutput(); out == "" {
		t.Fatalf("expected output captured")
	}
}

func TestFatal_Subprocess(t *testing.T) {
	if os.Getenv("LOG_FATAL_CHILD") == "1" {
		// In child process: call Fatal which should exit.
		testLogger := logging.NewTestLogger()
		logging.SetTestLogger(testLogger)
		logging.Fatal("fatal message", "key", "value")
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

func TestNewTestSafeLogger(t *testing.T) {
	logger := logging.NewTestSafeLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Buffer)
	assert.NotNil(t, logger.Logger)
	assert.NotNil(t, logger.FatalFn)

	// Test that FatalFn is a no-op (doesn't call os.Exit)
	logger.FatalFn(1) // Should not exit

	// Test that it can log without issues
	logger.Info("test message")
	output := logger.GetOutput()
	assert.Contains(t, output, "test message")
}

func TestLoggerFatalf(t *testing.T) {
	logger := logging.NewTestSafeLogger()

	// Test Fatalf with formatting
	logger.Fatalf("test %s message", "formatted")

	// Should log the error message but not exit due to no-op FatalFn
	output := logger.GetOutput()
	assert.Contains(t, output, "test formatted message")
}

func TestLoggerFatalfWithNilFatalFn(t *testing.T) {
	logger := logging.NewTestSafeLogger()
	logger.FatalFn = nil // Set to nil to test nil check

	// Should not panic when FatalFn is nil
	logger.Fatalf("test message")

	// Should still log the error message
	output := logger.GetOutput()
	assert.Contains(t, output, "test message")
}

func TestLoggerFatalMethod(t *testing.T) {
	logger := logging.NewTestSafeLogger()

	// Test Fatal method
	logger.Fatal("fatal message", "key", "value")

	// Should log the error message but not exit due to no-op FatalFn
	output := logger.GetOutput()
	assert.Contains(t, output, "fatal message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
}

func TestLoggerFatalMethodWithNilFatalFn(t *testing.T) {
	logger := logging.NewTestSafeLogger()
	logger.FatalFn = nil // Set to nil to test nil check

	// Should not panic when FatalFn is nil
	logger.Fatal("fatal message")

	// Should still log the error message
	output := logger.GetOutput()
	assert.Contains(t, output, "fatal message")
}

func TestGetOutputWithNilBuffer(t *testing.T) {
	logger := &logging.Logger{
		Logger:  log.New(os.Stderr),
		Buffer:  nil,
		FatalFn: os.Exit,
	}

	// Should return empty string when buffer is nil
	output := logger.GetOutput()
	assert.Equal(t, "", output)
}

func TestBaseLoggerPanic(t *testing.T) {
	// Test panic when logger is nil
	assert.Panics(t, func() {
		var logger *logging.Logger
		logger.BaseLogger()
	})

	// Test panic when underlying logger is nil
	logger := &logging.Logger{
		Logger:  nil,
		Buffer:  new(bytes.Buffer),
		FatalFn: os.Exit,
	}
	assert.Panics(t, func() {
		logger.BaseLogger()
	})
}

func TestWithMethod(t *testing.T) {
	logger := logging.NewTestSafeLogger()

	// Test With method
	newLogger := logger.With("key", "value")
	assert.NotNil(t, newLogger)
	assert.Equal(t, logger.Buffer, newLogger.Buffer)
	// Don't compare FatalFn as function comparison is not allowed in Go
	assert.NotEqual(t, logger.Logger, newLogger.Logger) // Should be different due to With
}

func TestCreateLoggerWithDebugEnv(t *testing.T) {
	// Set DEBUG environment variable
	os.Setenv("DEBUG", "1")
	defer os.Unsetenv("DEBUG")

	// Reset logger state
	logging.ResetForTest()

	// Create logger
	logging.CreateLogger()

	// Get logger and verify it's created
	logger := logging.GetLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	// Do not compare functions (ExitFn)
}

func TestCreateLoggerWithoutDebugEnv(t *testing.T) {
	// Ensure DEBUG is not set
	os.Unsetenv("DEBUG")

	// Reset logger state
	logging.ResetForTest()

	// Create logger
	logging.CreateLogger()

	// Get logger and verify it's created
	logger := logging.GetLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
	// Do not compare functions (ExitFn)
}

func TestEnsureInitializedCreatesLogger(t *testing.T) {
	// Reset logger state
	logging.ResetForTest()

	// Ensure logger is nil initially
	logging.ResetForTest() // Double reset to ensure nil state

	// Call EnsureInitialized
	logging.EnsureInitialized()

	// Verify logger is created
	assert.NotNil(t, logging.GetLogger())
}

func TestSetTestLogger(t *testing.T) {
	// Reset logger state
	logging.ResetForTest()

	// Create a test logger
	testLogger := logging.NewTestSafeLogger()

	// Set it as the global logger
	logging.SetTestLogger(testLogger)

	// Verify it's set
	assert.Equal(t, testLogger, logging.GetLogger())

	// Test that global functions use the test logger
	logging.Info("test message")
	output := testLogger.GetOutput()
	assert.Contains(t, output, "test message")
}
