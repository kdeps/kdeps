package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// WriteCounter tracks the total number of bytes written and prints download progress.
type WriteCounter struct {
	Total         uint64
	LocalFilePath string
	DownloadURL   string
}

// Write implements the io.Writer interface and updates the total byte count.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress displays the download progress in the terminal.
func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 50)) // Clear the line
	fmt.Printf("\rDownloading %s - %s complete ", wc.DownloadURL, humanize.Bytes(wc.Total))
}

// Given a list of URLs, download it to a target.
func DownloadFiles(fs afero.Fs, ctx context.Context, downloadDir string, urls []string, logger *logging.Logger) error {
	// Create the downloads directory if it doesn't exist
	err := os.MkdirAll(downloadDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Iterate over each URL
	for _, url := range urls {
		// Extract the file name from the URL
		fileName := filepath.Base(url)

		// Define the local path to save the file
		localPath := filepath.Join(downloadDir, fileName)

		// Download the file
		err := DownloadFile(fs, ctx, url, localPath, logger)
		if err != nil {
			logger.Error("Failed to download", "url", url, "err", err)
		} else {
			logger.Info("Successfully downloaded", "url", url, "path", localPath)
		}
	}

	return nil
}

// DownloadFile downloads a file from the specified URL and saves it to the given path.
// It skips the download if the file already exists and is non-empty.
func DownloadFile(fs afero.Fs, ctx context.Context, url, filePath string, logger *logging.Logger) error {
	logger.Debug("Checking if file exists", "destination", filePath)

	if filePath == "" {
		logger.Error("Invalid file path provided", "file-path", filePath)
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// Check if the file already exists
	if exists, err := afero.Exists(fs, filePath); err != nil {
		logger.Error("Error checking file existence", "file-path", filePath, "error", err)
		return fmt.Errorf("error checking file existence: %w", err)
	} else if exists {
		// Check if the file is non-empty
		info, err := fs.Stat(filePath)
		if err != nil {
			logger.Error("Failed to stat file", "file-path", filePath, "error", err)
			return fmt.Errorf("failed to stat file: %w", err)
		}
		if info.Size() > 0 {
			logger.Debug("File already exists and is non-empty, skipping download", "file-path", filePath)
			return nil
		}
	}

	logger.Debug("Starting file download", "url", url, "destination", filePath)

	tmpFilePath := filePath + ".tmp"

	// Create a temporary file
	out, err := fs.Create(tmpFilePath)
	if err != nil {
		logger.Error("Failed to create temporary file", "file-path", tmpFilePath, "error", err)
		return fmt.Errorf("failed to create temporary file '%s': %w", tmpFilePath, err)
	}
	defer out.Close()

	// Perform the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Failed to download file", "url", url, "error", err)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("failed to download file: status code %d", resp.StatusCode)
		logger.Error(errMsg, "url", url)
		return errors.New(errMsg)
	}

	// Create a WriteCounter to track and display download progress
	counter := &WriteCounter{
		LocalFilePath: filePath,
		DownloadURL:   url,
	}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		logger.Error("Failed to copy data", "error", err)
		return fmt.Errorf("failed to copy data: %w", err)
	}

	logger.Debug("Download complete", "url", url, "file-path", filePath)

	// Rename the temporary file to the final destination
	if err = fs.Rename(tmpFilePath, filePath); err != nil {
		logger.Error("Failed to rename temporary file", "tmp-file-path", tmpFilePath, "file-path", filePath, "error", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}
