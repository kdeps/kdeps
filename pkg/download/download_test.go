package download

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	logger *logging.Logger
	ctx    = context.Background()
)

func TestWriteCounter_Write(t *testing.T) {

	counter := &WriteCounter{}
	data := []byte("Hello, World!")
	n, err := counter.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, uint64(len(data)), counter.Total)
}

func TestWriteCounter_PrintProgress(t *testing.T) {

	counter := &WriteCounter{
		DownloadURL: "example.com/file.txt",
	}
	counter.Total = 1024

	expectedOutput := "\r                                                  \rDownloading example.com/file.txt - 1.0 kB complete "

	// Capture the output of PrintProgress
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Save the original os.Stdout
	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()

	// Redirect os.Stdout to the pipe
	os.Stdout = w

	// Call the method to test
	counter.PrintProgress()

	// Close the writer and read the output
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Error(err)
	}

	// Check the captured output
	assert.Equal(t, expectedOutput, buf.String())
}

func TestDownloadFile_HTTPServer(t *testing.T) {

	logger := logging.NewTestLogger()

	// Spin up an in-memory HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "content")
	}))
	defer ts.Close()

	fs := afero.NewMemMapFs()
	err := DownloadFile(fs, context.Background(), ts.URL, "/file.dat", logger, true)
	require.NoError(t, err)

	data, _ := afero.ReadFile(fs, "/file.dat")
	assert.Equal(t, "content", string(data))
}

func TestDownloadFile_StatusError(t *testing.T) {

	logger := logging.NewTestLogger()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	fs := afero.NewMemMapFs()
	err := DownloadFile(fs, context.Background(), ts.URL, "/errfile", logger, true)
	assert.Error(t, err)
}

func TestDownloadFiles_SkipExisting(t *testing.T) {
	logger := logging.NewTestLogger()
	fs := afero.NewMemMapFs()
	dir := "/downloads"
	// Pre-create file with content
	_ = fs.MkdirAll(dir, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(dir, "f1"), []byte("old"), 0o644)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "new")
	}))
	defer ts.Close()

	items := []DownloadItem{{URL: ts.URL, LocalName: "f1"}}
	// useLatest=true forces overwrite of existing file
	_ = DownloadFiles(fs, context.Background(), dir, items, logger, true)
	exists, _ := afero.Exists(fs, filepath.Join(dir, "f1"))
	assert.True(t, exists)
}

func TestDownloadFile_FileCreationError(t *testing.T) {

	logger = logging.GetLogger()
	fs := afero.NewMemMapFs()

	// Invalid file path test case
	err := DownloadFile(fs, ctx, "http://localhost:8080", "", logger, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file path")
}

func TestDownloadFile_HTTPGetError(t *testing.T) {

	logger = logging.GetLogger()
	fs := afero.NewMemMapFs()

	// Trying to download a file from an invalid URL
	err := DownloadFile(fs, ctx, "http://invalid-url", "/testfile", logger, true)
	require.Error(t, err)
}

func newTestSetup() (afero.Fs, context.Context, *logging.Logger) {
	return afero.NewMemMapFs(), context.Background(), logging.NewTestLogger()
}

func TestDownloadFileSuccessAndSkip(t *testing.T) {
	fs, ctx, logger := newTestSetup()

	// Fake server serving content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	dest := "/tmp/file.txt"
	// Ensure directory exists
	_ = fs.MkdirAll(filepath.Dir(dest), 0o755)

	// 1) successful download
	if err := DownloadFile(fs, ctx, srv.URL, dest, logger, false); err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}

	// Verify file content
	data, _ := afero.ReadFile(fs, dest)
	if string(data) != "hello" {
		t.Errorf("unexpected file content: %s", string(data))
	}

	// 2) call again with useLatest=false  should skip because file exists and non-empty
	if err := DownloadFile(fs, ctx, srv.URL, dest, logger, false); err != nil {
		t.Fatalf("second DownloadFile error: %v", err)
	}

	// 3) call with useLatest=true  should overwrite (simulate by serving different content)
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new"))
	})
	if err := DownloadFile(fs, ctx, srv.URL, dest, logger, true); err != nil {
		t.Fatalf("DownloadFile with latest error: %v", err)
	}
	data, _ = afero.ReadFile(fs, dest)
	if string(data) != "new" {
		t.Errorf("file not overwritten with latest: %s", string(data))
	}
}

