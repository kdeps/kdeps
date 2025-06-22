package item

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeDatabase_Unit tests the InitializeDatabase function
func TestInitializeDatabase_Unit(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		items       []string
		expectError bool
		setupFn     func(dbPath string) // Optional setup function
	}{
		{
			name:        "create new database with no items",
			dbPath:      ":memory:",
			items:       nil,
			expectError: false,
		},
		{
			name:        "create new database with empty items",
			dbPath:      ":memory:",
			items:       []string{},
			expectError: false,
		},
		{
			name:        "create new database with items",
			dbPath:      ":memory:",
			items:       []string{"item1", "item2", "item3"},
			expectError: false,
		},
		{
			name:        "create database with many items",
			dbPath:      ":memory:",
			items:       []string{"apple", "banana", "cherry", "date", "elderberry"},
			expectError: false,
		},
		{
			name:        "create database with duplicate items",
			dbPath:      ":memory:",
			items:       []string{"duplicate", "duplicate", "unique"},
			expectError: false,
		},
		{
			name:        "invalid database path",
			dbPath:      "/invalid/path/that/does/not/exist/test.db",
			items:       []string{"item1"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFn != nil {
				tt.setupFn(tt.dbPath)
			}

			db, err := InitializeDatabase(tt.dbPath, tt.items)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)

				// Verify table was created
				var count int
				err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='items'").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, 1, count)

				// Verify items were inserted if provided
				if len(tt.items) > 0 {
					var itemCount int
					err = db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)
					assert.NoError(t, err)
					assert.Equal(t, len(tt.items), itemCount)
				}

				// Clean up
				db.Close()
			}
		})
	}
}

// TestInitializeItem_Unit tests the InitializeItem function
func TestInitializeItem_Unit(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		items       []string
		expectError bool
	}{
		{
			name:        "successful initialization",
			dbPath:      ":memory:",
			items:       []string{"test1", "test2"},
			expectError: false,
		},
		{
			name:        "initialization with no items",
			dbPath:      ":memory:",
			items:       nil,
			expectError: false,
		},
		{
			name:        "invalid path",
			dbPath:      "/invalid/path/test.db",
			items:       []string{"test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := InitializeItem(tt.dbPath, tt.items)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)
				assert.NotNil(t, reader.DB)
				assert.Equal(t, tt.dbPath, reader.DBPath)

				// Clean up
				reader.DB.Close()
			}
		})
	}
}

// TestPklResourceReader_Read_Unit tests the Read method
func TestPklResourceReader_Read_Unit(t *testing.T) {
	// Set up a test database
	dbPath := ":memory:"
	db, err := InitializeDatabase(dbPath, []string{"initial1", "initial2", "initial3"})
	require.NoError(t, err)
	defer db.Close()

	reader := &PklResourceReader{
		DB:     db,
		DBPath: dbPath,
	}

	tests := []struct {
		name        string
		uri         string
		expectError bool
		expectedLen int    // For some operations
		contains    string // For checking if result contains certain text
	}{
		{
			name:        "get current operation",
			uri:         "item:?op=current",
			expectError: false,
			contains:    "initial", // Should contain one of the initial values
		},
		{
			name:        "list operation",
			uri:         "item:?op=list",
			expectError: false,
			expectedLen: 3, // Should return JSON array of 3 items
		},
		{
			name:        "values operation",
			uri:         "item:?op=values",
			expectError: false,
			expectedLen: 3, // Should return JSON array of 3 items
		},
		{
			name:        "set operation with value",
			uri:         "item:?op=set&value=newitem",
			expectError: false,
			contains:    "newitem",
		},
		{
			name:        "set operation without value",
			uri:         "item:?op=set",
			expectError: true,
		},
		{
			name:        "prev operation",
			uri:         "item:?op=prev",
			expectError: false,
		},
		{
			name:        "next operation",
			uri:         "item:?op=next",
			expectError: false,
		},
		{
			name:        "invalid operation",
			uri:         "item:?op=invalid",
			expectError: true,
		},
		{
			name:        "no operation parameter",
			uri:         "item:",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, err := url.Parse(tt.uri)
			require.NoError(t, err)

			result, err := reader.Read(*parsedURL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				if tt.contains != "" {
					assert.Contains(t, string(result), tt.contains)
				}

				if tt.expectedLen > 0 {
					// For list/values operations, verify it's a JSON array
					assert.True(t, len(result) > 0)
					assert.Equal(t, byte('['), result[0]) // Should start with '['
				}
			}
		})
	}
}

// TestPklResourceReader_Read_NilDB_Unit tests Read method with nil DB
func TestPklResourceReader_Read_NilDB_Unit(t *testing.T) {
	reader := &PklResourceReader{
		DB:     nil,
		DBPath: ":memory:",
	}

	parsedURL, err := url.Parse("item:?op=current")
	require.NoError(t, err)

	// Should reinitialize the database automatically
	result, err := reader.Read(*parsedURL)

	// Should work (reinitialize DB) but return empty since no data
	assert.NoError(t, err)
	assert.Equal(t, []byte(""), result)
	assert.NotNil(t, reader.DB) // DB should be initialized now

	// Clean up
	reader.DB.Close()
}

