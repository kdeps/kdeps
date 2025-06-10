package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	ctx := context.Background()

	// Save the original value of UseLatest to avoid test interference
	originalUseLatest := UseLatest
	defer func() { UseLatest = originalUseLatest }()

	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		UseLatest = false
		result := SchemaVersion(ctx)
		assert.Equal(t, specifiedVersion, result, "expected specified version")
	})

	t.Run("caches and returns latest version when UseLatest is true", func(t *testing.T) {
		UseLatest = true
		// Clear any existing cache
		versionCache.Delete("version")

		// First call should fetch and cache
		result1 := SchemaVersion(ctx)
		assert.NotEmpty(t, result1, "expected non-empty version")

		// Second call should use cache
		result2 := SchemaVersion(ctx)
		assert.Equal(t, result1, result2, "expected cached version")

		// Verify it's in cache
		cached, ok := versionCache.Load("version")
		assert.True(t, ok, "expected version to be cached")
		assert.Equal(t, result1, cached.(string), "cached version mismatch")
	})
}

func TestSchemaVersionSpecifiedVersion(t *testing.T) {
	ctx := context.Background()
	UseLatest = false

	result := SchemaVersion(ctx)
	assert.Equal(t, specifiedVersion, result, "expected specified version")
}

func TestSchemaVersionCaching(t *testing.T) {
	ctx := context.Background()
	UseLatest = true

	// Clear any existing cache
	versionCache.Delete("version")

	// First call should fetch and cache
	result1 := SchemaVersion(ctx)
	assert.NotEmpty(t, result1, "expected non-empty version")

	// Second call should use cache
	result2 := SchemaVersion(ctx)
	assert.Equal(t, result1, result2, "expected cached version")

	// Verify it's in cache
	cached, ok := versionCache.Load("version")
	assert.True(t, ok, "expected version to be cached")
	assert.Equal(t, result1, cached.(string), "cached version mismatch")
}
