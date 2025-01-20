package utils

import (
	"os"
	"syscall"

	"github.com/kdeps/kdeps/pkg/logging"
)

// sendSigterm sends a SIGTERM signal to the current process.
func SendSigterm(logger *logging.Logger) {
	process, err := os.FindProcess(os.Getpid()) // Get the current process
	if err != nil {
		logger.Fatal("Failed to find process", "pid", err)
	}

	// Send SIGTERM to the current process
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		logger.Fatal("Failed to send SIGTERM", "pid", err)
	}

	logger.Info("SIGTERM signal sent. Server will shut down gracefully.")
}
