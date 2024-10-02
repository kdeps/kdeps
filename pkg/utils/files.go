package utils

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func WaitForFileReady(fs afero.Fs, filepath string, logger *log.Logger) error {
	logger.Info("Waiting for file: ", filepath)

	// Create a ticker that checks for the file periodically
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if the file exists
			exists, err := afero.Exists(fs, filepath)
			if err != nil {
				return fmt.Errorf("error checking file %s: %w", filepath, err)
			}
			if exists {
				logger.Info("File found: ", filepath)
				return nil
			}
		}
	}

	return nil
}
