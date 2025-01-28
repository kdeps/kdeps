package utils

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func WaitForFileReady(fs afero.Fs, filepath string, logger *logging.Logger) error {
	logger.Debug("waiting for file to be ready...", "file", filepath)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Introduce a timeout
	timeout := time.After(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			// Check if the file exists
			exists, err := afero.Exists(fs, filepath)
			if err != nil {
				return fmt.Errorf("error checking file %s: %w", filepath, err)
			}

			if exists {
				logger.Debug("file is ready!", "file", filepath)
				return nil
			}

		case <-timeout:
			return fmt.Errorf("timeout waiting for file %s", filepath)
		}
	}
}

// GenerateResourceIDFilename sanitizes a resource ID string to be filename-friendly.
func GenerateResourceIDFilename(input string, requestID string) string {
	// Replace non-filename-friendly characters (@, /, :) with _
	re := regexp.MustCompile(`[@/:]`)
	sanitized := re.ReplaceAllString(input, "_")

	// Remove leading "_" if present
	return strings.TrimPrefix(requestID+sanitized, "_")
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

func CreateFiles(fs afero.Fs, ctx context.Context, files []string) error {
	for _, file := range files {
		// Create the file and any necessary parent directories
		f, err := fs.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}

		// Close the file after creating it to ensure it’s properly written to disk
		err = f.Close()
		if err != nil {
			return fmt.Errorf("failed to close file %s: %w", file, err)
		}
	}
	return nil
}

// Sanitize archive file pathing from "G305: Zip Slip vulnerability".
func SanitizeArchivePath(d, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}
