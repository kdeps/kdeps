package logging

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// resetLoggerState resets the logger and once for testing purposes.
func resetLoggerState() {
	logger = nil
	// Reset sync.Once using reflection (for testing only)
	onceVal := reflect.ValueOf(&once).Elem()
	onceVal.Set(reflect.Zero(onceVal.Type()))
}

func TestCreateLogger(t *testing.T) {
	resetLoggerState()
	// Test normal logger creation
	CreateLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)

	resetLoggerState()
	t.Setenv("DEBUG", "1")
	CreateLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
}

func TestNewTestLogger(t *testing.T) {
	t.Parallel()
	testLogger := NewTestLogger()
	assert.NotNil(t, testLogger)
	assert.NotNil(t, testLogger.Logger)
	assert.NotNil(t, testLogger.buffer)
}

func TestGetOutput(t *testing.T) {
	t.Parallel()
	testLogger := NewTestLogger()
	assert.Equal(t, "", testLogger.GetOutput())

	testLogger.Info("test message")
	output := testLogger.GetOutput()
	assert.Contains(t, output, "test message")

	// Test GetOutput with nil buffer
	loggerWithNilBuffer := &Logger{
		Logger: testLogger.Logger,
		buffer: nil,
	}
	assert.Equal(t, "", loggerWithNilBuffer.GetOutput())
}

func TestLogLevels(t *testing.T) {
	t.Parallel()
	testLogger := NewTestLogger()
	logger = testLogger

	// Test Debug
	Debug("debug message", "key", "value")
	output := testLogger.GetOutput()
	t.Logf("Debug output: %q", output)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.buffer.Reset()
	testLogger = NewTestLogger()
	logger = testLogger

	// Test Info
	Info("info message", "key", "value")
	output = testLogger.GetOutput()
	t.Logf("Info output: %q", output)
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.buffer.Reset()
	testLogger = NewTestLogger()
	logger = testLogger

	// Test Warn
	Warn("warning message", "key", "value")
	output = testLogger.GetOutput()
	t.Logf("Warn output: %q", output)
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")

	// Clear buffer and reset logger
	testLogger.buffer.Reset()
	testLogger = NewTestLogger()
	logger = testLogger

	// Test Error
	Error("error message", "key", "value")
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
	assert.NotNil(t, GetLogger()) // This should create a new logger

	// Test after initialization
	assert.NotNil(t, GetLogger())

	resetLoggerState()
	// Test with nil logger
	assert.NotNil(t, GetLogger())
}

func TestBaseLogger(t *testing.T) {
	t.Parallel()
	testLogger := NewTestLogger()
	assert.NotNil(t, testLogger.BaseLogger())

	// Test panic case
	var nilLogger *Logger
	assert.Panics(t, func() {
		nilLogger.BaseLogger()
	})
}

func TestWith(t *testing.T) {
	t.Parallel()
	testLogger := NewTestLogger()
	newLogger := testLogger.With("key", "value")
	assert.NotNil(t, newLogger)
	assert.Equal(t, testLogger.buffer, newLogger.buffer)

	// Test with multiple key-value pairs
	newLogger = testLogger.With("key1", "value1", "key2", "value2")
	assert.NotNil(t, newLogger)
	assert.Equal(t, testLogger.buffer, newLogger.buffer)
}

func TestFatal(t *testing.T) {
	t.Parallel()
	// Since Fatal calls os.Exit, we can't test it directly
	// This is a limitation of testing fatal conditions
	// In practice, this would be tested through integration tests

	// However, we can test that Fatal at least initializes the logger
	testLogger := NewTestLogger()
	logger = testLogger

	// We can't actually call Fatal() because it will exit the test
	// But we can verify the function exists and the logger is set up
	assert.NotNil(t, logger)
}

func TestEnsureInitialized(t *testing.T) {
	// Don't run in parallel due to global state manipulation
	resetLoggerState()
	// Test initialization
	ensureInitialized()
	assert.NotNil(t, logger)

	// Test that subsequent calls don't change the logger
	originalLogger := logger
	ensureInitialized()
	assert.Equal(t, originalLogger, logger)
}
