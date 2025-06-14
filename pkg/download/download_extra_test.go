package download

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(w, bytes.NewBufferString("file-content"))
	}))
	defer srv.Close()

	dest := filepath.Join("/", "tmp", "file.txt")
	err := DownloadFile(fs, ctx, srv.URL, dest, logger, true /* useLatest */)
	require.NoError(t, err)

	// Verify file was written
	data, err := afero.ReadFile(fs, dest)
	require.NoError(t, err)
	require.Equal(t, "file-content", string(data))

	// Non-OK status code should error
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badSrv.Close()
	err = DownloadFile(fs, ctx, badSrv.URL, filepath.Join("/", "tmp", "bad.txt"), logger, true)
	require.Error(t, err)

	// Empty destination path should error immediately
	err = DownloadFile(fs, ctx, srv.URL, "", logger, true)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	require.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))
	// DownloadFile should skip and leave content unchanged
	err := DownloadFile(fs, ctx, "http://unused", path, logger, false)
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
	require.NoError(t, afero.WriteFile(fs, path, []byte("old"), 0644))
	// Setup test server for new content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("new"))
	}))
	defer srv.Close()
	// Use useLatest true to force re-download
	err := DownloadFile(fs, ctx, srv.URL, path, logger, true)
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, path)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
}

// Additional wrapper tests are present in other *_test.go files to avoid name collisions.
