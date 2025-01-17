package logging

import (
	"os"
	"sync"

	"github.com/charmbracelet/log"
)

// Logger is a wrapper around the log.Logger from the charmbracelet/log package.
type Logger struct {
	*log.Logger
}

var (
	logger *Logger
	once   sync.Once
)

// CreateLogger sets up the logger. It must be called before using the logger.
func CreateLogger() {
	once.Do(func() {
		// Create a logger with default settings
		baseLogger := log.New(os.Stderr)

		// Check if DEBUG environment variable is set to 1
		if os.Getenv("DEBUG") == "1" {
			// Set log options only when DEBUG is enabled
			baseLogger = log.NewWithOptions(os.Stderr, log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				Prefix:          "kdeps",
			})

			baseLogger.SetLevel(log.DebugLevel)
		} else {
			// Use InfoLevel for normal operation without special logging options
			baseLogger.SetLevel(log.InfoLevel)
		}

		// Wrap the base logger in the custom Logger type
		logger = &Logger{Logger: baseLogger}
	})
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

// UnderlyingLogger returns the underlying *log.Logger from the custom Logger.
func (l *Logger) BaseLogger() *log.Logger {
	return l.Logger
}

// ensureInitialized ensures the logger is initialized before use.
func ensureInitialized() {
	if logger == nil {
		CreateLogger()
	}
}
