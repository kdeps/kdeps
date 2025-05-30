package item

import (
	"net/url"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	t.Parallel()

	// Use in-memory SQLite database for testing
	dbPath := "file::memory:"
	reader, err := InitializeItem(dbPath, nil, "test123")
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "item", reader.Scheme())
	})

	t.Run("Read_GetRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a record
		_, err = reader.DB.Exec("INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)", "20250101120000.000000", "value1", "test123")
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=current")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value1"), data)

		// Test with no records
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_UpdateCurrent", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("item:/_?op=updateCurrent&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		// Verify the inserted record
		var id, actionID, value string
		err = reader.DB.QueryRow("SELECT id, action_id, value FROM items ORDER BY id DESC LIMIT 1").Scan(&id, &actionID, &value)
		require.NoError(t, err)
		require.Equal(t, "test123", actionID)
		require.Equal(t, "newvalue", value)

		// Test missing value parameter
		uri, _ = url.Parse("item:/_?op=updateCurrent")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Verify a new record was inserted with empty value
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 2, count) // Two records: one for newvalue, one for empty value

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_Set", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a current item
		currentID := "20250101120000.000000"
		_, err = reader.DB.Exec("INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)", currentID, "item1", "test123")
		require.NoError(t, err)

		// Append first result value
		uri, _ := url.Parse("item:/_?op=set&value=result1")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("result1"), data)

		// Append second result value
		uri, _ = url.Parse("item:/_?op=set&value=result2")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("result2"), data)

		// Verify both results are stored
		rows, err := reader.DB.Query("SELECT result_value, action_id FROM results WHERE item_id = ? ORDER BY created_at", currentID)
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value, actionID string
			err := rows.Scan(&value, &actionID)
			require.NoError(t, err)
			require.Equal(t, "test123", actionID)
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"result1", "result2"}, values)

		// Test missing value parameter
		uri, _ = url.Parse("item:/_?op=set")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with no current item
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		uri, _ = url.Parse("item:/_?op=set&value=result3")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data) // Implementation fails due to foreign key constraint

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		_, err = readerEmpty.DB.Exec("INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)", currentID, "item1", "")
		require.NoError(t, err)
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("result3"), data) // Implementation inserts result regardless of actionID

		// Test actionID mismatch
		_, err = reader.DB.Exec("INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)", currentID, "item1", "wrongID")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("result3"), data) // Implementation ignores actionID mismatch
	})

	t.Run("Read_Values", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert items and results
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, action_id) VALUES
			('20250101120000.000000', 'item1', 'test123'),
			('20250101120001.000000', 'item2', 'test123')
		`)
		require.NoError(t, err)

		_, err = reader.DB.Exec(`
			INSERT INTO results (id, item_id, result_value, created_at, action_id) VALUES
			('result1', '20250101120000.000000', 'result1', '2025-01-01T12:00:00Z', 'test123'),
			('result2', '20250101120000.000000', 'result2', '2025-01-01T12:00:01Z', 'test123'),
			('result3', '20250101120001.000000', 'result3', '2025-01-01T12:00:02Z', 'test123')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/test123?op=values")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["result1","result2","result3"]`), data)

		// Test with no results
		_, err = reader.DB.Exec("DELETE FROM results")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), data)

		// Test with missing actionID in URI
		uri, _ = url.Parse("item:/?op=values")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_Values_DatabaseError", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Drop results table to simulate database error
		_, err = reader.DB.Exec("DROP TABLE results")
		require.NoError(t, err)

		uri, _ := url.Parse("item:/test123?op=values")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_LastResult", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert items and results
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, action_id) VALUES
			('20250101120000.000000', 'item1', 'test123'),
			('20250101120001.000000', 'item2', 'test123')
		`)
		require.NoError(t, err)

		_, err = reader.DB.Exec(`
			INSERT INTO results (id, item_id, result_value, created_at, action_id) VALUES
			('result1', '20250101120000.000000', 'result1', '2025-01-01T12:00:00Z', 'test123'),
			('result2', '20250101120000.000000', 'result2', '2025-01-01T12:00:01Z', 'test123'),
			('result3', '20250101120001.000000', 'result3', '2025-01-01T12:00:02Z', 'test123')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=lastResult")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("result3"), data)

		// Test with no results
		_, err = reader.DB.Exec("DELETE FROM results")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_PrevRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, action_id) VALUES
			('20250101120000.000000', 'value1', 'test123'),
			('20250101120001.000000', 'value2', 'test123'),
			('20250101120002.000000', 'value3', 'test123')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=prev")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value2"), data) // Previous to most recent

		// Test with no records
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_NextRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, action_id) VALUES
			('20250101120000.000000', 'value1', 'test123'),
			('20250101120001.000000', 'value2', 'test123'),
			('20250101120002.000000', 'value3', 'test123')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=next")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data) // No next record after most recent

		// Test with only one record
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)", "20250101120000.000000", "value1", "test123")
		require.NoError(t, err)

		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with no records
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)

		// Test with empty actionID
		readerEmpty, err := InitializeItem("file::memory:", nil, "")
		require.NoError(t, err)
		defer readerEmpty.DB.Close()
		data, err = readerEmpty.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("Read_InvalidOperation", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("item:/_?op=invalid")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("InitializeWithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"item1", "item2", "item1"} // Includes duplicate
		reader, err := InitializeItem("file::memory:", items, "test123")
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify items were inserted
		rows, err := reader.DB.Query("SELECT value, action_id FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value, actionID string
			err := rows.Scan(&value, &actionID)
			require.NoError(t, err)
			require.Equal(t, "", actionID) // Initial items have empty action_id
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"item1", "item2", "item1"}, values)

		// Verify record count
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
	})
}

