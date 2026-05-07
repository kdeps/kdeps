// Package log provides structured JSON-capable logging for kdeps.
// It wraps log/slog with a package-level logger that supports:
//   - Log levels: DEBUG, INFO, WARN, ERROR
//   - Output formats: pretty (terminal) or JSON (production)
//   - Level control via CLI flags (--debug, --verbose) and env vars
package log

import (
	"log/slog"
	"os"

	"github.com/kdeps/kdeps/v2/pkg/infra/logging"
)

//nolint:gochecknoglobals // package-level logger
var logger *slog.Logger

// Init initializes the package-level logger. Call once at startup.
// debug sets level to DEBUG. verbose sets level to INFO.
// Neither set defaults to WARN (only warnings and errors shown).
func Init(debug, verbose bool) {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelInfo
	}
	if debug {
		level = slog.LevelDebug
	}

	format := os.Getenv("KDEPS_LOG_FORMAT")
	if format == "json" {
		opts := &slog.HandlerOptions{Level: level}
		logger = slog.New(slog.NewJSONHandler(os.Stderr, opts))
		return
	}

	opts := &logging.PrettyHandlerOptions{
		Level:      level,
		AddSource:  debug,
		TimeFormat: "15:04:05.000",
		Indent:     "  ",
	}
	handler := logging.NewPrettyHandler(os.Stderr, opts)
	logger = slog.New(handler)
}

func ensure() *slog.Logger {
	if logger == nil {
		Init(false, false)
	}
	return logger
}

// Debug logs a debug message.
func Debug(msg string, args ...any) { ensure().Debug(msg, args...) }

// Info logs an info message.
func Info(msg string, args ...any) { ensure().Info(msg, args...) }

// Warn logs a warning message.
func Warn(msg string, args ...any) { ensure().Warn(msg, args...) }

// Error logs an error message.
func Error(msg string, args ...any) { ensure().Error(msg, args...) }

// Logger returns the underlying slog.Logger for direct use.
func Logger() *slog.Logger { return ensure() }
