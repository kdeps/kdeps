package logging

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// Reset the global logger and once for each test
	logger = nil
	t.Run("CreateLogger", func(t *testing.T) {
		// Reset once to allow reinitialization
		once = sync.Once{}
		// Test default logger creation
		CreateLogger()
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)

		// Test debug logger creation
		os.Setenv("DEBUG", "1")
		defer os.Unsetenv("DEBUG")
		logger = nil
		once = sync.Once{}
		CreateLogger()
		assert.NotNil(t, logger)
		assert.NotNil(t, logger.Logger)
	})

	t.Run("NewTestLogger", func(t *testing.T) {
		testLogger := NewTestLogger()
		assert.NotNil(t, testLogger)
		assert.NotNil(t, testLogger.Logger)
		assert.NotNil(t, testLogger.buffer)
	})

	t.Run("LoggingFunctions", func(t *testing.T) {
		testLogger := NewTestLogger()
		oldLogger := logger
		logger = testLogger
		defer func() { logger = oldLogger }()

		// Test Debug
		Debug("test debug", "key", "value")
		output := testLogger.GetOutput()
		assert.Contains(t, output, "test debug")
		assert.Contains(t, output, "key=value")

		// Test Info
		Info("test info", "key", "value")
		output = testLogger.GetOutput()
		assert.Contains(t, output, "test info")
		assert.Contains(t, output, "key=value")

		// Test Warn
		Warn("test warn", "key", "value")
		output = testLogger.GetOutput()
		assert.Contains(t, output, "test warn")
		assert.Contains(t, output, "key=value")

		// Test Error
		Error("test error", "key", "value")
		output = testLogger.GetOutput()
		assert.Contains(t, output, "test error")
		assert.Contains(t, output, "key=value")
	})

	t.Run("GetLogger", func(t *testing.T) {
		oldLogger := logger
		logger = nil
		once = sync.Once{}
		defer func() { logger = oldLogger }()

		// Test GetLogger creates logger if not initialized
		l := GetLogger()
		assert.NotNil(t, l)
		assert.NotNil(t, l.Logger)

		// Test GetLogger returns existing logger
		l2 := GetLogger()
		assert.Equal(t, l, l2)
	})

	t.Run("BaseLogger", func(t *testing.T) {
		testLogger := NewTestLogger()
		baseLogger := testLogger.BaseLogger()
		assert.NotNil(t, baseLogger)

		// Test panic when logger is nil
		var nilLogger *Logger
		assert.Panics(t, func() {
			nilLogger.BaseLogger()
		})
	})

	t.Run("With", func(t *testing.T) {
		testLogger := NewTestLogger()
		oldLogger := logger
		logger = testLogger
		defer func() { logger = oldLogger }()

		// Test With creates new logger with additional fields
		newLogger := testLogger.With("key", "value")
		assert.NotNil(t, newLogger)
		assert.NotEqual(t, testLogger, newLogger)
		assert.Equal(t, testLogger.buffer, newLogger.buffer)

		// Test logging with additional fields
		newLogger.Info("test with fields")
		output := testLogger.GetOutput()
		assert.Contains(t, output, "test with fields")
		assert.Contains(t, output, "key=value")
	})

	t.Run("GetOutput", func(t *testing.T) {
		// Test GetOutput with nil buffer
		var nilLogger Logger
		assert.Empty(t, nilLogger.GetOutput())

		// Test GetOutput with buffer
		testLogger := NewTestLogger()
		testLogger.Info("test output")
		output := testLogger.GetOutput()
		assert.Contains(t, output, "test output")
	})

	t.Run("LogLevels", func(t *testing.T) {
		testLogger := NewTestLogger()
		oldLogger := logger
		logger = testLogger
		defer func() { logger = oldLogger }()

		// Test different log levels
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")

		output := testLogger.GetOutput()
		assert.True(t, strings.Contains(output, "debug message"))
		assert.True(t, strings.Contains(output, "info message"))
		assert.True(t, strings.Contains(output, "warn message"))
		assert.True(t, strings.Contains(output, "error message"))
	})
}
