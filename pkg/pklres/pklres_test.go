package pklres_test

import (
	"net/url"
	"testing"

	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	// Use in-memory SQLite database for testing
	dbPath := "file::memory:"
	reader, err := pklres.InitializePklResource(dbPath)
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		require.Equal(t, "pklres", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("pklres:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("Read_GetRecord_Simple", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a simple record (no key)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "", "value1")
		require.NoError(t, err)

		// Get the record
		uri, _ := url.Parse("pklres:///test1?type=config")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value1"), data)

		// Test with non-existent record
		uri, _ = url.Parse("pklres:///nonexistent?type=config")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_GetRecord_WithKey", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a structured record with key
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "database", "postgresql://localhost")
		require.NoError(t, err)

		// Get the record by key
		uri, _ := url.Parse("pklres:///test1?type=config&key=database")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("postgresql://localhost"), data)

		// Test with non-existent key
		uri, _ = url.Parse("pklres:///test1?type=config&key=nonexistent")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_SetRecord_Simple", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Set a simple record
		uri, _ := url.Parse("pklres:///test1?type=config&op=set&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		// Verify it was stored
		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ? AND type = ? AND key = ?", "test1", "config", "").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)

		// Test missing parameters
		uri, _ = url.Parse("pklres:///test1?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires id and type parameters")

		uri, _ = url.Parse("pklres:///test1?type=config&op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")
	})

	t.Run("Read_SetRecord_WithKey", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Set a structured record with key
		uri, _ := url.Parse("pklres:///test1?type=config&key=database&op=set&value=postgresql://localhost")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("postgresql://localhost"), data)

		// Verify it was stored
		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ? AND type = ? AND key = ?", "test1", "config", "database").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "postgresql://localhost", value)

		// Set another key for the same record
		uri, _ = url.Parse("pklres:///test1?type=config&key=redis&op=set&value=redis://localhost:6379")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("redis://localhost:6379"), data)

		// Verify both keys exist
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE id = ? AND type = ?", "test1", "config").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 2, count)
	})

	t.Run("Read_DeleteRecord_Simple", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a simple record
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "", "value1")
		require.NoError(t, err)

		// Delete the record
		uri, _ := url.Parse("pklres:///test1?type=config&op=delete")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Deleted 1 record(s)")

		// Verify it was deleted
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE id = ? AND type = ?", "test1", "config").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		// Test missing parameters
		uri, _ = url.Parse("pklres:///test1?op=delete")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "delete operation requires id and type parameters")
	})

	t.Run("Read_DeleteRecord_WithKey", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert multiple keys for the same record
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "database", "postgresql://localhost")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "redis", "redis://localhost:6379")
		require.NoError(t, err)

		// Delete specific key
		uri, _ := url.Parse("pklres:///test1?type=config&key=database&op=delete")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Deleted 1 record(s)")

		// Verify only one key was deleted
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE id = ? AND type = ?", "test1", "config").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// Verify the right key was deleted
		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ? AND type = ? AND key = ?", "test1", "config", "redis").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "redis://localhost:6379", value)
	})

	t.Run("Read_ClearRecords_ByType", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert records of different types
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "", "value1")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test2", "config", "", "value2")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test3", "cache", "", "value3")
		require.NoError(t, err)

		// Clear only config records
		uri, _ := url.Parse("pklres:///_?type=config&op=clear")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Cleared 2 records")

		// Verify config records were cleared but cache records remain
		var configCount, cacheCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE type = ?", "config").Scan(&configCount)
		require.NoError(t, err)
		require.Equal(t, 0, configCount)

		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records WHERE type = ?", "cache").Scan(&cacheCount)
		require.NoError(t, err)
		require.Equal(t, 1, cacheCount)

		// Test missing type parameter
		uri, _ = url.Parse("pklres:///_?op=clear")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "clear operation requires type parameter")
	})

	t.Run("Read_ClearRecords_All", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert records of different types
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "", "value1")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test2", "cache", "", "value2")
		require.NoError(t, err)

		// Clear all records
		uri, _ := url.Parse("pklres:///_?type=_&op=clear")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Cleared 2 records")

		// Verify all records were cleared
		var totalCount int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM records").Scan(&totalCount)
		require.NoError(t, err)
		require.Equal(t, 0, totalCount)
	})

	t.Run("Read_ListRecords", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert records of different types
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "", "value1")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test1", "config", "database", "postgresql://localhost")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test2", "config", "", "value2")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO records (id, type, key, value) VALUES (?, ?, ?, ?)", "test3", "cache", "", "value3")
		require.NoError(t, err)

		// List config records
		uri, _ := url.Parse("pklres:///_?type=config&op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "test1")
		require.Contains(t, string(data), "test2")
		require.NotContains(t, string(data), "test3")

		// List cache records
		uri, _ = url.Parse("pklres:///_?type=cache&op=list")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "test3")
		require.NotContains(t, string(data), "test1")

		// List non-existent type
		uri, _ = url.Parse("pklres:///_?type=nonexistent&op=list")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, "[]", string(data))

		// Test missing type parameter
		uri, _ = url.Parse("pklres:///_?op=list")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "list operation requires type parameter")
	})

	t.Run("Read_InvalidParameters", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test get without id
		uri, _ := url.Parse("pklres:///?type=config")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get operation requires id and type parameters")

		// Test get without type
		uri, _ = url.Parse("pklres:///test1")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "get operation requires id and type parameters")
	})

	t.Run("Database_Initialization", func(t *testing.T) {
		// Test database initialization
		db, err := pklres.InitializeDatabase(":memory:")
		require.NoError(t, err)
		defer db.Close()

		// Verify table was created
		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='records'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "records", name)

		// Verify indexes were created
		var indexCount int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name='records'").Scan(&indexCount)
		require.NoError(t, err)
		require.GreaterOrEqual(t, indexCount, 2) // Should have at least 2 indexes
	})

	t.Run("InitializePklResource", func(t *testing.T) {
		reader, err := pklres.InitializePklResource(":memory:")
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, ":memory:", reader.DBPath)
		defer reader.DB.Close()
	})

	t.Run("Read_EdgeCases", func(t *testing.T) {
		reader, err := pklres.InitializePklResource("file::memory:")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test update existing record
		uri, _ := url.Parse("pklres:///test1?type=config&op=set&value=value1")
		_, err = reader.Read(*uri)
		require.NoError(t, err)

		// Update with new value
		uri, _ = url.Parse("pklres:///test1?type=config&op=set&value=value2")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value2"), data)

		// Verify it was updated
		var value string
		err = reader.DB.QueryRow("SELECT value FROM records WHERE id = ? AND type = ? AND key = ?", "test1", "config", "").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "value2", value)

		// Test delete non-existent record
		uri, _ = url.Parse("pklres:///nonexistent?type=config&op=delete")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Deleted 0 record(s)")
	})
}
