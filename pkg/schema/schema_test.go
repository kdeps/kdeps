package schema_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/stretchr/testify/assert"
)

var (
	testMutex   sync.Mutex
	schemaMutex sync.Mutex
)

func withSchemaTestState(t *testing.T, fn func()) {
	testMutex.Lock()
	defer testMutex.Unlock()
	origUseLatest := schema.UseLatest
	// Instead of copying VersionCache, just clear it after
	defer func() {
		schema.UseLatest = origUseLatest
		clearSyncMap(&schema.VersionCache)
	}()
	fn()
}

// Mutex-protected helpers for global variable mutation
var schemaGlobalsMutex sync.Mutex

func saveAndRestoreSchemaGlobals(t *testing.T, useLatest bool) func() {
	schemaGlobalsMutex.Lock()
	origUseLatest := schema.UseLatest
	// Instead of copying VersionCache, just clear it after
	schema.UseLatest = useLatest
	return func() {
		schema.UseLatest = origUseLatest
		clearSyncMap(&schema.VersionCache)
		schemaGlobalsMutex.Unlock()
	}
}

func saveAndRestoreGitHubReleaseFetcher(t *testing.T, newFetcher func(ctx context.Context, repo, baseURL string) (string, error)) func() {
	schemaGlobalsMutex.Lock()
	origFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = newFetcher
	return func() {
		utils.GitHubReleaseFetcher = origFetcher
		schemaGlobalsMutex.Unlock()
	}
}

func saveAndRestoreExitFunc(t *testing.T, newExit func(int)) func() {
	schemaGlobalsMutex.Lock()
	origExit := schema.ExitFunc
	schema.ExitFunc = newExit
	return func() {
		schema.ExitFunc = origExit
		schemaGlobalsMutex.Unlock()
	}
}

func TestVersion(t *testing.T) {
	ctx := context.Background()

	// Save the original value of UseLatest to avoid test interference
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		schema.UseLatest = false
		result := schema.Version(ctx)
		assert.Equal(t, "v1", result, "expected default schema version")
	})

	t.Run("caches and returns latest version when UseLatest is true", func(t *testing.T) {
		schema.UseLatest = true
		// Clear any existing cache
		clearSyncMap(&schema.VersionCache)

		// First call should fetch and cache
		result1 := schema.Version(ctx)
		assert.NotEmpty(t, result1, "expected non-empty version")

		// Second call should use cache
		result2 := schema.Version(ctx)
		assert.Equal(t, result1, result2, "expected cached version")

		// Verify it's in cache
		cached, ok := schema.VersionCache.Load("version")
		assert.True(t, ok, "expected version to be cached")
		assert.Equal(t, result1, cached.(string), "cached version mismatch")
	})
}

func TestVersionSpecifiedVersion(t *testing.T) {
	ctx := context.Background()
	schema.UseLatest = false

	result := schema.Version(ctx)
	assert.Equal(t, "v1", result, "expected default schema version")
}

func TestVersionCaching(t *testing.T) {
	ctx := context.Background()
	schema.UseLatest = true

	// Clear any existing cache
	clearSyncMap(&schema.VersionCache)

	// First call should fetch and cache
	result1 := schema.Version(ctx)
	assert.NotEmpty(t, result1, "expected non-empty version")

	// Second call should use cache
	result2 := schema.Version(ctx)
	assert.Equal(t, result1, result2, "expected cached version")

	// Verify it's in cache
	cached, ok := schema.VersionCache.Load("version")
	assert.True(t, ok, "expected version to be cached")
	assert.Equal(t, result1, cached.(string), "cached version mismatch")
}

func TestSchemaVersionErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Save original values
	originalUseLatest := schema.UseLatest
	originalExitFunc := schema.ExitFunc
	defer func() {
		schema.UseLatest = originalUseLatest
		schema.ExitFunc = originalExitFunc
	}()

	schema.UseLatest = true
	clearSyncMap(&schema.VersionCache)

	// Mock exitFunc to prevent actual exit
	exitCalled := false
	schema.ExitFunc = func(code int) {
		exitCalled = true
	}

	// Mock GitHubReleaseFetcher to return error
	originalFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
		return "", assert.AnError
	}
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	// Call Version
	schema.Version(ctx)

	// Verify exit was called
	assert.True(t, exitCalled, "expected exit to be called on error")
}

