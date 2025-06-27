package memory_test

import (
	"net/url"
	"os"
	"testing"

	. "github.com/kdeps/kdeps/pkg/memory"
	"github.com/stretchr/testify/require"
)

// TestInitializeMemory_Basic ensures InitializeMemory creates a valid
// PklResourceReader and opens a writable SQLite database.
func TestInitializeMemory_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := tmpDir + "/test.db"

	reader, err := InitializeMemory(tmpPath)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, reader.DB)

	// Ensure we can ping the database.
	require.NoError(t, reader.DB.Ping())

	// Cleanup
	reader.DB.Close()

	// The db file should exist on disk.
	if _, err := os.Stat(tmpPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}

// TestPklResourceReaderOperations exercises the Read method paths (set, get,
// clear) to bump coverage through conditional branches.
func TestPklResourceReaderOperations(t *testing.T) {
	tmpDir := t.TempDir()
	reader, err := InitializeMemory(tmpDir + "/db.sqlite")
	require.NoError(t, err)

	// Set value
	setURL := url.URL{Scheme: "memory", Path: "/foo", RawQuery: "op=set&value=bar"}
	_, err = reader.Read(setURL)
	require.NoError(t, err)

	// Get value
	getURL := url.URL{Scheme: "memory", Path: "/foo"}
	data, err := reader.Read(getURL)
	require.NoError(t, err)
	require.Equal(t, "bar", string(data))

	// Clear all
	clearURL := url.URL{Scheme: "memory", Path: "/_", RawQuery: "op=clear"}
	_, err = reader.Read(clearURL)
	require.NoError(t, err)

	// Ensure value cleared
	data, err = reader.Read(getURL)
	require.NoError(t, err)
	require.Empty(t, string(data))
}
