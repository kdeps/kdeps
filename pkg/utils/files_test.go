package utils

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var logger = logging.GetLogger()

var ctx context.Context

type errorFs struct {
	afero.Fs
}

func (e *errorFs) Stat(name string) (os.FileInfo, error) {
	return nil, errors.New("simulated error checking file")
}

func TestWaitForFileReady(t *testing.T) {
	t.Parallel()
	t.Run("FileExists", func(t *testing.T) {
		t.Parallel()
		// Arrange
		fs := afero.NewMemMapFs()
		filepath := "/testfile.txt"

		// Create the file in the in-memory filesystem
		_, err := fs.Create(filepath)
		assert.NoError(t, err)

		// Act
		err = WaitForFileReady(fs, ctx, filepath, logger)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		t.Parallel()
		// Arrange
		fs := afero.NewMemMapFs()
		filepath := "/nonexistent.txt"

		// Act
		go func() {
			time.Sleep(1 * time.Second)
			fs.Create(filepath) // Create the file after a delay
		}()
		err := WaitForFileReady(fs, ctx, filepath, logger)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("ErrorCheckingFile", func(t *testing.T) {
		t.Parallel()
		// Arrange
		fs := &errorFs{Fs: afero.NewMemMapFs()} // Wrap with error-inducing Fs
		filepath := "/cannotcreate.txt"

		// Act
		err := WaitForFileReady(fs, ctx, filepath, logger)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error checking file")
	})
}
