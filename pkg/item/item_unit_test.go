package item

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitializeItem_NewAPI tests the new InitializeItem function with actionID
func TestInitializeItem_NewAPI(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		items       []string
		actionID    string
		expectError bool
	}{
		{
			name:        "valid initialization with empty items",
			dbPath:      ":memory:",
			items:       []string{},
			actionID:    "test-action-1",
			expectError: false,
		},
		{
			name:        "valid initialization with items",
			dbPath:      ":memory:",
			items:       []string{"item1", "item2"},
			actionID:    "test-action-2",
			expectError: false,
		},
		{
			name:        "empty actionID",
			dbPath:      ":memory:",
			items:       []string{"item1"},
			actionID:    "",
			expectError: false, // ActionID can be empty for initialization
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := InitializeItem(tt.dbPath, tt.items, tt.actionID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)
				assert.NotNil(t, reader.DB)
				assert.Equal(t, tt.dbPath, reader.DBPath)
				assert.Equal(t, tt.actionID, reader.ActionID)

				// Clean up
				reader.DB.Close()
			}
		})
	}
}

// TestPklResourceReader_NewAPI_Read tests the Read method with new operations
func TestPklResourceReader_NewAPI_Read(t *testing.T) {
	// Set up a test database with items
	reader, err := InitializeItem(":memory:", []string{"initial1", "initial2"}, "test-action")
	require.NoError(t, err)
	defer reader.DB.Close()

	tests := []struct {
		name        string
		uri         string
		expectError bool
		contains    string
	}{
		{
			name:        "updateCurrent operation",
			uri:         "item:?op=updateCurrent&value=newitem",
			expectError: false,
			contains:    "newitem",
		},
		{
			name:        "current operation",
			uri:         "item:?op=current",
			expectError: false,
			contains:    "", // May be empty if no current item for this actionID
		},
		{
			name:        "set operation (append to results)",
			uri:         "item:?op=set&value=result1",
			expectError: false,
			contains:    "result1",
		},
		{
			name:        "values operation",
			uri:         "item:/test-action?op=values",
			expectError: false,
			contains:    "", // May be empty if no results for this actionID
		},
		{
			name:        "lastResult operation",
			uri:         "item:?op=lastResult",
			expectError: false,
			contains:    "", // May be empty if no results
		},
		{
			name:        "prev operation",
			uri:         "item:?op=prev",
			expectError: false,
			contains:    "", // May be empty
		},
		{
			name:        "next operation",
			uri:         "item:?op=next",
			expectError: false,
			contains:    "", // May be empty
		},
		{
			name:        "invalid operation",
			uri:         "item:?op=invalid",
			expectError: false, // Returns empty but no error in new API
			contains:    "",
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
			}
		})
	}
}

// TestPklResourceReader_NewAPI_Interfaces tests the interface methods
func TestPklResourceReader_NewAPI_Interfaces(t *testing.T) {
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

// TestPklResourceReader_NewAPI_Close tests the Close method
func TestPklResourceReader_NewAPI_Close(t *testing.T) {
	reader, err := InitializeItem(":memory:", []string{"test"}, "test-action")
	require.NoError(t, err)

	// Close should work without error
	err = reader.Close()
	assert.NoError(t, err)

	// DB should be nil after close
	assert.Nil(t, reader.DB)

	// Calling close again should not error
	err = reader.Close()
	assert.NoError(t, err)
}

// TestInitializeDatabase_NewAPI tests the InitializeDatabase function
func TestInitializeDatabase_NewAPI(t *testing.T) {
	tests := []struct {
		name        string
		dbPath      string
		items       []string
		expectError bool
	}{
		{
			name:        "memory database with items",
			dbPath:      ":memory:",
			items:       []string{"item1", "item2", "item3"},
			expectError: false,
		},
		{
			name:        "memory database empty",
			dbPath:      ":memory:",
			items:       []string{},
			expectError: false,
		},
		{
			name:        "memory database nil items",
			dbPath:      ":memory:",
			items:       nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := InitializeDatabase(tt.dbPath, tt.items)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)

				// Clean up
				db.Close()
			}
		})
	}
}