func TestSchemaVersionCachedValue(t *testing.T) {
	ctx := context.Background()

	// Save original value
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	schema.UseLatest = true

	// Pre-populate cache
	testVersion := "1.2.3"
	schema.VersionCache.Store("version", testVersion)

	// Call Version
	result := schema.Version(ctx)

	// Verify cached value was used
	assert.Equal(t, testVersion, result, "expected cached version to be used")
}

// TestSchemaVersionSpecified ensures the function returns the default schema version when UseLatest is false.
func TestSchemaVersionSpecified(t *testing.T) {
	// Preserve global state and restore afterwards
	origLatest := schema.UseLatest
	defer func() {
		schema.UseLatest = origLatest
	}()

	schema.UseLatest = false

	ver := schema.Version(context.Background())
	assert.Equal(t, "v1", ver)
}

// TestSchemaVersionLatestSuccess exercises the successful latest-fetch path.
func TestSchemaVersionLatestSuccess(t *testing.T) {
	// Save globals
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		clearSyncMap(&schema.VersionCache)
	}()

	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "1.2.3", nil
	}

	ctx := context.Background()

	ver1 := schema.Version(ctx)
	assert.Equal(t, "1.2.3", ver1)
	// Second call should hit cache and not invoke fetcher again
	ver2 := schema.Version(ctx)
	assert.Equal(t, "1.2.3", ver2)
}

// TestSchemaVersionLatestFailure hits the error branch and verifies exitFunc is called.
func TestSchemaVersionLatestFailure(t *testing.T) {
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origExit := schema.ExitFunc
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		schema.ExitFunc = origExit
	}()

	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "", errors.New("network error")
	}

	var code int
	schema.ExitFunc = func(c int) { code = c }

	schema.Version(context.Background())
	assert.Equal(t, 1, code)
}

// TestSchemaVersionSpecified verifies that when UseLatest is false the
// function returns the compile-time default schema version without making any
// external fetch calls.
func TestSchemaVersionSpecifiedExtra(t *testing.T) {
	// Ensure we start from a clean slate.
	schema.UseLatest = false
	clearSyncMap(&schema.VersionCache)

	got := schema.Version(context.Background())
	if got != "v1" {
		t.Fatalf("expected DefaultSchemaVersion %s, got %s", "v1", got)
	}
}

// TestSchemaVersionLatestCaching ensures that when UseLatest is true the
// version is fetched via GitHubReleaseFetcher exactly once and then served
// from the cache on subsequent invocations.
func TestSchemaVersionLatestCachingExtra(t *testing.T) {
	// Prepare stub fetcher.
	fetchCount := 0
	oldFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		fetchCount++
		return "1.2.3", nil
	}
	defer func() { utils.GitHubReleaseFetcher = oldFetcher }()

	// Activate latest mode and clear cache.
	schema.UseLatest = true
	clearSyncMap(&schema.VersionCache)

	ctx := context.Background()
	first := schema.Version(ctx)
	second := schema.Version(ctx)

	if first != "1.2.3" || second != "1.2.3" {
		t.Fatalf("unexpected versions returned: %s and %s", first, second)
	}
	if fetchCount != 1 {
		t.Fatalf("GitHubReleaseFetcher should be called once, got %d", fetchCount)
	}
}

func TestVersion_WithExitFunc(t *testing.T) {
	withSchemaTestState(t, func() {
		// Test with UseLatest = true
		schema.UseLatest = true
		clearSyncMap(&schema.VersionCache)

		// Mock exitFunc to prevent actual exit
		exitCalled := false
		schema.ExitFunc = func(code int) {
			exitCalled = true
		}

		// Mock GitHubReleaseFetcher to return error
		utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
			return "", assert.AnError
		}

		// Call Version
		schema.Version(context.Background())

		// Verify exit was called
		assert.True(t, exitCalled, "expected exit to be called on error")

		// Test with UseLatest = false
		schema.UseLatest = false
		clearSyncMap(&schema.VersionCache)

		got := schema.Version(context.Background())
		if got != version.DefaultSchemaVersion {
			t.Fatalf("expected DefaultSchemaVersion %s, got %s", version.DefaultSchemaVersion, got)
		}
	})
}

func TestVersion_WithGitHubFetcher(t *testing.T) {
	withSchemaTestState(t, func() {
		schema.UseLatest = true
		clearSyncMap(&schema.VersionCache)

		utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
			return "1.2.3", nil
		}

		ctx := context.Background()

		ver1 := schema.Version(ctx)
		assert.Equal(t, "1.2.3", ver1)
		// Second call should hit cache and not invoke fetcher again
		ver2 := schema.Version(ctx)
		assert.Equal(t, "1.2.3", ver2)
	})
}

