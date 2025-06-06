package utils

import (
	"context"
	"errors"
	"os"
	"testing"

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

func TestGenerateResourceIDFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		requestID string
		expected  string
	}{
		{"SimpleInput", "resource", "req123", "req123resource"},
		{"WithSpecialChars", "user@domain.com:/path", "req123", "req123user_domain.com__path"},
		{"OnlySpecialChars", "@/:", "req123", "req123___"},
		{"EmptyInput", "", "req123", "req123"},
		{"EmptyRequestID", "resource", "", "resource"},
		{"LeadingUnderscore", "_resource", "req123_", "req123__resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := GenerateResourceIDFilename(tt.input, tt.requestID)
			assert.Equal(t, tt.expected, result)
		})
	}
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

func TestSanitizeArchivePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		destination string
		target      string
		expectError bool
		expected    string
	}{
		{"ValidPath", "/safe/dir", "file.txt", false, "/safe/dir/file.txt"},
		{"ValidNestedPath", "/safe/dir", "subdir/file.txt", false, "/safe/dir/subdir/file.txt"},
		{"ZipSlipAttack", "/safe/dir", "../../../etc/passwd", true, ""},
		{"ZipSlipWithDots", "/safe/dir", "../../malicious.txt", true, ""},
		{"AbsolutePath", "/safe/dir", "/etc/passwd", false, "/safe/dir/etc/passwd"},
		{"EmptyTarget", "/safe/dir", "", false, "/safe/dir"},
		{"DotPath", "/safe/dir", ".", false, "/safe/dir"},
		{"DoubleDotPath", "/safe/dir", "..", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := SanitizeArchivePath(tt.destination, tt.target)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "content filepath is tainted")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