func TestInitializeDatabase(t *testing.T) {
	t.Parallel()

	t.Run("SuccessfulInitialization", func(t *testing.T) {
		t.Parallel()
		db, err := InitializeDatabase("file::memory:", []string{})
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify items table
		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "items", name)

		// Verify results table
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='results'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "results", name)

		// Verify items table schema
		var sql string
		err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='items'").Scan(&sql)
		require.NoError(t, err)
		require.Contains(t, sql, "id TEXT PRIMARY KEY")
		require.Contains(t, sql, "value TEXT NOT NULL")
		require.Contains(t, sql, "action_id TEXT")

		// Verify results table schema
		err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='results'").Scan(&sql)
		require.NoError(t, err)
		require.Contains(t, sql, "id TEXT PRIMARY KEY")
		require.Contains(t, sql, "item_id TEXT NOT NULL")
		require.Contains(t, sql, "result_value TEXT NOT NULL")
		require.Contains(t, sql, "created_at TIMESTAMP NOT NULL")
		require.Contains(t, sql, "action_id TEXT")
		require.Contains(t, sql, "FOREIGN KEY (item_id) REFERENCES items(id)")
	})

	t.Run("InitializationWithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"test1", "test2", "test1"}
		db, err := InitializeDatabase("file::memory:", items)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify items were inserted
		rows, err := db.Query("SELECT value, action_id FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value, actionID string
			err := rows.Scan(&value, &actionID)
			require.NoError(t, err)
			require.Equal(t, "", actionID) // Initial items have empty action_id
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"test1", "test2", "test1"}, values) // Includes duplicates in DB

		// Verify record count
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
	})
}

func TestInitializeItem(t *testing.T) {
	t.Parallel()

	t.Run("WithoutItems", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil, "test123")
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
		require.Equal(t, "test123", reader.ActionID)
		defer reader.DB.Close()

		// Verify empty database
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("WithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"item1", "item2", "item1"}
		reader, err := InitializeItem("file::memory:", items, "test123")
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
		require.Equal(t, "test123", reader.ActionID)
		defer reader.DB.Close()

		// Verify items were inserted
		rows, err := reader.DB.Query("SELECT value, action_id FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value, actionID string
			err := rows.Scan(&value, &actionID)
			require.NoError(t, err)
			require.Equal(t, "", actionID) // Initial items have empty action_id
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"item1", "item2", "item1"}, values)

		// Verify record count
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
	})
}
