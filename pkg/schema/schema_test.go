package schema

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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

	// Reset the sync.Once for this test
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

	// Reset the sync.Once and cached version for the second test
	once = sync.Once{}
	cachedVersion = ""

	// Second call should fetch again since we reset the cache
	result2 := SchemaVersion(ctx)
	assert.Equal(t, "cached-version", result2)
	assert.Equal(t, 2, callCount, "Expected GitHubReleaseFetcher to be called twice")
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

func TestSchemaVersionError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Save the original values to avoid test interference
	originalUseLatest := UseLatest
	originalCachedVersion := cachedVersion
	originalFetcher := utils.GitHubReleaseFetcher
	originalExitFunc := exitFunc
	defer func() {
		UseLatest = originalUseLatest
		cachedVersion = originalCachedVersion
		utils.GitHubReleaseFetcher = originalFetcher
		exitFunc = originalExitFunc
	}()

	// Reset the sync.Once for this test
	once = sync.Once{}
	cachedVersion = ""

	// Mock GitHubReleaseFetcher to return an error
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "", fmt.Errorf("mock error")
	}

	UseLatest = true

	// Capture os.Stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	// Override exitFunc to prevent os.Exit
	exited := false
	exitFunc = func(code int) {
		exited = true
		w.Close() // Close the writer to unblock the reader
	}

	// Call SchemaVersion (should trigger exitFunc)
	SchemaVersion(ctx)

	// Read the error message
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify the error message and that exitFunc was called
	assert.True(t, exited, "exitFunc should have been called")
	assert.Contains(t, buf.String(), "Error: Unable to fetch the latest schema version")
}