// Instead of copying sync.Map, clear it in place for test setup/teardown.
func clearSyncMap(m *sync.Map) {
	m.Range(func(key, value any) bool {
		m.Delete(key)
		return true
	})
}

func TestSchemaVersion(t *testing.T) {
	restore := saveAndRestoreSchemaGlobals(t, false)
	defer restore()

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version)
}

func TestSchemaVersionWithLatest(t *testing.T) {
	restore := saveAndRestoreSchemaGlobals(t, true)
	defer restore()

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version)
}

func TestSchemaVersionWithError(t *testing.T) {
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	schema.UseLatest = true

	// Mock the fetcher to return an error
	originalFetcher := utils.GitHubReleaseFetcher
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
		return "", errors.New("mock error")
	}

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version) // Should fall back to default version
}

func TestSchemaVersionWithExit(t *testing.T) {
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	schema.UseLatest = true

	// Mock the fetcher to trigger exit
	originalFetcher := utils.GitHubReleaseFetcher
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "", errors.New("mock error")
	}

	// Mock exit function
	originalExitFunc := schema.ExitFunc
	defer func() { schema.ExitFunc = originalExitFunc }()

	schema.UseLatest = true

	schema.ExitFunc = func(exitCode int) {
		// Capture exit code but don't actually exit
		_ = exitCode // Use the parameter to avoid unused variable warning
	}

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version)
}

func TestSchemaVersionWithLatestAndError(t *testing.T) {
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	originalFetcher := utils.GitHubReleaseFetcher
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	schema.UseLatest = true

	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "", errors.New("mock error")
	}

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version)
}

func TestSchemaVersionWithExitAndError(t *testing.T) {
	originalUseLatest := schema.UseLatest
	defer func() { schema.UseLatest = originalUseLatest }()

	originalFetcher := utils.GitHubReleaseFetcher
	defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

	originalExitFunc := schema.ExitFunc
	defer func() { schema.ExitFunc = originalExitFunc }()

	schema.UseLatest = true

	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "", errors.New("mock error")
	}

	schema.ExitFunc = func(code int) {
		// Capture exit code but don't actually exit
	}

	version := schema.Version(context.Background())
	assert.NotEmpty(t, version)
}

func TestSchemaVersionWithCache(t *testing.T) {
	oldFetcher := utils.GitHubReleaseFetcher
	defer func() { utils.GitHubReleaseFetcher = oldFetcher }()

	restore := saveAndRestoreSchemaGlobals(t, true)

	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "v1.2.3", nil
	}

	// First call should fetch
	version1 := schema.Version(context.Background())
	assert.Equal(t, "v1.2.3", version1)

	// Second call should use cache
	version2 := schema.Version(context.Background())
	assert.Equal(t, "v1.2.3", version2)

	restore()
}

func TestSchemaVersionWithCacheHit(t *testing.T) {
	restore := saveAndRestoreSchemaGlobals(t, true)

	// Mock the fetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
		return "v2.0.0", nil
	}

	// First call
	version1 := schema.Version(context.Background())
	assert.Equal(t, "v2.0.0", version1)

	// Second call should use cache
	version2 := schema.Version(context.Background())
	assert.Equal(t, "v2.0.0", version2)

	restore()
}

func TestSchemaVersionWithCacheMiss(t *testing.T) {
	restore := saveAndRestoreSchemaGlobals(t, true)

	// Mock the fetcher to return different versions
	callCount := 0
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		callCount++
		if callCount == 1 {
			return "v1.0.0", nil
		}
		return "v2.0.0", nil
	}

	// First call
	version1 := schema.Version(context.Background())
	assert.Equal(t, "v1.0.0", version1)

	// Second call should use cache (same version)
	version2 := schema.Version(context.Background())
	assert.Equal(t, "v1.0.0", version2)

	restore()
}

func TestSchemaVersionWithCacheClear(t *testing.T) {
	restore := saveAndRestoreSchemaGlobals(t, true)

	// Mock the fetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, _ string) (string, error) {
		return "v3.0.0", nil
	}

	// First call
	version1 := schema.Version(context.Background())
	assert.Equal(t, "v3.0.0", version1)

	// Clear cache
	clearSyncMap(&schema.VersionCache)

	// Second call should fetch again
	version2 := schema.Version(context.Background())
	assert.Equal(t, "v3.0.0", version2)

	restore()
}
