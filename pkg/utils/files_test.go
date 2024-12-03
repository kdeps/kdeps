package utils

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type errorFs struct {
	afero.Fs
}

func (e *errorFs) Stat(name string) (os.FileInfo, error) {
	return nil, errors.New("simulated error checking file")
}

func TestWaitForFileReady(t *testing.T) {
	t.Run("FileExists", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		logger := log.New(os.Stderr)
		filepath := "/testfile.txt"

		// Create the file in the in-memory filesystem
		_, err := fs.Create(filepath)
		assert.NoError(t, err)

		// Act
		err = WaitForFileReady(fs, filepath, logger)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		// Arrange
		fs := afero.NewMemMapFs()
		logger := log.New(os.Stderr)
		filepath := "/nonexistent.txt"

		// Act
		go func() {
			time.Sleep(1 * time.Second)
			fs.Create(filepath) // Create the file after a delay
		}()
		err := WaitForFileReady(fs, filepath, logger)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ErrorCheckingFile", func(t *testing.T) {
		// Arrange
		fs := &errorFs{Fs: afero.NewMemMapFs()} // Wrap with error-inducing Fs
		logger := log.New(os.Stderr)
		filepath := "/cannotcreate.txt"

		// Act
		err := WaitForFileReady(fs, filepath, logger)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error checking file")
	})
}
