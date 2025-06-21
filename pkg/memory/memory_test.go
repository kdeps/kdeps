package memory_test

import (
	"net/url"
	"testing"

	"path/filepath"

	. "github.com/kdeps/kdeps/pkg/memory"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	// Use in-memory SQLite database for testing
	dbPath := "file::memory:"
	reader, err := InitializeMemory(dbPath)
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		require.Equal(t, "memory", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("memory:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("Read_GetRecord", func(t *testing.T) {
		reader, err := InitializeMemory("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec("INSERT INTO records (id, value) VALUES (?, ?)", "test1", "value1")
		require.NoError(t, err)

		uri, _ := url.Parse("memory:///test1")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value1"), data)

		uri, _ = url.Parse("memory:///nonexistent")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		uri, _ = url.Parse("memory:///")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no record ID provided")
	})

	t.Run("Read_SetRecord", func(t *testing.T) {
		reader, err := InitializeMemory("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("memory:///test2?op=set&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ?", "test2").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)

		uri, _ = url.Parse("memory:///test3?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")

		uri, _ = url.Parse("memory:///?op=set&value=value")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no record ID provided for set operation")
	})

	t.Run("Read_DeleteRecord", func(t *testing.T) {
		reader, err := InitializeMemory("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec("INSERT INTO records (id, value) VALUES (?, ?)", "test4", "value4")
		require.NoError(t, err)

		uri, _ := url.Parse("memory:///test4?op=delete")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("Deleted 1 record(s)"), data)

		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE id = ?", "test4").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("Deleted 0 record(s)"), data)

		uri, _ = url.Parse("memory:///?op=delete")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no record ID provided for delete operation")
	})

	t.Run("Read_Clear", func(t *testing.T) {
		reader, err := InitializeMemory("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Clear any existing data to ensure a clean state
		_, err = reader.DB.Exec("DELETE FROM records")
		require.NoError(t, err, "Failed to clear table before test")

		// Set up test data
		result, err := reader.DB.Exec("INSERT INTO records (id, value) VALUES (?, ?), (?, ?)",
			"test5", "value5", "test6", "value6")
		require.NoError(t, err, "Failed to insert test data")
		rowsAffected, err := result.RowsAffected()
		require.NoError(t, err, "Failed to check rows affected")
		require.Equal(t, int64(2), rowsAffected, "Expected 2 rows to be inserted")

		// Verify data is present
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records").Scan(&count)
		require.NoError(t, err, "Failed to count records")
		require.Equal(t, 2, count, "Expected 2 records in table before clear")

		// Perform clear operation
		uri, _ := url.Parse("memory:///_?op=clear")
		data, err := reader.Read(*uri)
		require.NoError(t, err, "Clear operation failed")
		require.Equal(t, []byte("Cleared 2 records"), data, "Unexpected response from clear")

		// Verify all records were deleted
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records").Scan(&count)
		require.NoError(t, err, "Failed to count records after clear")
		require.Equal(t, 0, count, "Expected 0 records in table after clear")

		// Test invalid path
		uri, _ = url.Parse("memory:///invalid?op=clear")
		_, err = reader.Read(*uri)
		require.Error(t, err, "Expected error for invalid clear path")
		require.Contains(t, err.Error(), "clear operation requires path '/_'", "Unexpected error message")
	})

	t.Run("Read_NilReceiver", func(t *testing.T) {
		nilReader := &PklResourceReader{DBPath: dbPath}
		uri, _ := url.Parse("memory:///test7?op=set&value=value7")
		data, err := nilReader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value7"), data)

		var value string
		err = nilReader.DB.QueryRow("SELECT value FROM records WHERE id = ?", "test7").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "value7", value)
	})

	t.Run("Read_NilDB", func(t *testing.T) {
		reader := &PklResourceReader{DBPath: dbPath, DB: nil}
		uri, _ := url.Parse("memory:///test8?op=set&value=value8")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value8"), data)

		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ?", "test8").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "value8", value)
	})

	t.Run("Read_SetRecord_DatabaseClosed", func(t *testing.T) {
		reader, err := InitializeMemory("file::memory:")
		require.NoError(t, err)
		reader.DB.Close() // Close database to simulate failure

		uri, _ := url.Parse("memory:///failset?op=set&value=fail")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to set record")
	})
}

func TestInitializeDatabase(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "memory-db")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		// Test successful database initialization
		dbPath := filepath.Join(tmpDir, "success.db")
		db, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Verify the database was created and is accessible
		err = db.Ping()
		require.NoError(t, err)

		// Verify the records table exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM records").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("DatabaseAlreadyExists", func(t *testing.T) {
		// Test that the function works when the database already exists
		dbPath := filepath.Join(tmpDir, "existing.db")

		// Create the database first
		db1, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		db1.Close()

		// Initialize again
		db2, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db2.Close()

		// Verify the database is accessible
		err = db2.Ping()
		require.NoError(t, err)

		// Verify the records table exists
		var count int
		err = db2.QueryRow("SELECT COUNT(*) FROM records").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("InvalidDatabasePath", func(t *testing.T) {
		// Test with an invalid database path (directory doesn't exist)
		invalidPath := filepath.Join(tmpDir, "nonexistent", "db.sqlite")
		_, err := InitializeDatabase(invalidPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping database after 5 attempts")
	})

	t.Run("RetryLogic", func(t *testing.T) {
		// This test verifies that the retry logic works
		// We can't easily simulate database failures, but we can verify the function
		// handles normal cases correctly
		dbPath := filepath.Join(tmpDir, "retry.db")
		db, err := InitializeDatabase(dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Verify the database works
		err = db.Ping()
		require.NoError(t, err)
	})
}

func TestInitializeMemory(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "memory-init")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		// Test successful initialization
		dbPath := filepath.Join(tmpDir, "memory.db")

		reader, err := InitializeMemory(dbPath)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify the reader was created correctly
		require.NotNil(t, reader)
		require.Equal(t, dbPath, reader.DBPath)
		require.NotNil(t, reader.DB)

		// Verify the database is accessible
		err = reader.DB.Ping()
		require.NoError(t, err)

		// Verify the records table exists
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("DatabaseInitializationError", func(t *testing.T) {
		// Test when database initialization fails
		invalidPath := filepath.Join(tmpDir, "nonexistent", "db.sqlite")
		_, err := InitializeMemory(invalidPath)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error initializing database")
	})
}

func TestInitializeDatabase_InvalidPath(t *testing.T) {
	// Try to open a database at an invalid path
	db, err := InitializeDatabase("/invalid/path/to/db.sqlite")
	require.Error(t, err)
	require.Nil(t, db)
}

func TestInitializeDatabase_PermissionDenied(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "memory-db-perm")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	// Create a read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err = fs.MkdirAll(readOnlyDir, 0o400)
	require.NoError(t, err)

	// Try to create a database file in the read-only directory
	dbPath := filepath.Join(readOnlyDir, "test.db")
	db, err := InitializeDatabase(dbPath)
	require.Error(t, err)
	require.Nil(t, db)
}

func TestInitializeMemory_InvalidPath(t *testing.T) {
	reader, err := InitializeMemory("/invalid/path/to/db.sqlite")
	require.Error(t, err)
	require.Nil(t, reader)
}

func TestInitializeDatabase_TransactionFailure(t *testing.T) {
	// Use a closed DB to simulate transaction failure
	db, err := InitializeDatabase("file::memory:")
	require.NoError(t, err)
	db.Close()
	// Try to use the closed DB
	_, err = db.Exec("INSERT INTO records (id, value) VALUES (?, ?)", "fail", "fail")
	require.Error(t, err)
}
