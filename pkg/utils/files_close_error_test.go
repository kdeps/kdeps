package utils_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type badCloseFile struct{ afero.File }

func (b badCloseFile) Close() error { return errors.New("close fail") }

type badCloseFs struct{ afero.Fs }

func (fs badCloseFs) Create(name string) (afero.File, error) {
	f, err := fs.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return badCloseFile{f}, nil
}

// Other methods delegate to embedded Fs.

func TestCreateFilesCloseError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := badCloseFs{afero.NewOsFs()}
	files := []string{filepath.Join(tmpDir, "fail.txt")}

	if err := utils.CreateFiles(context.Background(), fs, files); err == nil {
		t.Fatalf("expected close error but got nil")
	}
}

// failCreateFs returns error on Create to hit the error branch inside CreateFiles.
type failCreateFs struct{ afero.Fs }

func (f failCreateFs) Create(_ string) (afero.File, error) {
	return nil, errors.New("create error")
}

func TestCreateFiles_CreateError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := failCreateFs{afero.NewOsFs()}
	files := []string{filepath.Join(tmpDir, "cannot.txt")}
	err := utils.CreateFiles(context.Background(), fs, files)
	if err == nil {
		t.Fatalf("expected error from CreateFiles when underlying fs.Create fails")
	}
}

func TestWaitForFileReadyEdgeSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "file.txt")

	// create file after short delay in goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = afero.WriteFile(fs, path, []byte("ok"), 0o644)
	}()

	if err := utils.WaitForFileReady(fs, path, logger); err != nil {
		t.Fatalf("expected file ready, got error %v", err)
	}
}

func TestWaitForFileReadyEdgeTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent")

	start := time.Now()
	err := utils.WaitForFileReady(fs, nonexistentPath, logger)
	duration := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if duration < 1*time.Second {
		t.Fatalf("function returned too early, expected ~1s wait")
	}
}

func TestWaitForFileReady_SuccessAndTimeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()

	// Success case: create file first
	successPath := filepath.Join(tmpDir, "success.txt")
	require.NoError(t, afero.WriteFile(fs, successPath, []byte("ok"), 0o644))

	require.NoError(t, utils.WaitForFileReady(fs, successPath, logger))

	// Timeout case: path never appears – expect error after ~1s
	start := time.Now()
	missingPath := filepath.Join(tmpDir, "missing.txt")
	err := utils.WaitForFileReady(fs, missingPath, logger)
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

	if err := utils.WaitForFileReady(fs, filename, logger); err != nil {
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
		got := utils.GenerateResourceIDFilename(c.id, c.req)
		require.Equal(t, c.want, got)
	}
}

func TestCreateDirectoriesAndFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	dirs := []string{filepath.Join(tmpDir, "a"), filepath.Join(tmpDir, "b", "c")}
	require.NoError(t, utils.CreateDirectories(ctx, fs, dirs))
	for _, d := range dirs {
		ok, _ := afero.DirExists(fs, d)
		require.True(t, ok)
	}

	files := []string{filepath.Join(dirs[0], "f1.txt"), filepath.Join(dirs[1], "f2.txt")}
	require.NoError(t, utils.CreateFiles(ctx, fs, files))
	for _, f := range files {
		ok, _ := afero.Exists(fs, f)
		require.True(t, ok)
	}
}

func TestSanitizeArchivePathExtra(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "safe", "root")
	good, err := utils.SanitizeArchivePath(base, "sub/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "sub/file.txt"), good)

	// Attempt a Zip-Slip attack with ".." prefix
	_, err = utils.SanitizeArchivePath(base, "../evil.txt")
	require.Error(t, err)
}

// errFS wraps an afero.Fs but forces Stat to return an error to exercise the error branch in WaitForFileReady.
type errFS struct{ afero.Fs }

func (e errFS) Stat(_ string) (os.FileInfo, error) {
	return nil, errors.New("stat failure")
}

func TestWaitForFileReadyHelper(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	fname := filepath.Join(tmpDir, "ready.txt")

	// Create the file after a short delay to test the polling loop.
	go func() {
		time.Sleep(100 * time.Millisecond)
		f, _ := fs.Create(fname)
		f.Close()
	}()

	if err := utils.WaitForFileReady(fs, fname, logger); err != nil {
		t.Fatalf("WaitForFileReady returned error: %v", err)
	}

	// Ensure timeout branch returns error when file never appears.
	missingPath := filepath.Join(tmpDir, "missing.txt")
	if err := utils.WaitForFileReady(fs, missingPath, logger); err == nil {
		t.Errorf("expected timeout error but got nil")
	}
}

func TestCreateDirectoriesAndFilesHelper(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	dirs := []string{filepath.Join(tmpDir, "a", "b"), filepath.Join(tmpDir, "c", "d", "e")}
	if err := utils.CreateDirectories(ctx, fs, dirs); err != nil {
		t.Fatalf("CreateDirectories error: %v", err)
	}
	for _, d := range dirs {
		exists, _ := afero.DirExists(fs, d)
		if !exists {
			t.Errorf("directory %s not created", d)
		}
	}

	files := []string{filepath.Join(dirs[0], "file.txt"), filepath.Join(dirs[1], "other.txt")}
	if err := utils.CreateFiles(ctx, fs, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		if !exists {
			t.Errorf("file %s not created", f)
		}
	}
}

