package utils

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/spf13/afero"
)

const (
	dirPerm         = 0o755
	fileReadyTicker = 500 * time.Millisecond
)

func WaitForFileReady(fs afero.Fs, filepath string, logger *logging.Logger) error {
	logger.Debug(messages.MsgWaitingForFileReady, "file", filepath)

	ticker := time.NewTicker(fileReadyTicker)
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
				logger.Debug(messages.MsgFileIsReady, "file", filepath)
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

func CreateDirectories(_ context.Context, fs afero.Fs, dirs []string) error {
	for _, dir := range dirs {
		// Use fs.MkdirAll to create the directory and its parents if they don't exist
		err := fs.MkdirAll(dir, dirPerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func CreateFiles(_ context.Context, fs afero.Fs, files []string) error {
	for _, file := range files {
		// Create the file and any necessary parent directories
		f, err := fs.Create(file)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", file, err)
		}

		// Close the file after creating it to ensure it's properly written to disk
		err = f.Close()
		if err != nil {
			return fmt.Errorf("failed to close file %s: %w", file, err)
		}
	}
	return nil
}

// SanitizeArchivePath sanitizes archive file pathing from "G305: Zip Slip vulnerability".
func SanitizeArchivePath(base, target string) (string, error) {
	v := filepath.Join(base, target)
	if strings.HasPrefix(v, filepath.Clean(base)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", target)
}
