package schema

import (
	"context"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const mockLockedVersion = "0.2.30" // Define the version once and reuse it
	const mockVersion = "0.2.30"       // Define the version once and reuse it

	// Save the original value of UseLatest to avoid test interference
	originalUseLatest := UseLatest
	defer func() { UseLatest = originalUseLatest }()

	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		t.Parallel()

		// Ensure UseLatest is set to false and mock behavior is consistent
		UseLatest = false
		result := SchemaVersion(ctx)

		assert.Equal(t, mockVersion, result, "expected the specified version to be returned when UseLatest is false")
	})

	t.Run("returns latest version when UseLatest is true", func(t *testing.T) {
		t.Parallel()

		// Save original state
		originalFetcher := utils.GitHubReleaseFetcher
		originalCachedVersion := cachedVersion
		defer func() {
			utils.GitHubReleaseFetcher = originalFetcher
			cachedVersion = originalCachedVersion
		}()

		// Reset global state for this test
		once = sync.Once{}
		cachedVersion = ""

		// Mock GitHubReleaseFetcher to return a specific version for testing
		utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
			return mockLockedVersion, nil // Use the reusable constant
		}

		UseLatest = true
		result := SchemaVersion(ctx)

		assert.Equal(t, mockLockedVersion, result, "expected the latest version to be returned when UseLatest is true")
	})
}

func TestSchemaVersionCaching(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Save the original values to avoid test interference
	originalUseLatest := UseLatest
	originalCachedVersion := cachedVersion
	originalFetcher := utils.GitHubReleaseFetcher
	defer func() {
		UseLatest = originalUseLatest
		cachedVersion = originalCachedVersion
		utils.GitHubReleaseFetcher = originalFetcher
	}()

	// Reset the sync.Once for this test (note: this is a bit hacky but necessary for testing)
	once = sync.Once{}
	cachedVersion = ""

	callCount := 0
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		callCount++
		return "cached-version", nil
	}

	UseLatest = true

	// First call should fetch from GitHub
	result1 := SchemaVersion(ctx)
	assert.Equal(t, "cached-version", result1)
	assert.Equal(t, 1, callCount, "Expected GitHubReleaseFetcher to be called once")

	// Second call should use cached version
	result2 := SchemaVersion(ctx)
	assert.Equal(t, "cached-version", result2)
	assert.Equal(t, 1, callCount, "Expected GitHubReleaseFetcher to still be called only once (cached)")
}

func TestSchemaVersionSpecifiedVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Save the original values to avoid test interference
	originalUseLatest := UseLatest
	originalSpecifiedVersion := specifiedVersion
	defer func() {
		UseLatest = originalUseLatest
		specifiedVersion = originalSpecifiedVersion
	}()

	// Test with different specified version
	specifiedVersion = "1.0.0"
	UseLatest = false

	result := SchemaVersion(ctx)
	assert.Equal(t, "1.0.0", result)
}