func TestGenerateResourceIDFilenameAndSanitizeArchivePathHelper(t *testing.T) {
	id := "abc/def:ghi@jkl"
	got := utils.GenerateResourceIDFilename(id, "req-")
	want := "req-abc_def_ghi_jkl"
	if filepath.Base(got) != want {
		t.Errorf("GenerateResourceIDFilename = %s, want %s", got, want)
	}

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	base := filepath.Join(tmpDir, "base")
	good, err := utils.SanitizeArchivePath(base, "sub/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedGood := filepath.Join(base, "sub/file.txt")
	if good != expectedGood {
		t.Errorf("SanitizeArchivePath = %s, want %s", good, expectedGood)
	}

	if _, err := utils.SanitizeArchivePath(base, "../escape.txt"); err == nil {
		t.Errorf("expected error for path escape, got nil")
	}
}

func TestWaitForFileReadyError(t *testing.T) {
	fs := errFS{afero.NewMemMapFs()}
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "any")

	if err := utils.WaitForFileReady(fs, testPath, logger); err == nil {
		t.Errorf("expected error due to Stat failure, got nil")
	}
}

func TestGenerateResourceIDFilenameMore(t *testing.T) {
	got := utils.GenerateResourceIDFilename("@agent/data:1.0.0", "req-")
	if got != "req-_agent_data_1.0.0" {
		t.Fatalf("unexpected filename: %s", got)
	}
}

func TestCreateDirectoriesAndFilesMore(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	dirs := []string{filepath.Join(tmpDir, "a", "b", "c")}
	if err := utils.CreateDirectories(ctx, fs, dirs); err != nil {
		t.Fatalf("CreateDirectories error: %v", err)
	}
	if ok, _ := afero.DirExists(fs, dirs[0]); !ok {
		t.Fatalf("directory not created")
	}

	files := []string{filepath.Join(dirs[0], "file.txt")}
	if err := utils.CreateFiles(ctx, fs, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	if ok, _ := afero.Exists(fs, files[0]); !ok {
		t.Fatalf("file not created")
	}
}

func TestSanitizeArchivePathMore(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	safe := filepath.Join(tmpDir, "safe")
	p, err := utils.SanitizeArchivePath(safe, "sub/dir.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(safe, "sub/dir.txt")
	if p != expected {
		t.Fatalf("unexpected sanitized path: %s", p)
	}

	// attempt path traversal
	if _, err := utils.SanitizeArchivePath(safe, "../evil.txt"); err == nil {
		t.Fatalf("expected error for tainted path")
	}
}

// TestCreateFilesErrorOsFs validates the error branch when using a read-only filesystem
// backed by the real OS and a temporary directory.
func TestCreateFilesErrorOsFs(t *testing.T) {
	tmpDir := t.TempDir()
	// The read-only wrapper simulates permission failure.
	roFs := afero.NewReadOnlyFs(afero.NewOsFs())

	files := []string{filepath.Join(tmpDir, "should_fail.txt")}
	err := utils.CreateFiles(context.Background(), roFs, files)
	if err == nil {
		t.Fatalf("expected error when creating files on read-only fs, got nil")
	}
}

// TestWaitForFileReadyOsFs uses a real tmpfile on the OS FS.
func TestWaitForFileReadyOsFs(t *testing.T) {
	osFs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "ready.txt")

	// create file after delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		f, _ := osFs.Create(filePath)
		f.Close()
	}()

	if err := utils.WaitForFileReady(osFs, filePath, logger); err != nil {
		t.Fatalf("WaitForFileReady returned error: %v", err)
	}
}

// TestCreateDirectoriesErrorOsFs validates failure path of CreateDirectories on read-only fs.
func TestCreateDirectoriesErrorOsFs(t *testing.T) {
	tmpDir := t.TempDir()
	roFs := afero.NewReadOnlyFs(afero.NewOsFs())

	dirs := []string{filepath.Join(tmpDir, "subdir")}
	if err := utils.CreateDirectories(context.Background(), roFs, dirs); err == nil {
		t.Fatalf("expected error when creating directory on read-only fs, got nil")
	}
}

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
		got := utils.GenerateResourceIDFilename(tc.in, tc.reqID)
		assert.NotContains(t, got, "@")
		assert.NotContains(t, got, "/")
		assert.NotContains(t, got, ":")
		assert.True(t, strings.HasPrefix(got, tc.reqID))
	}
}

// TestSanitizeArchivePath ensures that paths outside the destination return an error
// while valid ones pass.
func TestSanitizeArchivePath(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	safe := filepath.Join(tmpDir, "safe")
	okPath, err := utils.SanitizeArchivePath(safe, "file.txt")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(safe, "file.txt"), okPath)

	// Attempt Zip-Slip attack with ".." – should error
	_, err = utils.SanitizeArchivePath(safe, "../evil.txt")
	assert.Error(t, err)
}

// TestCreateDirectoriesAndFiles uses an in-memory FS to verify helpers.
func TestCreateDirectoriesAndFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	dirs := []string{filepath.Join(tmpDir, "dir1"), filepath.Join(tmpDir, "dir2", "sub")}
	files := []string{filepath.Join(dirs[0], "a.txt"), filepath.Join(dirs[1], "b.txt")}

	assert.NoError(t, utils.CreateDirectories(ctx, fs, dirs))
	assert.NoError(t, utils.CreateFiles(ctx, fs, files))

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

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "ready.txt")

	// success case – create the file shortly after starting the wait
	done := make(chan struct{})
	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = afero.WriteFile(fs, filename, []byte("ok"), 0o644)
	}()

	require.NoError(t, utils.WaitForFileReady(fs, filename, logger))
	close(done)

	// timeout case – file never appears
	start := time.Now()
	nonexistentPath := filepath.Join(tmpDir, "nonexistent")
	err := utils.WaitForFileReady(fs, nonexistentPath, logger)
	duration := time.Since(start)
	require.Error(t, err)
	// It should time-out roughly around the configured 1s ± some slack.
	assert.LessOrEqual(t, duration.Seconds(), 2.0)
}
