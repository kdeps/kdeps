package item

import (
	"database/sql"
	"fmt"
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

	t.Run("IsGlobbable", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		t.Parallel()
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		t.Parallel()
		uri, _ := url.Parse("item:/_")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("Read_GetRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert a record
		_, err = reader.DB.Exec("INSERT INTO items (id, value, isLoop) VALUES (?, ?, ?)", "20250101120000.000000", "value1", 0)
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

	t.Run("Read_SetRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test without refs (basic set operation)
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'initial1', 0),
			('20250101120001.000000', 'initial2', 0),
			('20250101120002.000000', 'initial3', 0)
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=set&value=newvalue")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		// Find the most recent ID
		var id string
		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		var value string
		var isLoop int
		err = reader.DB.QueryRow("SELECT value, isLoop FROM items WHERE id = ?", id).Scan(&value, &isLoop)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)
		require.Equal(t, 0, isLoop) // No refs, so isLoop is false

		// Test with refs parameter (using the generated ID)
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'initial1', 0),
			('20250101120001.000000', 'initial2', 0),
			('20250101120002.000000', 'initial3', 0)
		`)
		require.NoError(t, err)

		// First set to get a generated ID
		uri, _ = url.Parse("item:/_?op=set&value=intermediate")
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("intermediate"), data)

		// Get the generated ID
		var generatedID string
		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&generatedID)
		require.NoError(t, err)

		// Set again with refs including the generated ID
		refs := fmt.Sprintf(`["20250101120000.000000","%s","20250101120002.000000"]`, generatedID)
		uri, _ = url.Parse(fmt.Sprintf("item:/_?op=set&value=newvalue&refs=%s", url.QueryEscape(refs)))
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		// Find the most recent ID
		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		err = reader.DB.QueryRow("SELECT value, isLoop FROM items WHERE id = ?", id).Scan(&value, &isLoop)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)
		require.Equal(t, 0, isLoop) // isLoop is false because refCurrentID does not match the new generated ID

		err = reader.DB.QueryRow("SELECT value FROM items WHERE id = ?", "20250101120000.000000").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "initial1", value) // No update occurs due to empty oldValue

		err = reader.DB.QueryRow("SELECT value FROM items WHERE id = ?", "20250101120002.000000").Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "initial3", value) // No update occurs due to empty oldValue

		// Test with refs but no prev/next
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO items (id, value, isLoop) VALUES (?, ?, ?)", "20250101120001.000000", "initial2", 0)
		require.NoError(t, err)

		uri, _ = url.Parse(`item:/_?op=set&value=newvalue&refs=["","20250101120001.000000",""]`)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("newvalue"), data)

		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		err = reader.DB.QueryRow("SELECT value, isLoop FROM items WHERE id = ?", id).Scan(&value, &isLoop)
		require.NoError(t, err)
		require.Equal(t, "newvalue", value)
		require.Equal(t, 0, isLoop) // isLoop is false because refCurrentID does not match the generated ID

		// Test invalid refs (wrong length)
		uri, _ = url.Parse(`item:/_?op=set&value=newvalue&refs=["20250101120000.000000","20250101120001.000000"]`)
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "refs must contain exactly 3 elements")

		// Test invalid refs (bad JSON)
		uri, _ = url.Parse(`item:/_?op=set&value=newvalue&refs=invalid`)
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse refs parameter")

		// Test missing value parameter
		uri, _ = url.Parse("item:/_?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")
	})

	t.Run("Read_PrevRecord", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'value1', 0),
			('20250101120001.000000', 'value2', 0),
			('20250101120002.000000', 'value3', 0)
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=prev")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value2"), data) // Previous to most recent (20250101120002.000000)

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
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'value1', 0),
			('20250101120001.000000', 'value2', 0),
			('20250101120002.000000', 'value3', 0)
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=next")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data) // No next record after most recent (20250101120002.000000)

		// Test with only one record
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		_, err = reader.DB.Exec("INSERT INTO items (id, value, isLoop) VALUES (?, ?, ?)", "20250101120000.000000", "value1", 0)
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

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'value1', 0),
			('20250101120001.000000', 'value2', 0)
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["value1","value2"]`), data)

		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), data)
	})

	t.Run("Read_Values", func(t *testing.T) {
		t.Parallel()
		reader, err := InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test empty database
		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)

		// Debug: Query all records
		rows, err := reader.DB.Query("SELECT id, value FROM items")
		require.NoError(t, err)
		defer rows.Close()
		var records []string
		for rows.Next() {
			var id, value string
			if err := rows.Scan(&id, &value); err != nil {
				t.Fatalf("Failed to scan record: %v", err)
			}
			records = append(records, fmt.Sprintf("id=%s, value=%s", id, value))
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("Failed to iterate records: %v", err)
		}
		if len(records) > 0 {
			t.Logf("Unexpected records in database: %v", records)
		}

		// Check record count
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "Expected empty database")

		uri, _ := url.Parse("item:/_?op=values")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`[]`), data)

		// Test with records
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value, isLoop) VALUES
			('20250101120000.000000', 'value1', 0),
			('20250101120001.000000', 'value2', 0)
		`)
		require.NoError(t, err)

		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["value1","value2"]`), data)
	})

	t.Run("Read_NilReceiver", func(t *testing.T) {
		t.Parallel()
		nilReader := &PklResourceReader{DBPath: dbPath}
		uri, _ := url.Parse("item:/_?op=set&value=value7")
		data, err := nilReader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value7"), data)

		// Find the most recent ID
		var id string
		err = nilReader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		var value string
		err = nilReader.DB.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "value7", value)
	})

	t.Run("Read_NilDB", func(t *testing.T) {
		t.Parallel()
		reader := &PklResourceReader{DBPath: dbPath, DB: nil}
		uri, _ := url.Parse("item:/_?op=set&value=value8")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("value8"), data)

		// Find the most recent ID
		var id string
		err = reader.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
		require.NoError(t, err)

		var value string
		err = reader.DB.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&value)
		require.NoError(t, err)
		require.Equal(t, "value8", value)
	})

	t.Run("InitializeWithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"item1", "item2", "item3"}
		reader, err := InitializeItem("file::memory:", items)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify items were inserted
		uri, _ := url.Parse("item:/_?op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["item1","item2","item3"]`), data)

		// Verify record count
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)

		// Verify isLoop is 0 for all items
		rows, err := reader.DB.Query("SELECT value, isLoop FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			var isLoop int
			err := rows.Scan(&value, &isLoop)
			require.NoError(t, err)
			require.Equal(t, 0, isLoop)
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, items, values)
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

		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "items", name)

		// Verify isLoop column exists
		rows, err := db.Query("PRAGMA table_info(items)")
		require.NoError(t, err)
		defer rows.Close()

		isLoopExists := false
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dfltValue sql.NullString
			err = rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk)
			require.NoError(t, err)
			if name == "isLoop" {
				isLoopExists = true
				break
			}
		}
		require.NoError(t, rows.Err())
		require.True(t, isLoopExists, "isLoop column not found in items table")
	})

	t.Run("InitializationWithItems", func(t *testing.T) {
		t.Parallel()
		items := []string{"test1", "test2"}
		db, err := InitializeDatabase("file::memory:", items)
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		// Verify items were inserted
		rows, err := db.Query("SELECT value, isLoop FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			var isLoop int
			err := rows.Scan(&value, &isLoop)
			require.NoError(t, err)
			require.Equal(t, 0, isLoop)
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, items, values)

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
		items := []string{"item1", "item2"}
		reader, err := InitializeItem("file::memory:", items)
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.NotNil(t, reader.DB)
		require.Equal(t, "file::memory:", reader.DBPath)
		defer reader.DB.Close()

		// Verify items were inserted
		rows, err := reader.DB.Query("SELECT value, isLoop FROM items ORDER BY id")
		require.NoError(t, err)
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			var isLoop int
			err := rows.Scan(&value, &isLoop)
			require.NoError(t, err)
			require.Equal(t, 0, isLoop)
			values = append(values, value)
		}
		require.NoError(t, rows.Err())
		require.Equal(t, items, values)

		// Verify record count
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
	})
}