func TestDownloadFileHTTPErrorAndBadPath(t *testing.T) {
	fs, ctx, logger := newTestSetup()

	// Server returns 404
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	dest := "/tmp/err.txt"
	_ = fs.MkdirAll(filepath.Dir(dest), 0o755)

	if err := DownloadFile(fs, ctx, srv.URL, dest, logger, false); err == nil {
		t.Errorf("expected error on non-200 status, got nil")
	}

	// Empty path should error immediately
	if err := DownloadFile(fs, ctx, srv.URL, "", logger, false); err == nil {
		t.Errorf("expected error on empty destination path, got nil")
	}
}

func TestDownloadFilesWrapper(t *testing.T) {
	// Use the OS filesystem with a temp directory because DownloadFiles creates dirs via os.MkdirAll.
	dir := filepath.Join(t.TempDir(), "downloads")
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// server returns simple content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("x"))
	}))
	defer srv.Close()

	items := []DownloadItem{{URL: srv.URL, LocalName: "x.txt"}}

	if err := DownloadFiles(fs, ctx, dir, items, logger, false); err != nil {
		t.Fatalf("DownloadFiles error: %v", err)
	}

	// Ensure file exists
	content, err := afero.ReadFile(fs, filepath.Join(dir, "x.txt"))
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if string(content) != "x" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

// createTestServer returns a httptest.Server that serves the provided body with status 200.
func createTestServer(body string, status int) *httptest.Server {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	return httptest.NewServer(h)
}

func TestDownloadFile_SuccessUnit(t *testing.T) {
	srv := createTestServer("hello", http.StatusOK)
	defer srv.Close()

	mem := afero.NewMemMapFs()
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "file.txt")

	err := DownloadFile(mem, context.Background(), srv.URL, dst, logging.NewTestLogger(), false)
	assert.NoError(t, err)

	data, err := afero.ReadFile(mem, dst)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestDownloadFile_StatusErrorUnit(t *testing.T) {
	srv := createTestServer("bad", http.StatusInternalServerError)
	defer srv.Close()

	mem := afero.NewMemMapFs()
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "err.txt")

	err := DownloadFile(mem, context.Background(), srv.URL, dst, logging.NewTestLogger(), false)
	assert.Error(t, err)
}

func TestDownloadFile_ExistingSkipUnit(t *testing.T) {
	srv := createTestServer("new", http.StatusOK)
	defer srv.Close()

	mem := afero.NewMemMapFs()
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "skip.txt")

	// Pre-create file with content
	assert.NoError(t, afero.WriteFile(mem, dst, []byte("old"), 0o644))

	err := DownloadFile(mem, context.Background(), srv.URL, dst, logging.NewTestLogger(), false)
	assert.NoError(t, err)

	data, _ := afero.ReadFile(mem, dst)
	assert.Equal(t, "old", string(data)) // should not overwrite
}

func TestDownloadFile_OverwriteWithLatestUnit(t *testing.T) {
	srv := createTestServer("fresh", http.StatusOK)
	defer srv.Close()

	mem := afero.NewMemMapFs()
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "latest.txt")

	// Pre-create file with stale content
	assert.NoError(t, afero.WriteFile(mem, dst, []byte("stale"), 0o644))

	err := DownloadFile(mem, context.Background(), srv.URL, dst, logging.NewTestLogger(), true)
	assert.NoError(t, err)

	data, _ := afero.ReadFile(mem, dst)
	assert.Equal(t, "fresh", string(data)) // should overwrite
}

func TestDownloadFiles_MultipleUnit(t *testing.T) {
	srv1 := createTestServer("one", http.StatusOK)
	defer srv1.Close()
	srv2 := createTestServer("two", http.StatusOK)
	defer srv2.Close()

	tmpDir := t.TempDir()
	mem := afero.NewOsFs() // DownloadFiles uses os.MkdirAll; use real fs under tmpDir

	items := []DownloadItem{
		{URL: srv1.URL, LocalName: "a.txt"},
		{URL: srv2.URL, LocalName: "b.txt"},
	}

	err := DownloadFiles(mem, context.Background(), tmpDir, items, logging.NewTestLogger(), false)
	assert.NoError(t, err)

	for _, n := range []string{"a.txt", "b.txt"} {
		path := filepath.Join(tmpDir, n)
		info, err := mem.Stat(path)
		assert.NoError(t, err)
		assert.NotZero(t, info.Size())
	}

	// Cleanup tmpDir to avoid clutter; ignore errors.
	_ = os.RemoveAll(tmpDir)
}
