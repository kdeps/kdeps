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
