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
	level := resolveLogLevel(debug, verbose)
	if os.Getenv("KDEPS_LOG_FORMAT") == "json" {
		logger = newJSONLogger(level)
		return
	}
	logger = newPrettyPackageLogger(level, debug)
}

func resolveLogLevel(debug, verbose bool) slog.Level {
	if debug {
		return slog.LevelDebug
	}
	if verbose {
		return slog.LevelInfo
	}
	return slog.LevelWarn
}

func newJSONLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	return slog.New(slog.NewJSONHandler(os.Stderr, opts))
}

func newPrettyPackageLogger(level slog.Level, debug bool) *slog.Logger {
	opts := &logging.PrettyHandlerOptions{
		Level:      level,
		AddSource:  debug,
		TimeFormat: "15:04:05.000",
		Indent:     "  ",
	}
	handler := logging.NewPrettyHandler(os.Stderr, opts)
	return slog.New(handler)
}

func ensure() *slog.Logger {
	if logger == nil {
		Init(false, false)
	}
	return logger
}

// Info logs an info message.
func Info(msg string, args ...any) { ensure().Info(msg, args...) }

// Warn logs a warning message.
func Warn(msg string, args ...any) { ensure().Warn(msg, args...) }

// Error logs an error message.
func Error(msg string, args ...any) { ensure().Error(msg, args...) }
