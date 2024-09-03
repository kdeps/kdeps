package download

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	counter := &WriteCounter{}
	counter.Total = 1024

	expectedOutput := "\r                                                  \rDownloading... 1.0 kB complete"

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
	io.Copy(&buf, r)

	// Check the captured output
	assert.Equal(t, expectedOutput, buf.String())
}

func TestDownloadFile(t *testing.T) {
	// Mock a simple HTTP server to simulate file download
	server := http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Test file content"))
		}),
	}
	go server.ListenAndServe()
	defer server.Close()

	// Use afero in-memory filesystem
	fs := afero.NewMemMapFs()

	// Run the file download
	err := DownloadFile(fs, "http://localhost:8080", "/testfile")
	require.NoError(t, err)

	// Verify the downloaded content
	content, err := afero.ReadFile(fs, "/testfile")
	require.NoError(t, err)
	assert.Equal(t, "Test file content", string(content))
}

func TestDownloadFile_FileCreationError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Trying to download a file with an invalid filepath
	err := DownloadFile(fs, "http://localhost:8080", "")
	assert.Error(t, err)
}

func TestDownloadFile_HttpGetError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Trying to download a file from an invalid URL
	err := DownloadFile(fs, "http://invalid-url", "/testfile")
	assert.Error(t, err)
}
