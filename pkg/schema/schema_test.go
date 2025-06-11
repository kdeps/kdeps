package schema

import (
	"context"
	"errors"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
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

func TestSchemaVersionErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Save original values
	originalUseLatest := UseLatest
	originalExitFunc := exitFunc
	defer func() {
		UseLatest = originalUseLatest
		exitFunc = originalExitFunc
	}()

	UseLatest = true
	versionCache.Delete("version")

	// Mock exitFunc to prevent actual exit
	exitCalled := false
	exitFunc = func(code int) {
		exitCalled = true
	}

	// Mock GitHubReleaseFetcher to return error
	originalFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
		return "", assert.AnError
	}
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	// Call SchemaVersion
	SchemaVersion(ctx)

	// Verify exit was called
	assert.True(t, exitCalled, "expected exit to be called on error")
}

func TestSchemaVersionCachedValue(t *testing.T) {
	ctx := context.Background()

	// Save original value
	originalUseLatest := UseLatest
	defer func() { UseLatest = originalUseLatest }()

	UseLatest = true

	// Pre-populate cache
	testVersion := "1.2.3"
	versionCache.Store("version", testVersion)

	// Call SchemaVersion
	result := SchemaVersion(ctx)

	// Verify cached value was used
	assert.Equal(t, testVersion, result, "expected cached version to be used")
}

// TestSchemaVersionSpecified ensures the function returns the hard-coded version when UseLatest is false.
func TestSchemaVersionSpecified(t *testing.T) {
	// Preserve global state and restore afterwards
	origLatest := UseLatest
	origSpecified := specifiedVersion
	defer func() {
		UseLatest = origLatest
		specifiedVersion = origSpecified
	}()

	UseLatest = false
	specifiedVersion = "9.9.9"

	ver := SchemaVersion(context.Background())
	assert.Equal(t, "9.9.9", ver)
}

// TestSchemaVersionLatestSuccess exercises the successful latest-fetch path.
func TestSchemaVersionLatestSuccess(t *testing.T) {
	// Save globals
	origLatest := UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	defer func() {
		UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		versionCache.Delete("version")
	}()

	UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "1.2.3", nil
	}

	ctx := context.Background()

	ver1 := SchemaVersion(ctx)
	assert.Equal(t, "1.2.3", ver1)
	// Second call should hit cache and not invoke fetcher again
	ver2 := SchemaVersion(ctx)
	assert.Equal(t, "1.2.3", ver2)
}

// TestSchemaVersionLatestFailure hits the error branch and verifies exitFunc is called.
func TestSchemaVersionLatestFailure(t *testing.T) {
	origLatest := UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origExit := exitFunc
	defer func() {
		UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		exitFunc = origExit
	}()

	UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "", errors.New("network error")
	}

	var code int
	exitFunc = func(c int) { code = c }

	SchemaVersion(context.Background())
	assert.Equal(t, 1, code)
}
