package download

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	logger *logging.Logger
	ctx    context.Context
)

func TestWriteCounter_Write(t *testing.T) {
	t.Parallel()
	counter := &WriteCounter{}
	data := []byte("Hello, World!")
	n, err := counter.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, uint64(len(data)), counter.Total)
}

func TestWriteCounter_PrintProgress(t *testing.T) {
	t.Parallel()
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

func TestDownloadFile(t *testing.T) {
	t.Parallel()
	logger = logging.GetLogger()
	// Mock a simple HTTP server to simulate file download
	server := http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("Test file content")); err != nil {
				t.Error(err)
			}
		}),
	}
	go server.ListenAndServe()
	defer server.Close()

	// Use afero in-memory filesystem
	fs := afero.NewMemMapFs()

	// Run the file download
	err := DownloadFile(fs, ctx, "http://localhost:8080", "/testfile", logger)
	require.NoError(t, err)

	// Verify the downloaded content
	content, err := afero.ReadFile(fs, "/testfile")
	require.NoError(t, err)
	assert.Equal(t, "Test file content", string(content))
}

func TestDownloadFile_FileCreationError(t *testing.T) {
	t.Parallel()
	logger = logging.GetLogger()
	fs := afero.NewMemMapFs()

	// Invalid file path test case
	err := DownloadFile(fs, ctx, "http://localhost:8080", "", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file path")
}

func TestDownloadFile_HttpGetError(t *testing.T) {
	t.Parallel()
	logger = logging.GetLogger()
	fs := afero.NewMemMapFs()

	// Trying to download a file from an invalid URL
	err := DownloadFile(fs, ctx, "http://invalid-url", "/testfile", logger)
	assert.Error(t, err)
}
