package memory

import (
	"net/url"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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
}

func TestInitializeDatabase(t *testing.T) {
	t.Run("SuccessfulInitialization", func(t *testing.T) {
		db, err := InitializeDatabase("file::memory:")
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='records'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "records", name)
	})

	t.Run("InvalidPath", func(t *testing.T) {
		db, err := InitializeDatabase("file::memory:?cache=invalid")
		if err != nil {
			if db != nil {
				defer db.Close()
				err = db.Ping()
				require.NoError(t, err, "Expected database to be usable even with invalid cache parameter")
			}
		}
	})
}

func TestInitializeMemory(t *testing.T) {
	reader, err := InitializeMemory("file::memory:")
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, reader.DB)
	require.Equal(t, "file::memory:", reader.DBPath)
	defer reader.DB.Close()
}
