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
	"github.com/kdeps/kdeps/pkg/schema"
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
		ItemName:    "file.txt",
		IsCache:     false,
	}
	counter.Total = 1024

	expectedOutput := "\r                                                                                \r📥 Downloading file.txt - 1.0 kB"

	// Capture the output of PrintProgress
	r, w, err := os.Pipe()
	require.NoError(t, err)

	// Save the original os.Stdout
	originalStdout := os.Stdout
	defer func() { os.Stdout = originalStdout }()

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
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "content")
	}))
	defer ts.Close()

	fs := afero.NewMemMapFs()
	err := DownloadFile(context.Background(), fs, ts.URL, "/file.dat", logger, true)
	require.NoError(t, err)

	data, _ := afero.ReadFile(fs, "/file.dat")
	assert.Equal(t, "content", string(data))
}

func TestDownloadFile_StatusError(t *testing.T) {
	logger := logging.NewTestLogger()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	fs := afero.NewMemMapFs()
	err := DownloadFile(context.Background(), fs, ts.URL, "/errfile", logger, true)
	assert.Error(t, err)
}

func TestDownloadFiles_SkipExisting(t *testing.T) {
	logger := logging.NewTestLogger()
	fs := afero.NewMemMapFs()
	dir := "/downloads"
	// Pre-create file with content
	_ = fs.MkdirAll(dir, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(dir, "f1"), []byte("old"), 0o644)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	err := DownloadFile(ctx, fs, "http://localhost:8080", "", logger, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file path")
}

func TestDownloadFile_HTTPGetError(t *testing.T) {
	logger = logging.GetLogger()
	fs := afero.NewMemMapFs()

	// Trying to download a file from an invalid URL
	err := DownloadFile(ctx, fs, "http://invalid-url", "/testfile", logger, true)
	require.Error(t, err)
}

func newTestSetup() (afero.Fs, context.Context, *logging.Logger) {
	return afero.NewMemMapFs(), context.Background(), logging.NewTestLogger()
}

func TestDownloadFileSuccessAndSkip(t *testing.T) {
	fs, ctx, logger := newTestSetup()

	// Fake server serving content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()

	dest := "/tmp/file.txt"
	// Ensure directory exists
	_ = fs.MkdirAll(filepath.Dir(dest), 0o755)

	// 1) successful download
	if err := DownloadFile(ctx, fs, srv.URL, dest, logger, false); err != nil {
		t.Fatalf("DownloadFile returned error: %v", err)
	}

	// Verify file content
	data, _ := afero.ReadFile(fs, dest)
	if string(data) != "hello" {
		t.Errorf("unexpected file content: %s", string(data))
	}

	// 2) call again with useLatest=false  should skip because file exists and non-empty
	if err := DownloadFile(ctx, fs, srv.URL, dest, logger, false); err != nil {
		t.Fatalf("second DownloadFile error: %v", err)
	}

	// 3) call with useLatest=true  should overwrite (simulate by serving different content)
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("new"))
	})
	if err := DownloadFile(ctx, fs, srv.URL, dest, logger, true); err != nil {
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

	if err := DownloadFile(ctx, fs, srv.URL, dest, logger, false); err == nil {
		t.Errorf("expected error on non-200 status, got nil")
	}

	// Empty path should error immediately
	if err := DownloadFile(ctx, fs, srv.URL, "", logger, false); err == nil {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	err := DownloadFile(context.Background(), mem, srv.URL, dst, logging.NewTestLogger(), false)
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

	err := DownloadFile(context.Background(), mem, srv.URL, dst, logging.NewTestLogger(), false)
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

	err := DownloadFile(context.Background(), mem, srv.URL, dst, logging.NewTestLogger(), false)
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

	err := DownloadFile(context.Background(), mem, srv.URL, dst, logging.NewTestLogger(), true)
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

func TestWriteCounter(t *testing.T) {
	wc := &WriteCounter{DownloadURL: "example.com/file"}
	n, err := wc.Write([]byte("hello world"))
	require.NoError(t, err)
	require.Equal(t, 11, n)
	require.Equal(t, uint64(11), wc.Total)
}

func TestDownloadFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Successful download via httptest server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(w, bytes.NewBufferString("file-content"))
	}))
	defer srv.Close()

	dest := filepath.Join("/", "tmp", "file.txt")
	err := DownloadFile(ctx, fs, srv.URL, dest, logger, true /* useLatest */)
	require.NoError(t, err)

	// Verify file was written
	data, err := afero.ReadFile(fs, dest)
	require.NoError(t, err)
	require.Equal(t, "file-content", string(data))

	// Non-OK status code should error
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badSrv.Close()
	err = DownloadFile(ctx, fs, badSrv.URL, filepath.Join("/", "tmp", "bad.txt"), logger, true)
	require.Error(t, err)

	// Empty destination path should error immediately
	err = DownloadFile(ctx, fs, srv.URL, "", logger, true)
	require.Error(t, err)
}

func TestDownloadFilesSkipExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	downloadDir := "downloads"
	_ = fs.MkdirAll(downloadDir, 0o755)
	existingPath := filepath.Join(downloadDir, "existing.txt")
	_ = afero.WriteFile(fs, existingPath, []byte("content"), 0o644)

	items := []DownloadItem{{URL: "https://example.com/does-not-matter", LocalName: "existing.txt"}}

	// useLatest = false, so DownloadFile should skip re-download
	err := DownloadFiles(fs, ctx, downloadDir, items, logger, false)
	require.NoError(t, err)

	// Ensure file still contains original content (not overwritten)
	data, err := afero.ReadFile(fs, existingPath)
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
}

func TestDownloadFilesSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// httptest server to serve content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("abc"))
	}))
	defer srv.Close()

	downloadDir := "downloads"
	items := []DownloadItem{{URL: srv.URL, LocalName: "file.dat"}}

	// Create dir in memfs to avoid create error inside DownloadFile
	_ = fs.MkdirAll(downloadDir, 0o755)

	err := DownloadFiles(fs, ctx, downloadDir, items, logger, true) // useLatest so always download
	require.NoError(t, err)

	// verify file content exists and correct
	data, err := afero.ReadFile(fs, filepath.Join(downloadDir, "file.dat"))
	require.NoError(t, err)
	require.Equal(t, "abc", string(data))
}

// TestMakeGetRequestError verifies that MakeGetRequest returns an error for invalid URLs.
func TestMakeGetRequestError(t *testing.T) {
	ctx := context.Background()
	_, err := MakeGetRequest(ctx, "://invalid-url")
	require.Error(t, err)
}

// TestDownloadFileSkipExisting verifies DownloadFile skips downloading when file exists and non-empty
func TestDownloadFileSkipExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	// create existing file with content
	path := "existing.txt"
	require.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0o644))
	// DownloadFile should skip and leave content unchanged
	err := DownloadFile(ctx, fs, "http://unused", path, logger, false)
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, path)
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
}

// TestDownloadFileUseLatest ensures existing files are removed when useLatest is true
func TestDownloadFileUseLatest(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	// create existing file with content
	path := "file.dat"
	require.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0o644))
	// Setup test server for new content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("new"))
	}))
	defer srv.Close()
	// Use useLatest true to force re-download
	err := DownloadFile(ctx, fs, srv.URL, path, logger, true)
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, path)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
}

// Additional wrapper tests are present in other *_test.go files to avoid name collisions.

func TestDownloadFiles_HappyAndLatest(t *testing.T) {
	// touch schema version for rule compliance
	_ = schema.SchemaVersion(context.Background())

	fs := afero.NewOsFs()
	tempDir := t.TempDir()

	// Create httptest server that serves some content
	payload1 := []byte("v1-content")
	payload2 := []byte("v2-content")
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if call == 0 {
			w.Write(payload1)
		} else {
			w.Write(payload2)
		}
		call++
	}))
	defer srv.Close()

	items := []DownloadItem{{URL: srv.URL, LocalName: "file.bin"}}

	logger := logging.NewTestLogger()
	// First download (useLatest=false) should write payload1
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, false); err != nil {
		t.Fatalf("first DownloadFiles error: %v", err)
	}

	dest := filepath.Join(tempDir, "file.bin")
	data, _ := afero.ReadFile(fs, dest)
	if string(data) != string(payload1) {
		t.Fatalf("unexpected content after first download: %s", string(data))
	}

	// Second call with useLatest=false should skip (call counter unchanged)
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, false); err != nil {
		t.Fatalf("second DownloadFiles error: %v", err)
	}
	if call != 1 {
		t.Fatalf("expected server not called again, got call=%d", call)
	}

	// Third call with useLatest=true should re-download and overwrite with payload2
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, true); err != nil {
		t.Fatalf("third DownloadFiles error: %v", err)
	}
	data2, _ := afero.ReadFile(fs, dest)
	if string(data2) != string(payload2) {
		t.Fatalf("expected overwritten content, got %s", string(data2))
	}
}

// TestDownloadFile_SkipWhenExists verifies that DownloadFile returns nil and does not re-download
// when the target file already exists and useLatest is false.
func TestDownloadFile_SkipWhenExists(t *testing.T) {
	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "data.txt")

	// Pre-create non-empty file to trigger skip logic.
	if err := afero.WriteFile(fs, dest, []byte("cached"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// URL is irrelevant because we expect early return.
	err := DownloadFile(context.Background(), fs, "http://example.com/irrelevant", dest, logging.NewTestLogger(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDownloadFile_InvalidStatus exercises the non-200 status code branch.
func TestDownloadFile_InvalidStatus(t *testing.T) {
	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "out.txt")

	// Spin up a server that returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := DownloadFile(context.Background(), fs, srv.URL, dest, logging.NewTestLogger(), true)
	if err == nil {
		t.Fatalf("expected error on 500 status")
	}
}
