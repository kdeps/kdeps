package utils

import (
	"os"
	"syscall"
)

// SendSigterm sends a SIGTERM signal to the current process.
func SendSigterm() error {
	process, err := os.FindProcess(os.Getpid()) // Get the current process
	if err != nil {
		return err
	}

	// Send SIGTERM to the current process
	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	return nil
}
