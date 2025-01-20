package utils

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func WaitForFileReady(fs afero.Fs, ctx context.Context, filepath string, logger *logging.Logger) error {
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

// ConvertToFilenameFriendly sanitizes a resource ID string to be filename-friendly.
func ConvertToFilenameFriendly(input string) string {
	// Replace non-filename-friendly characters (@, /, :) with _
	re := regexp.MustCompile(`[@/:]`)
	sanitized := re.ReplaceAllString(input, "_")

	// Remove leading "_" if present
	return strings.TrimPrefix(sanitized, "_")
}

func CreateDirectories(fs afero.Fs, ctx context.Context, dirs []string) error {
	for _, dir := range dirs {
		// Use fs.MkdirAll to create the directory and its parents if they don't exist
		err := fs.MkdirAll(dir, 0o755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
