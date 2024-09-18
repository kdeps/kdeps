package logging

import (
	"os"

	"github.com/charmbracelet/log"
)

// Logger instance
var logger *log.Logger

func init() {
	// Create a new logger instance
	logger = log.New(os.Stderr)

	// Set the log level based on the DEBUG environment variable
	if os.Getenv("DEBUG") == "1" {
		logger.SetLevel(log.DebugLevel)
	} else {
		logger.SetLevel(log.InfoLevel)
	}
}

// Debug logs debug messages if debug logging is enabled.
func Debug(msg interface{}, keyvals ...interface{}) {
	logger.Debug(msg, keyvals...)
}

// Info logs informational messages.
func Info(msg interface{}, keyvals ...interface{}) {
	logger.Info(msg, keyvals...)
}

// Warn logs warning messages.
func Warn(msg interface{}, keyvals ...interface{}) {
	logger.Warn(msg, keyvals...)
}

// Error logs error messages.
func Error(msg interface{}, keyvals ...interface{}) {
	logger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and exits the program.
func Fatal(msg interface{}, keyvals ...interface{}) {
	logger.Fatal(msg, keyvals...)
}

// GetLogger returns the Logger instance
func GetLogger() *log.Logger {
	return logger
}
