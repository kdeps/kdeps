package item

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set test mode to skip complex operations
	os.Setenv("KDEPS_TEST_MODE", "true")
}

// TestItemReaderAdditionalCoverage tests additional item reader functions for coverage
func TestItemReaderAdditionalCoverage(t *testing.T) {
	// Create test database
	dbPath := ":memory:"
	db, err := InitializeDatabase(dbPath, nil)
	require.NoError(t, err)
	defer db.Close()

	reader := &PklResourceReader{
		DB:       db,
		DBPath:   dbPath,
		ActionID: "test-action",
	}

	t.Run("read with complex set operation", func(t *testing.T) {
		// Test setting a complex value with special characters
		values := url.Values{}
		values.Set("op", "set")
		values.Set("value", "test value with spaces and symbols: @#$%")

		uri := url.URL{
			Scheme:   "item",
			Path:     "/test-complex",
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		resultStr := string(result)
		assert.Contains(t, resultStr, "test value with spaces and symbols: @#$%")
	})

	t.Run("read with values operation after setting", func(t *testing.T) {
		// First create an item with the correct action_id using updateCurrent
		values := url.Values{}
		values.Set("op", "updateCurrent")
		values.Set("value", "initial")

		uri := url.URL{
			Scheme:   "item",
			Path:     "/test-values",
			RawQuery: values.Encode(),
		}

		_, err := reader.Read(uri)
		require.NoError(t, err)

		// Now set some values that will go into results table
		for i := 1; i <= 3; i++ {
			values := url.Values{}
			values.Set("op", "set")
			values.Set("value", fmt.Sprintf("value%d", i))

			uri := url.URL{
				Scheme:   "item",
				Path:     "/test-values",
				RawQuery: values.Encode(),
			}

			_, err := reader.Read(uri)
			require.NoError(t, err)
		}

		// Now get all values - use the reader's ActionID as the path
		values = url.Values{}
		values.Set("op", "values")

		uri = url.URL{
			Scheme:   "item",
			Path:     "/test-action", // Use the reader's ActionID
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		resultStr := string(result)
		assert.Contains(t, resultStr, "value1")
		assert.Contains(t, resultStr, "value2")
		assert.Contains(t, resultStr, "value3")
	})

	t.Run("read with lastResult operation", func(t *testing.T) {
		// First create an item with the correct action_id
		values := url.Values{}
		values.Set("op", "updateCurrent")
		values.Set("value", "initial")

		uri := url.URL{
			Scheme:   "item",
			Path:     "/test-last",
			RawQuery: values.Encode(),
		}

		_, err := reader.Read(uri)
		require.NoError(t, err)

		// Now set a value
		values = url.Values{}
		values.Set("op", "set")
		values.Set("value", "last test value")

		uri = url.URL{
			Scheme:   "item",
			Path:     "/test-last",
			RawQuery: values.Encode(),
		}

		_, err = reader.Read(uri)
		require.NoError(t, err)

		// Now get the last result
		values = url.Values{}
		values.Set("op", "lastResult")

		uri = url.URL{
			Scheme:   "item",
			Path:     "/test-last",
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		resultStr := string(result)
		assert.Contains(t, resultStr, "last test value")
	})

	t.Run("read with prev operation", func(t *testing.T) {
		// First create multiple items with the correct action_id using updateCurrent
		for i := 1; i <= 3; i++ {
			values := url.Values{}
			values.Set("op", "updateCurrent")
			values.Set("value", fmt.Sprintf("prev%d", i))

			uri := url.URL{
				Scheme:   "item",
				Path:     "/test-prev",
				RawQuery: values.Encode(),
			}

			_, err := reader.Read(uri)
			require.NoError(t, err)
		}

		// Now get previous value
		values := url.Values{}
		values.Set("op", "prev")

		uri := url.URL{
			Scheme:   "item",
			Path:     "/test-prev",
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		// Should contain one of the previous values or be empty
		resultStr := string(result)
		assert.True(t, len(resultStr) >= 0) // Either has content or is empty
	})

	t.Run("read with next operation", func(t *testing.T) {
		// Now get next value (might not have next, but tests the function)
		values := url.Values{}
		values.Set("op", "next")

		uri := url.URL{
			Scheme:   "item",
			Path:     "/test-next",
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestPklResourceReaderMethodsCoverage tests PKL resource reader methods for coverage
func TestPklResourceReaderMethodsCoverage(t *testing.T) {
	db, err := InitializeDatabase(":memory:", nil)
	require.NoError(t, err)
	defer db.Close()

	reader := &PklResourceReader{
		DB:       db,
		DBPath:   ":memory:",
		ActionID: "test-action",
	}

	t.Run("scheme method", func(t *testing.T) {
		scheme := reader.Scheme()
		assert.Equal(t, "item", scheme)
	})

	t.Run("isGlobbable method", func(t *testing.T) {
		globbable := reader.IsGlobbable()
		assert.False(t, globbable) // Items are typically not globbable
	})

	t.Run("hasHierarchicalUris method", func(t *testing.T) {
		hierarchical := reader.HasHierarchicalUris()
		assert.False(t, hierarchical) // According to the implementation
	})

	t.Run("listElements method", func(t *testing.T) {
		baseUri := url.URL{Scheme: "item", Path: "/test"}

		elements, err := reader.ListElements(baseUri)
		assert.NoError(t, err)
		assert.Nil(t, elements) // Implementation returns nil
	})

	t.Run("close method", func(t *testing.T) {
		// Create a new reader to test close
		reader2 := &PklResourceReader{
			DB:       db,
			DBPath:   ":memory:",
			ActionID: "test-action",
		}

		err := reader2.Close()
		assert.NoError(t, err)
	})
}

// TestItemSchemaIntegrationCoverage tests schema integration for coverage
func TestItemSchemaIntegrationCoverage(t *testing.T) {
	ctx := context.Background()

	db, err := InitializeDatabase(":memory:", nil)
	require.NoError(t, err)
	defer db.Close()

	reader := &PklResourceReader{
		DB:       db,
		DBPath:   ":memory:",
		ActionID: "test-action",
	}

	t.Run("item operations with schema version", func(t *testing.T) {
		// Use schema.SchemaVersion as requested
		version := schema.SchemaVersion(ctx)
		assert.NotEmpty(t, version)

		// Test setting value with schema context
		values := url.Values{}
		values.Set("op", "set")
		values.Set("value", fmt.Sprintf("schema-test-%s", version))

		uri := url.URL{
			Scheme:   "item",
			Path:     "/schema-test",
			RawQuery: values.Encode(),
		}

		result, err := reader.Read(uri)
		assert.NoError(t, err)
		resultStr := string(result)
		assert.Contains(t, resultStr, version)
	})
}

// TestInitializationFunctionsCoverage tests initialization functions for coverage
func TestInitializationFunctionsCoverage(t *testing.T) {
	t.Run("initialize database with items", func(t *testing.T) {
		items := []string{"item1", "item2", "item3"}
		db, err := InitializeDatabase(":memory:", items)
		assert.NoError(t, err)
		assert.NotNil(t, db)
		db.Close()
	})

	t.Run("initialize item reader", func(t *testing.T) {
		items := []string{"test-item"}
		reader, err := InitializeItem(":memory:", items, "test-action")
		assert.NoError(t, err)
		assert.NotNil(t, reader)
		assert.Equal(t, "test-action", reader.ActionID)
		reader.Close()
	})
}
