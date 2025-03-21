package utils

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logger = logging.GetLogger()

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
		require.NoError(t, err)

		// Act
		err = WaitForFileReady(fs, filepath, logger)

		// Assert
		require.NoError(t, err)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		t.Parallel()
		// Arrange
		fs := afero.NewMemMapFs()
		filepath := "/nonexistent.txt"

		// Act - Create the file **earlier** to avoid race condition
		go func() {
			time.Sleep(200 * time.Millisecond) // Give WaitForFileReady some time to run
			_, err := fs.Create(filepath)
			assert.NoError(t, err) // Fail test if file creation fails
		}()

		err := WaitForFileReady(fs, filepath, logger)

		// Assert
		require.NoError(t, err)
	})

	t.Run("ErrorCheckingFile", func(t *testing.T) {
		t.Parallel()
		// Arrange
		fs := &errorFs{Fs: afero.NewMemMapFs()} // Wrap with error-inducing Fs
		filepath := "/cannotcreate.txt"

		// Act
		err := WaitForFileReady(fs, filepath, logger)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error checking file")
	})
}
