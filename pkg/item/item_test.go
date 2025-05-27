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
	reader, err := InitializeItem(dbPath, nil)
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "item", reader.Scheme())
	})

	t.Run("Read_GetRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a record
		_, err = reader.DB.Exec("INSERT INTO items (id, value) VALUES (?, ?)", "20250101120000.000000", "value1")
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
	})

	t.Run("Read_UpdateCurrent", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("item:/_?op=updateCurrent&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		// Find the most recent ID
		var id string
		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		var value string
		err = reader.DB.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)

		// Test missing value parameter
		uri, _ = url.Parse("item:/_?op=updateCurrent")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")
	})

	t.Run("Read_Set", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a current item
		currentID := "20250101120000.000000"
		_, err = reader.DB.Exec("INSERT INTO items (id, value) VALUES (?, ?)", currentID, "item1")
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
		rows, err := reader.DB.Query("SELECT result_value FROM results WHERE item_id = ? ORDER BY created_at", currentID)
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			err := rows.Scan(&value)
			require.NoError(t, err)
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, []string{"result1", "result2"}, values)

		// Test missing value parameter
		uri, _ = url.Parse("item:/_?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")

		// Test with no current item
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		uri, _ = url.Parse("item:/_?op=set&value=result3")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no current record exists")
	})

	t.Run("Read_Results", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert items and results
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'item1'),
			('20250101120001.000000', 'item2')
		`)
		require.NoError(t, err)

		_, err = reader.DB.Exec(`
			INSERT INTO results (id, item_id, result_value, created_at) VALUES
			('result1', '20250101120000.000000', 'result1', '2025-01-01T12:00:00Z'),
			('result2', '20250101120000.000000', 'result2', '2025-01-01T12:00:01Z'),
			('result3', '20250101120001.000000', 'result3', '2025-01-01T12:00:02Z')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=results")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["result1","result2","result3"]`), data)

		// Test with no results
		_, err = reader.DB.Exec("DELETE FROM results")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), data)
	})

	t.Run("Read_LastResult", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert items and results
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'item1'),
			('20250101120001.000000', 'item2')
		`)
		require.NoError(t, err)

		_, err = reader.DB.Exec(`
			INSERT INTO results (id, item_id, result_value, created_at) VALUES
			('result1', '20250101120000.000000', 'result1', '2025-01-01T12:00:00Z'),
			('result2', '20250101120000.000000', 'result2', '2025-01-01T12:00:01Z'),
			('result3', '20250101120001.000000', 'result3', '2025-01-01T12:00:02Z')
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
	})

	t.Run("Read_PrevRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'value1'),
			('20250101120001.000000', 'value2'),
			('20250101120002.000000', 'value3')
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
	})

	t.Run("Read_NextRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'value1'),
			('20250101120001.000000', 'value2'),
			('20250101120002.000000', 'value3')
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=next")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data) // No next record after most recent

		// Test with only one record
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO items (id, value) VALUES (?, ?)", "20250101120000.000000", "value1")
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
	})

	t.Run("Read_ListRecords", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert items
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'item1'),
			('20250101120001.000000', 'item2')
		`)
		require.NoError(t, err)

		// Insert results
		_, err = reader.DB.Exec(`
			INSERT INTO results (id, item_id, result_value, created_at) VALUES
			('result1', '20250101120000.000000', 'result1', '2025-01-01T12:00:00Z'),
			('result2', '20250101120000.000000', 'result2', '2025-01-01T12:00:01Z'),
			('result3', '20250101120001.000000', 'result3', '2025-01-01T12:00:02Z')
		`)
		require.NoError(t, err)

		// Verify results
		uri, _ := url.Parse("item:/_?op=results")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["result1","result2","result3"]`), data)

		// Test with no results
		_, err = reader.DB.Exec("DELETE FROM results")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), data)
	})

	t.Run("InitializeWithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"item1", "item2", "item1"} // Includes duplicate
		reader, err := InitializeItem("file::memory:", items)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify items were inserted
		rows, err := reader.DB.Query("SELECT value FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			err := rows.Scan(&value)
			require.NoError(t, err)
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

		// Verify results table schema
		var sql string
		err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='results'").Scan(&sql)
		require.NoError(t, err)
		require.Contains(t, sql, "id TEXT PRIMARY KEY")
		require.Contains(t, sql, "item_id TEXT NOT NULL")
		require.Contains(t, sql, "result_value TEXT NOT NULL")
		require.Contains(t, sql, "created_at TIMESTAMP NOT NULL")
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
		rows, err := db.Query("SELECT value FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			err := rows.Scan(&value)
			require.NoError(t, err)
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
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
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
		reader, err := InitializeItem("file::memory:", items)
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
		defer reader.DB.Close()

		// Verify items were inserted
		rows, err := reader.DB.Query("SELECT value FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			err := rows.Scan(&value)
			require.NoError(t, err)
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
