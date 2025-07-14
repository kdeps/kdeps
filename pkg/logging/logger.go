package logging

import (
	"bytes"
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

type Logger struct {
	*log.Logger
	buffer *bytes.Buffer // Buffer to capture logs in tests
}

var (
	logger *Logger
	once   sync.Once
)

func CreateLogger() {
	once.Do(func() {
		baseLogger := log.New(os.Stderr)
		if os.Getenv("DEBUG") == "1" {
			baseLogger = log.NewWithOptions(os.Stderr, log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				Prefix:          "kdeps",
			})
			baseLogger.SetLevel(log.DebugLevel)
		} else {
			baseLogger.SetLevel(log.InfoLevel)
		}
		logger = &Logger{Logger: baseLogger}
	})
}

// NewTestLogger creates a logger that writes to a buffer for testing.
func NewTestLogger() *Logger {
	buf := new(bytes.Buffer)
	baseLogger := log.New(buf)
	baseLogger.SetLevel(log.DebugLevel)
	baseLogger.SetFormatter(log.TextFormatter)
	return &Logger{
		Logger: baseLogger,
		buffer: buf,
	}
}

// GetOutput returns the captured log output for test verification.
func (l *Logger) GetOutput() string {
	if l.buffer == nil {
		return ""
	}
	return l.buffer.String()
}

// Debug logs debug messages if debug logging is enabled.
func Debug(msg interface{}, keyvals ...interface{}) {
	ensureInitialized()
	logger.Debug(msg, keyvals...)
}

// Info logs informational messages.
func Info(msg interface{}, keyvals ...interface{}) {
	ensureInitialized()
	logger.Info(msg, keyvals...)
}

// Warn logs warning messages.
func Warn(msg interface{}, keyvals ...interface{}) {
	ensureInitialized()
	logger.Warn(msg, keyvals...)
}

// Error logs error messages.
func Error(msg interface{}, keyvals ...interface{}) {
	ensureInitialized()
	logger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and exits the program.
func Fatal(msg interface{}, keyvals ...interface{}) {
	ensureInitialized()
	logger.Fatal(msg, keyvals...)
}

// GetLogger returns the Logger instance.
func GetLogger() *Logger {
	ensureInitialized()
	return logger
}

// IsDebugEnabled returns true if debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.Logger.GetLevel() <= log.DebugLevel
}

// UnderlyingLogger returns the underlying *log.Logger from the custom Logger.
func (l *Logger) BaseLogger() *log.Logger {
	if l == nil || l.Logger == nil {
		panic("logger not initialized")
	}
	return l.Logger
}

// ensureInitialized ensures the logger is initialized before use.
func ensureInitialized() {
	if logger == nil {
		CreateLogger()
	}
}

// Add this method to your Logger struct.
func (l *Logger) With(keyvals ...interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(keyvals...),
		buffer: l.buffer,
	}
}
