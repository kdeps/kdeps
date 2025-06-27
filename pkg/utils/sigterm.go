package utils

import (
	"os"
	"syscall"

	"github.com/kdeps/kdeps/pkg/logging"
)

// Process interface for better testability
type Process interface {
	Signal(sig os.Signal) error
}

// Injectable function declarations for better testability
var (
	OsGetpidFunc      = os.Getpid
	OsFindProcessFunc = func(pid int) (Process, error) {
		return os.FindProcess(pid)
	}
)

// sendSigterm sends a SIGTERM signal to the current process.
func SendSigterm(logger *logging.Logger) {
	process, err := OsFindProcessFunc(OsGetpidFunc()) // Get the current process
	if err != nil {
		logger.Fatal("failed to find process", "pid", err)
		return
	}

	// Send SIGTERM to the current process
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		logger.Fatal("failed to send SIGTERM", "pid", err)
		return
	}

	logger.Info("sIGTERM signal sent. Server will shut down gracefully.")
}