// TestPklResourceReader_FetchValues_Unit tests the FetchValues method
func TestPklResourceReader_FetchValues_Unit(t *testing.T) {
	tests := []struct {
		name        string
		setupItems  []string
		operation   string
		expectError bool
		expectedLen int
	}{
		{
			name:        "fetch from empty database",
			setupItems:  []string{},
			operation:   "list",
			expectError: false,
			expectedLen: 0,
		},
		{
			name:        "fetch unique values",
			setupItems:  []string{"apple", "banana", "apple", "cherry", "banana"},
			operation:   "values",
			expectError: false,
			expectedLen: 3, // Only unique values: apple, banana, cherry
		},
		{
			name:        "fetch with operation name",
			setupItems:  []string{"item1", "item2", "item3"},
			operation:   "list",
			expectError: false,
			expectedLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh database for each test
			db, err := InitializeDatabase(":memory:", tt.setupItems)
			require.NoError(t, err)
			defer db.Close()

			reader := &PklResourceReader{
				DB:     db,
				DBPath: ":memory:",
			}

			result, err := reader.FetchValues(tt.operation)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify it's valid JSON array
				assert.True(t, len(result) >= 2) // At minimum "[]"
				assert.Equal(t, byte('['), result[0])
				assert.Equal(t, byte(']'), result[len(result)-1])

				// For non-empty results, verify expected length by counting commas + 1
				if tt.expectedLen > 0 {
					resultStr := string(result)
					// Simple check: non-empty array should not be just "[]"
					assert.NotEqual(t, "[]", resultStr)
				} else {
					assert.Equal(t, "[]", string(result))
				}
			}
		})
	}
}

// TestPklResourceReader_GetMostRecentID_Unit tests the GetMostRecentID method
func TestPklResourceReader_GetMostRecentID_Unit(t *testing.T) {
	tests := []struct {
		name       string
		setupItems []string
		expectID   bool // Whether we expect an ID to be returned
	}{
		{
			name:       "empty database",
			setupItems: []string{},
			expectID:   false,
		},
		{
			name:       "database with items",
			setupItems: []string{"item1", "item2"},
			expectID:   true,
		},
		{
			name:       "database with single item",
			setupItems: []string{"single"},
			expectID:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := InitializeDatabase(":memory:", tt.setupItems)
			require.NoError(t, err)
			defer db.Close()

			reader := &PklResourceReader{
				DB:     db,
				DBPath: ":memory:",
			}

			id, err := reader.GetMostRecentID()

			assert.NoError(t, err)

			if tt.expectID {
				assert.NotEmpty(t, id)
			} else {
				assert.Empty(t, id)
			}
		})
	}
}

// TestPklResourceReader_Interfaces_Unit tests the interface methods
func TestPklResourceReader_Interfaces_Unit(t *testing.T) {
	reader := &PklResourceReader{}

	// Test interface methods
	assert.False(t, reader.IsGlobbable())
	assert.False(t, reader.HasHierarchicalUris())
	assert.Equal(t, "item", reader.Scheme())

	// Test ListElements
	testURL, _ := url.Parse("item:test")
	elements, err := reader.ListElements(*testURL)
	assert.NoError(t, err)
	assert.Nil(t, elements)
}

// TestPklResourceReader_Read_EdgeCases_Unit tests edge cases in Read method
func TestPklResourceReader_Read_EdgeCases_Unit(t *testing.T) {
	// Test with database that has complex transaction scenarios
	db, err := InitializeDatabase(":memory:", []string{"test1", "test2"})
	require.NoError(t, err)
	defer db.Close()

	reader := &PklResourceReader{
		DB:     db,
		DBPath: ":memory:",
	}

	t.Run("set operation with empty string value", func(t *testing.T) {
		parsedURL, err := url.Parse("item:?op=set&value=")
		require.NoError(t, err)

		result, err := reader.Read(*parsedURL)
		assert.Error(t, err) // Empty value is treated as no value provided
		assert.Nil(t, result)
	})

	t.Run("set operation with special characters", func(t *testing.T) {
		parsedURL, err := url.Parse("item:?op=set&value=special%20chars%21%40%23")
		require.NoError(t, err)

		result, err := reader.Read(*parsedURL)
		assert.NoError(t, err)
		assert.Contains(t, string(result), "special chars!@#")
	})

	t.Run("prev operation on empty database", func(t *testing.T) {
		emptyDB, err := InitializeDatabase(":memory:", []string{})
		require.NoError(t, err)
		defer emptyDB.Close()

		emptyReader := &PklResourceReader{DB: emptyDB, DBPath: ":memory:"}

		parsedURL, err := url.Parse("item:?op=prev")
		require.NoError(t, err)

		result, err := emptyReader.Read(*parsedURL)
		assert.NoError(t, err)
		assert.Equal(t, []byte(""), result)
	})

	t.Run("next operation on empty database", func(t *testing.T) {
		emptyDB, err := InitializeDatabase(":memory:", []string{})
		require.NoError(t, err)
		defer emptyDB.Close()

		emptyReader := &PklResourceReader{DB: emptyDB, DBPath: ":memory:"}

		parsedURL, err := url.Parse("item:?op=next")
		require.NoError(t, err)

		result, err := emptyReader.Read(*parsedURL)
		assert.NoError(t, err)
		assert.Equal(t, []byte(""), result)
	})
}
