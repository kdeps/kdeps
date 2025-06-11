package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

	logger := logging.NewTestLogger()

	t.Run("FileExists", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		filepath := "/test/file.txt"

		// Create the file
		afero.WriteFile(fs, filepath, []byte("content"), 0o644)

		err := WaitForFileReady(fs, filepath, logger)
		assert.NoError(t, err)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		filepath := "/test/nonexistent.txt"

		err := WaitForFileReady(fs, filepath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("ErrorCheckingFile", func(t *testing.T) {
		t.Parallel()
		// Use a filesystem that will cause an error when checking file existence
		fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		filepath := "/test/file.txt"

		err := WaitForFileReady(fs, filepath, logger)
		assert.Error(t, err)
	})
}

func TestWaitForFileReadySuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create file after 100ms in goroutine
	filename := "/tmp/success.txt"
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = afero.WriteFile(fs, filename, []byte("data"), 0o644)
	}()

	err := WaitForFileReady(fs, filename, logger)
	assert.NoError(t, err)
}

func TestWaitForFileReadyTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	start := time.Now()
	err := WaitForFileReady(fs, "/not/exist.txt", logger)
	// Should error and take at least 1s due to internal timeout
	assert.Error(t, err)
	assert.GreaterOrEqual(t, time.Since(start), time.Second)
}

func TestGenerateResourceIDFilename(t *testing.T) {
	got := GenerateResourceIDFilename("my@id/file:path", "req-")
	expected := "req-my_id_file_path"
	assert.Equal(t, expected, got)
}

func TestCreateDirectories(t *testing.T) {
	t.Parallel()

	t.Run("CreateSingleDirectory", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		dirs := []string{"/test/dir"}

		err := CreateDirectories(fs, ctx, dirs)
		assert.NoError(t, err)

		exists, _ := afero.DirExists(fs, "/test/dir")
		assert.True(t, exists)
	})

	t.Run("CreateMultipleDirectories", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		dirs := []string{"/test/dir1", "/test/dir2", "/nested/deep/dir"}

		err := CreateDirectories(fs, ctx, dirs)
		assert.NoError(t, err)

		for _, dir := range dirs {
			exists, _ := afero.DirExists(fs, dir)
			assert.True(t, exists)
		}
	})

	t.Run("EmptyDirectoriesList", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		dirs := []string{}

		err := CreateDirectories(fs, ctx, dirs)
		assert.NoError(t, err)
	})

	t.Run("FailToCreateDirectory", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		ctx := context.Background()
		dirs := []string{"/test/dir"}

		err := CreateDirectories(fs, ctx, dirs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})
}

func TestCreateFiles(t *testing.T) {
	t.Parallel()

	t.Run("CreateSingleFile", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		files := []string{"/test/file.txt"}

		err := CreateFiles(fs, ctx, files)
		assert.NoError(t, err)

		exists, _ := afero.Exists(fs, "/test/file.txt")
		assert.True(t, exists)
	})

	t.Run("CreateMultipleFiles", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		files := []string{"/test/file1.txt", "/test/file2.txt", "/nested/file.txt"}

		err := CreateFiles(fs, ctx, files)
		assert.NoError(t, err)

		for _, file := range files {
			exists, _ := afero.Exists(fs, file)
			assert.True(t, exists)
		}
	})

	t.Run("EmptyFilesList", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewMemMapFs()
		ctx := context.Background()
		files := []string{}

		err := CreateFiles(fs, ctx, files)
		assert.NoError(t, err)
	})

	t.Run("FailToCreateFile", func(t *testing.T) {
		t.Parallel()
		fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		ctx := context.Background()
		files := []string{"/test/file.txt"}

		err := CreateFiles(fs, ctx, files)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create file")
	})
}

func TestCreateDirectoriesAndFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	dirs := []string{"/a/b/c", "/a/b/d"}
	files := []string{"/a/b/c/file1", "/a/b/d/file2"}

	assert.NoError(t, CreateDirectories(fs, ctx, dirs))
	assert.NoError(t, CreateFiles(fs, ctx, files))

	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		assert.True(t, exists)
	}
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		assert.True(t, exists)
	}
}

func TestSanitizeArchivePath(t *testing.T) {
	base := "/home/user"
	good, err := SanitizeArchivePath(base, "sub/file.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "sub/file.txt"), good)

	bad, err := SanitizeArchivePath(base, "../../evil.txt")
	assert.Error(t, err)
	assert.Empty(t, bad)
}
