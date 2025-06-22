package logging

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

type Logger struct {
	*log.Logger
	Buffer  *bytes.Buffer // Buffer to capture logs in tests
	FatalFn func(int)     // Function to call on fatal, defaults to os.Exit
}

var (
	logger *Logger
	once   sync.Once

	// Injectable for testability
	ExitFn = os.Exit
	Stderr = os.Stderr
)

func CreateLogger() {
	once.Do(func() {
		baseLogger := log.New(Stderr)
		if os.Getenv("DEBUG") == "1" {
			baseLogger = log.NewWithOptions(Stderr, log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				Prefix:          "kdeps",
			})
			baseLogger.SetLevel(log.DebugLevel)
		} else {
			baseLogger.SetLevel(log.InfoLevel)
		}
		logger = &Logger{
			Logger:  baseLogger,
			FatalFn: ExitFn,
		}
	})
}

// NewTestLogger creates a logger that writes to a buffer for testing.
func NewTestLogger() *Logger {
	buf := new(bytes.Buffer)
	baseLogger := log.New(buf)
	baseLogger.SetLevel(log.DebugLevel)
	baseLogger.SetFormatter(log.TextFormatter)
	return &Logger{
		Logger:  baseLogger,
		Buffer:  buf,
		FatalFn: ExitFn, // Use injectable ExitFn
	}
}

// NewTestSafeLogger creates a logger for tests that doesn't call os.Exit on fatal.
func NewTestSafeLogger() *Logger {
	buf := new(bytes.Buffer)
	baseLogger := log.New(buf)
	baseLogger.SetLevel(log.DebugLevel)
	baseLogger.SetFormatter(log.TextFormatter)
	return &Logger{
		Logger:  baseLogger,
		Buffer:  buf,
		FatalFn: func(code int) {}, // No-op for test safety
	}
}

// GetOutput returns the captured log output for test verification.
func (l *Logger) GetOutput() string {
	if l.Buffer == nil {
		return ""
	}
	return l.Buffer.String()
}

// Debug logs debug messages if debug logging is enabled.
func Debug(msg interface{}, keyvals ...interface{}) {
	EnsureInitialized()
	logger.Debug(msg, keyvals...)
}

// Info logs informational messages.
func Info(msg interface{}, keyvals ...interface{}) {
	EnsureInitialized()
	logger.Info(msg, keyvals...)
}

// Warn logs warning messages.
func Warn(msg interface{}, keyvals ...interface{}) {
	EnsureInitialized()
	logger.Warn(msg, keyvals...)
}

// Error logs error messages.
func Error(msg interface{}, keyvals ...interface{}) {
	EnsureInitialized()
	logger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and calls the FatalFn.
func Fatal(msg interface{}, keyvals ...interface{}) {
	EnsureInitialized()
	logger.Fatal(msg, keyvals...)
}

// Fatalf logs a fatal message with formatting and calls the FatalFn.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
	if l.FatalFn != nil {
		l.FatalFn(1)
	}
}

// GetLogger returns the Logger instance.
func GetLogger() *Logger {
	EnsureInitialized()
	return logger
}

// UnderlyingLogger returns the underlying *log.Logger from the custom Logger.
func (l *Logger) BaseLogger() *log.Logger {
	if l == nil || l.Logger == nil {
		panic("logger not initialized")
	}
	return l.Logger
}

// ensureInitialized ensures the logger is initialized before use.
func EnsureInitialized() {
	if logger == nil {
		CreateLogger()
	}
}

// Add this method to your Logger struct.
func (l *Logger) With(keyvals ...interface{}) *Logger {
	return &Logger{
		Logger:  l.Logger.With(keyvals...),
		Buffer:  l.Buffer,
		FatalFn: l.FatalFn,
	}
}

// ResetForTest resets the global logger state and sync.Once for testing purposes.
func ResetForTest() {
	logger = nil
	// Reset sync.Once
	once = sync.Once{}
}

// SetTestLogger allows tests to inject a custom logger instance.
func SetTestLogger(l *Logger) {
	logger = l
}

// Fatal logs a fatal message and calls the FatalFn.
func (l *Logger) Fatal(msg interface{}, keyvals ...interface{}) {
	l.Error(msg, keyvals...)
	if l.FatalFn != nil {
		l.FatalFn(1)
	}
}
