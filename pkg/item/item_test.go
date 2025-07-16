package item_test

import (
	"net/url"
	"testing"

	"github.com/kdeps/kdeps/pkg/item"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	// Use in-memory SQLite database for testing
	dbPath := "file::memory:"
	reader, err := item.InitializeItem(dbPath, nil)
	require.NoError(t, err)
	defer reader.DB.Close()

	t.Run("Scheme", func(t *testing.T) {
		require.Equal(t, "item", reader.Scheme())
	})

	t.Run("Read_GetRecord", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
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

	t.Run("Read_SetRecord", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("item:/_?op=set&value=newvalue")
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
		uri, _ = url.Parse("item:/_?op=set")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "set operation requires a value parameter")
	})

	t.Run("Read_PrevRecord", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
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
		reader, err := item.InitializeItem("file::memory:", nil)
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
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'value1'),
			('20250101120001.000000', 'value2'),
			('20250101120002.000000', 'value1')  -- Duplicate value
		`)
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["value1","value2"]`), data) // Unique values

		_, err = reader.DB.Exec("DELETE FROM items")
		require.NoError(t, err)
		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), data)
	})

	t.Run("Read_Values", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Test empty database
		uri, _ := url.Parse("item:/_?op=values")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`[]`), data)

		// Test with records including duplicates
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'value1'),
			('20250101120001.000000', 'value2'),
			('20250101120002.000000', 'value1')
		`)
		require.NoError(t, err)

		data, err = reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["value1","value2"]`), data) // Unique values
	})

	t.Run("InitializeWithItems", func(t *testing.T) {
		items := []string{"item1", "item2", "item1"} // Includes duplicate
		reader, err := item.InitializeItem("file::memory:", items)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Verify items were inserted
		uri, _ := url.Parse("item:/_?op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(`["item1","item2"]`), data) // Unique values

		// Verify record count (all items inserted, even duplicates)
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
	})
}

func TestInitializeDatabase(t *testing.T) {
	t.Run("SuccessfulInitialization", func(t *testing.T) {
		db, err := item.InitializeDatabase("file::memory:", []string{})
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='items'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "items", name)
	})

	t.Run("InitializationWithItems", func(t *testing.T) {
		items := []string{"test1", "test2", "test1"}
		db, err := item.InitializeDatabase("file::memory:", items)
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
	t.Run("WithoutItems", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
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
		items := []string{"item1", "item2", "item1"}
		reader, err := item.InitializeItem("file::memory:", items)
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

// Additional unit tests for comprehensive coverage

func TestPklResourceReader_InterfaceMethods(t *testing.T) {
	reader := &item.PklResourceReader{}

	t.Run("IsGlobbable", func(t *testing.T) {
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("item:/_")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})
}

func TestRead_ErrorCases(t *testing.T) {
	t.Run("InvalidOperation", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		uri, _ := url.Parse("item:/_?op=invalid")
		_, err = reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid operation")
	})

	t.Run("DatabaseReinitialization", func(t *testing.T) {
		reader := &item.PklResourceReader{
			DB:     nil,
			DBPath: "file::memory:",
		}

		uri, _ := url.Parse("item:/_?op=current")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
		require.NotNil(t, reader.DB)
		defer reader.DB.Close()
	})

	t.Run("DatabaseInitializationFailure", func(t *testing.T) {
		reader := &item.PklResourceReader{
			DB:     nil,
			DBPath: "/invalid/path/database.db",
		}

		uri, _ := url.Parse("item:/_?op=current")
		_, err := reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to initialize database")
	})
}

func TestGetMostRecentID_EdgeCases(t *testing.T) {
	t.Run("EmptyDatabase", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		id, err := reader.GetMostRecentID()
		require.NoError(t, err)
		require.Empty(t, id)
	})
}

func TestFetchValues_EdgeCases(t *testing.T) {
	t.Run("EmptyDatabase", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		result, err := reader.FetchValues("test")
		require.NoError(t, err)
		require.Equal(t, []byte("[]"), result)
	})
}

func TestRead_TransactionErrorPaths(t *testing.T) {
	t.Run("SetRecord_DatabaseClosed", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		reader.DB.Close() // Close database to simulate failure

		uri, _ := url.Parse("item:/_?op=set&value=test")
		data, err := reader.Read(*uri)
		// The implementation automatically reconnects to the database, so this should succeed
		require.NoError(t, err)
		require.Equal(t, []byte("test"), data)
		// Verify the database was reconnected
		require.NotNil(t, reader.DB)
	})
}

func TestInitializeDatabase_ErrorCases(t *testing.T) {
	t.Run("InvalidDatabasePath", func(t *testing.T) {
		_, err := item.InitializeDatabase("/invalid/path/database.db", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to ping database")
	})
}

func TestInitializeItem_ErrorCases(t *testing.T) {
	t.Run("DatabaseInitializationFailure", func(t *testing.T) {
		_, err := item.InitializeItem("/invalid/path/database.db", []string{"test"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error initializing database")
	})
}

func TestRead_NavigationEdgeCases(t *testing.T) {
	t.Run("PrevRecord_NoEarlierRecord", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert only one record
		_, err = reader.DB.Exec("INSERT INTO items (id, value) VALUES (?, ?)", "20250101120000.000000", "value1")
		require.NoError(t, err)

		uri, _ := url.Parse("item:/_?op=prev")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data)
	})

	t.Run("NextRecord_HasNextRecord", func(t *testing.T) {
		reader, err := item.InitializeItem("file::memory:", nil)
		require.NoError(t, err)
		defer reader.DB.Close()

		// Insert records where the most recent is not the latest chronologically
		_, err = reader.DB.Exec(`
			INSERT INTO items (id, value) VALUES
			('20250101120000.000000', 'value1'),
			('20250101120002.000000', 'value3'),
			('20250101120001.000000', 'value2')
		`)
		require.NoError(t, err)

		// The most recent ID should be the highest value
		uri, _ := url.Parse("item:/_?op=next")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte(""), data) // No next record after the highest ID
	})
}
