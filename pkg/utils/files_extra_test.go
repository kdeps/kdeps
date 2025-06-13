package utils

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestWaitForFileReady_SuccessAndTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Success case: create file first
	successPath := "/tmp/success.txt"
	require.NoError(t, afero.WriteFile(fs, successPath, []byte("ok"), 0o644))

	require.NoError(t, WaitForFileReady(fs, successPath, logger))

	// Timeout case: path never appears – expect error after ~1s
	start := time.Now()
	err := WaitForFileReady(fs, "/tmp/missing.txt", logger)
	require.Error(t, err)
	// Ensure we did wait at least ~1s but not much longer (sanity)
	require.GreaterOrEqual(t, time.Since(start), time.Second)
	require.Less(t, time.Since(start), 1500*time.Millisecond)
}

func TestWaitForFileReady_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	filename := "ready.txt"

	// create file after small delay in goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		afero.WriteFile(fs, filename, []byte("ok"), 0o644)
	}()

	if err := WaitForFileReady(fs, filename, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateResourceIDFilenameExtra(t *testing.T) {
	cases := []struct {
		id   string
		req  string
		want string
	}{
		{"my@id/with:chars", "abc-", "abc-my_id_with_chars"},
		{"simple", "req", "reqsimple"},
		{"/leading", "r-", "r-_leading"},
	}
	for _, c := range cases {
		got := GenerateResourceIDFilename(c.id, c.req)
		require.Equal(t, c.want, got)
	}
}

func TestCreateDirectoriesAndFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	dirs := []string{"/tmp/a", "/tmp/b/c"}
	require.NoError(t, CreateDirectories(fs, ctx, dirs))
	for _, d := range dirs {
		ok, _ := afero.DirExists(fs, d)
		require.True(t, ok)
	}

	files := []string{filepath.Join(dirs[0], "f1.txt"), filepath.Join(dirs[1], "f2.txt")}
	require.NoError(t, CreateFiles(fs, ctx, files))
	for _, f := range files {
		ok, _ := afero.Exists(fs, f)
		require.True(t, ok)
	}
}

func TestSanitizeArchivePathExtra(t *testing.T) {
	base := "/safe/root"
	good, err := SanitizeArchivePath(base, "sub/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "sub/file.txt"), good)

	// Attempt a Zip-Slip attack with ".." prefix
	_, err = SanitizeArchivePath(base, "../evil.txt")
	require.Error(t, err)
}
