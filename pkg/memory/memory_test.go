package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/memory"
)

func TestInitializeMemory(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test successful initialization
	reader, err := memory.InitializeMemory(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, reader)
	assert.NotNil(t, reader.DB)

	// Verify the database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Test initialization with existing database
	reader2, err := memory.InitializeMemory(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, reader2)
	assert.NotNil(t, reader2.DB)

	// Clean up
	reader.DB.Close()
	reader2.DB.Close()
}

func TestInitializeMemoryWithInvalidPath(t *testing.T) {
	// Test with an invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/path/test.db"

	reader, err := memory.InitializeMemory(invalidPath)
	assert.Error(t, err)
	assert.Nil(t, reader)
}

func TestInitializeMemoryWithReadOnlyDirectory(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Make the directory read-only
	err := os.Chmod(tempDir, 0o444)
	require.NoError(t, err)

	// Try to create database in read-only directory
	dbPath := filepath.Join(tempDir, "test.db")
	reader, err := memory.InitializeMemory(dbPath)
	assert.Error(t, err)
	assert.Nil(t, reader)

	// Restore permissions for cleanup
	os.Chmod(tempDir, 0o755)
}

func TestInitializeMemoryWithExistingInvalidDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Create an invalid database file (not a SQLite database)
	err := os.WriteFile(dbPath, []byte("not a database"), 0o644)
	require.NoError(t, err)

	// Try to initialize with invalid database
	reader, err := memory.InitializeMemory(dbPath)
	assert.Error(t, err)
	assert.Nil(t, reader)
}

func TestPklResourceReaderMethods(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Initialize memory
	reader, err := memory.InitializeMemory(dbPath)
	require.NoError(t, err)
	defer reader.DB.Close()

	// Test Close method
	err = reader.DB.Close()
	assert.NoError(t, err)

	// Test that DB is nil after closing
	assert.Nil(t, reader.DB)
}

func TestInitializeDatabase(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test successful database initialization
	db, err := memory.InitializeDatabase(dbPath)
	require.NoError(t, err)
	assert.NotNil(t, db)

	// Verify the database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)

	// Test that we can execute a simple query
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	assert.NoError(t, err)

	// Clean up
	db.Close()
}

func TestInitializeDatabaseWithInvalidPath(t *testing.T) {
	// Test with an invalid path
	invalidPath := "/nonexistent/path/test.db"

	db, err := memory.InitializeDatabase(invalidPath)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestInitializeMemoryConcurrentAccess(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Test concurrent access to the same database
	done := make(chan bool, 2)

	go func() {
		reader, err := memory.InitializeMemory(dbPath)
		require.NoError(t, err)
		defer reader.DB.Close()
		done <- true
	}()

	go func() {
		reader, err := memory.InitializeMemory(dbPath)
		require.NoError(t, err)
		defer reader.DB.Close()
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done
}
