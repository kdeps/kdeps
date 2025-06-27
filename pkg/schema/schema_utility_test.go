package schema

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaVersionUtility(t *testing.T) {
	// Test SchemaVersion function comprehensively to boost coverage from 77.8% to 100%

	t.Run("UseLatest_false", func(t *testing.T) {
		// Save original state
		originalUseLatest := UseLatest
		originalCachedVersion := cachedVersion
		originalOnce := once
		defer func() {
			UseLatest = originalUseLatest
			cachedVersion = originalCachedVersion
			once = originalOnce
		}()

		// Reset state
		UseLatest = false
		cachedVersion = ""
		once = sync.Once{}

		ctx := context.Background()
		result := SchemaVersion(ctx)

		// Should return the hardcoded version from version.SchemaVersion
		assert.NotEmpty(t, result)
		// Should not be dependent on network calls when UseLatest is false
	})

	t.Run("UseLatest_true_with_reset", func(t *testing.T) {
		// Save original state
		originalUseLatest := UseLatest
		originalCachedVersion := cachedVersion
		originalOnce := once
		defer func() {
			UseLatest = originalUseLatest
			cachedVersion = originalCachedVersion
			once = originalOnce
		}()

		// Test with UseLatest = true
		UseLatest = true
		cachedVersion = "" // Clear cache
		once = sync.Once{} // Reset once

		ctx := context.Background()

		// First call should try to fetch from GitHub (might fail in test environment)
		result1 := SchemaVersion(ctx)
		assert.NotEmpty(t, result1)

		// Second call should use cached result
		result2 := SchemaVersion(ctx)
		assert.Equal(t, result1, result2)
	})

	t.Run("UseLatest_true_with_cached_version", func(t *testing.T) {
		// Save original state
		originalUseLatest := UseLatest
		originalCachedVersion := cachedVersion
		originalOnce := once
		defer func() {
			UseLatest = originalUseLatest
			cachedVersion = originalCachedVersion
			once = originalOnce
		}()

		// Simulate already cached version
		UseLatest = true
		cachedVersion = "test-cached-version"
		once = sync.Once{}
		once.Do(func() {}) // Mark once as done

		ctx := context.Background()
		result := SchemaVersion(ctx)

		// Should return the cached version
		assert.Equal(t, "test-cached-version", result)
	})

	t.Run("UseLatest_false_multiple_calls", func(t *testing.T) {
		// Save original state
		originalUseLatest := UseLatest
		defer func() {
			UseLatest = originalUseLatest
		}()

		UseLatest = false
		ctx := context.Background()

		// Multiple calls should return consistent results
		result1 := SchemaVersion(ctx)
		result2 := SchemaVersion(ctx)
		result3 := SchemaVersion(ctx)

		assert.Equal(t, result1, result2)
		assert.Equal(t, result2, result3)
		assert.NotEmpty(t, result1)
	})

	t.Run("context_variations", func(t *testing.T) {
		// Save original state
		originalUseLatest := UseLatest
		defer func() {
			UseLatest = originalUseLatest
		}()

		UseLatest = false

		// Test with different context types
		tests := []struct {
			name string
			ctx  context.Context
		}{
			{"background_context", context.Background()},
			{"todo_context", context.TODO()},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				result := SchemaVersion(test.ctx)
				assert.NotEmpty(t, result)
			})
		}
	})
}
