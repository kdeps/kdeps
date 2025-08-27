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
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/spf13/afero"
)

// WriteCounter tracks the total number of bytes written and prints download progress.
type WriteCounter struct {
	Total         uint64
	Expected      uint64
	LocalFilePath string
	DownloadURL   string
	ItemName      string
	IsCache       bool
}

type DownloadItem struct {
	URL       string
	LocalName string
}

// Write implements the io.Writer interface and updates the total byte count.
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress displays the download progress in the terminal.
func (wc *WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 80)) // Clear the line with more space
	
	// Choose appropriate icon and message based on context
	icon := "📥"
	prefix := "Downloading"
	if wc.IsCache {
		icon = "🔄"
		prefix = "Caching"
	}
	
	// Use item name if available, otherwise show URL
	name := wc.ItemName
	if name == "" {
		name = filepath.Base(wc.DownloadURL)
	}
	
	// Show progress with percentage if expected size is known
	if wc.Expected > 0 {
		percent := float64(wc.Total) / float64(wc.Expected) * 100
		fmt.Printf("\r%s %s %s - %s/%s (%.1f%%)", icon, prefix, name, 
			humanize.Bytes(wc.Total), humanize.Bytes(wc.Expected), percent)
	} else {
		fmt.Printf("\r%s %s %s - %s", icon, prefix, name, humanize.Bytes(wc.Total))
	}
}

// Given a list of URLs, download it to a target.
func DownloadFiles(fs afero.Fs, ctx context.Context, downloadDir string, items []DownloadItem, logger *logging.Logger, useLatest bool) error {
	// Create the downloads directory if it doesn't exist
	err := os.MkdirAll(downloadDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	for _, item := range items {
		localPath := filepath.Join(downloadDir, item.LocalName)

		// If using "latest", remove any existing file to avoid stale downloads
		if useLatest {
			if err := fs.Remove(localPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				logger.Warn("failed to remove existing file before re-downloading", "path", localPath, "err", err)
			} else if err == nil {
				logger.Debug(messages.MsgRemovedExistingLatestFile, "path", localPath)
			}
		}

		// Download the file
		err := DownloadFile(fs, ctx, item.URL, localPath, logger, useLatest)
		if err != nil {
			logger.Error("failed to download", "url", item.URL, "err", err)
		} else {
			logger.Info("successfully downloaded", "url", item.URL, "path", localPath)
		}
	}

	return nil
}

// DownloadFile downloads a file from the specified URL and saves it to the given path.
// If useLatest is true, it overwrites the destination file regardless of its existence.
func DownloadFile(fs afero.Fs, ctx context.Context, url, filePath string, logger *logging.Logger, useLatest bool) error {
	logger.Debug(messages.MsgCheckingFileExistsDownload, "destination", filePath)

	if filePath == "" {
		logger.Error("invalid file path provided", "file-path", filePath)
		return fmt.Errorf("invalid file path: %s", filePath)
	}

	// Skip the existence check if useLatest is true
	if !useLatest {
		// Check if the file already exists
		if exists, err := afero.Exists(fs, filePath); err != nil {
			logger.Error("error checking file existence", "file-path", filePath, "error", err)
			return fmt.Errorf("error checking file existence: %w", err)
		} else if exists {
			// Check if the file is non-empty
			info, err := fs.Stat(filePath)
			if err != nil {
				logger.Error("failed to stat file", "file-path", filePath, "error", err)
				return fmt.Errorf("failed to stat file: %w", err)
			}
			if info.Size() > 0 {
				logger.Debug(messages.MsgFileAlreadyExistsSkipping, "file-path", filePath)
				return nil
			}
		}
	}

	logger.Debug(messages.MsgStartingFileDownload, "url", url, "destination", filePath)

	tmpFilePath := filePath + ".tmp"

	// Create a temporary file
	out, err := fs.Create(tmpFilePath)
	if err != nil {
		logger.Error("failed to create temporary file", "file-path", tmpFilePath, "error", err)
		return fmt.Errorf("failed to create temporary file '%s': %w", tmpFilePath, err)
	}
	defer out.Close()

	// Perform the HTTP GET request
	resp, err := MakeGetRequest(ctx, url)
	if err != nil {
		logger.Error("failed to download file", "url", url, "error", err)
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
		logger.Error("failed to copy data", "error", err)
		return fmt.Errorf("failed to copy data: %w", err)
	}

	logger.Debug(messages.MsgDownloadComplete, "url", url, "file-path", filePath)

	// Rename the temporary file to the final destination
	if err = fs.Rename(tmpFilePath, filePath); err != nil {
		logger.Error("failed to rename temporary file", "tmp-file-path", tmpFilePath, "file-path", filePath, "error", err)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func MakeGetRequest(ctx context.Context, uri string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return res, err
	}

	return res, nil
}
