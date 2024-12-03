package utils

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func WaitForFileReady(fs afero.Fs, filepath string, logger *log.Logger) error {
	logger.Debug("Waiting for file to be ready...", "file", filepath)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Introduce a timeout
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			// Check if the file exists
			exists, err := afero.Exists(fs, filepath)
			if err != nil {
				return fmt.Errorf("error checking file %s: %w", filepath, err)
			}
			if exists {
				logger.Debug("File is ready!", "file", filepath)
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout waiting for file %s", filepath)
		}
	}
}
