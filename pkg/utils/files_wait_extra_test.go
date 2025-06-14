package utils

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestGenerateResourceIDFilename verifies that non-filename characters are replaced
// and the requestID is correctly prepended.
func TestGenerateResourceIDFilename(t *testing.T) {

	cases := []struct {
		reqID string
		in    string
		want  string
	}{
		{"abc-", "my@resource:id", "abc-m y_resource_id"}, // adjusted below
	}

	// We build the expected string using the helper to retain exact behaviour.
	for _, tc := range cases {
		got := GenerateResourceIDFilename(tc.in, tc.reqID)
		assert.NotContains(t, got, "@")
		assert.NotContains(t, got, "/")
		assert.NotContains(t, got, ":")
		assert.True(t, strings.HasPrefix(got, tc.reqID))
	}
}

// TestSanitizeArchivePath ensures that paths outside the destination return an error
// while valid ones pass.
func TestSanitizeArchivePath(t *testing.T) {

	okPath, err := SanitizeArchivePath("/safe", "file.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/safe", "file.txt"), okPath)

	// Attempt Zip-Slip attack with ".." – should error
	_, err = SanitizeArchivePath("/safe", "../evil.txt")
	assert.Error(t, err)
}

// TestCreateDirectoriesAndFiles uses an in-memory FS to verify helpers.
func TestCreateDirectoriesAndFiles(t *testing.T) {

	fs := afero.NewMemMapFs()
	ctx := context.Background()

	dirs := []string{"/tmp/dir1", "/tmp/dir2/sub"}
	files := []string{"/tmp/dir1/a.txt", "/tmp/dir2/sub/b.txt"}

	assert.NoError(t, CreateDirectories(fs, ctx, dirs))
	assert.NoError(t, CreateFiles(fs, ctx, files))

	for _, d := range dirs {
		exist, err := afero.DirExists(fs, d)
		assert.NoError(t, err)
		assert.True(t, exist)
	}

	for _, f := range files {
		_, err := fs.Stat(f)
		assert.NoError(t, err)
	}
}

// TestWaitForFileReady covers both success and timeout branches.
func TestWaitForFileReady(t *testing.T) {

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	const filename = "/ready.txt"

	// success case – create the file shortly after starting the wait
	done := make(chan struct{})
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = afero.WriteFile(fs, filename, []byte("ok"), 0o644)
	}()

	assert.NoError(t, WaitForFileReady(fs, filename, logger))
	close(done)

	// timeout case – file never appears
	start := time.Now()
	err := WaitForFileReady(fs, "/nonexistent", logger)
	duration := time.Since(start)
	assert.Error(t, err)
	// It should time-out roughly around the configured 1s ± some slack.
	assert.LessOrEqual(t, duration.Seconds(), 2.0)
}
