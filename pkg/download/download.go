package download

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/dustin/go-humanize"
	"github.com/spf13/afero"
)

// WriteCounter tracks the total number of bytes written and prints download progress.
type WriteCounter struct {
	Total uint64
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
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// DownloadFile downloads a file from the specified URL and saves it to the given path.
func DownloadFile(fs afero.Fs, url, filePath string, logger *log.Logger) error {
	logger.Info("Starting file download", "url", url, "destination", filePath)

	tmpFilePath := filePath + ".tmp"

	// Create a temporary file
	out, err := fs.Create(tmpFilePath)
	if err != nil {
		logger.Error("Failed to create temporary file", "file-path", tmpFilePath, "error", err)
		return fmt.Errorf("failed to create temporary file: %w", err)
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
		return fmt.Errorf(errMsg)
	}

	// Create a WriteCounter to track and display download progress
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		logger.Error("Failed to copy data", "error", err)
		return fmt.Errorf("failed to copy data: %w", err)
	}

	logger.Info("Download complete", "url", url, "file-path", filePath)

	// Rename the temporary file to the final destination
	if err = fs.Rename(tmpFilePath, filePath); err != nil {
		logger.Error("Failed to rename temporary file", "tmp-file-path", tmpFilePath, "file-path", filePath, "error", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}
