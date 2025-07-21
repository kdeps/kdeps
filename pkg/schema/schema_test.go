package schema_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	// returns specified version when UseLatest is false
	result := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    false,
		Fetcher:      func(context.Context, string, string) (string, error) { return "", nil },
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "0.4.6", result, "expected default schema version")

	// caches and returns latest version when UseLatest is true
	cache = sync.Map{}
	fetchCount := 0
	fetcher := func(context.Context, string, string) (string, error) {
		fetchCount++
		return "v2", nil
	}
	result1 := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "v2", result1, "expected fetched version")
	result2 := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, result1, result2, "expected cached version")
	assert.Equal(t, 1, fetchCount, "fetcher should be called once")
}

func TestVersion_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	exitCalled := false
	fetcher := func(context.Context, string, string) (string, error) {
		return "", assert.AnError
	}
	schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) { exitCalled = true },
		VersionCache: &cache,
	})
	assert.True(t, exitCalled, "expected exit to be called on error")
}

func TestVersion_CachedValue(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	cache.Store("version", "1.2.3")
	result := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      func(context.Context, string, string) (string, error) { return "should not call", nil },
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "1.2.3", result, "expected cached version to be used")
}

func TestVersion_FetcherErrorFallback(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	exitCalled := false
	fetcher := func(context.Context, string, string) (string, error) {
		return "", errors.New("mock error")
	}
	schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) { exitCalled = true },
		VersionCache: &cache,
	})
	assert.True(t, exitCalled, "expected exit to be called on error")
}

func TestVersion_DefaultSchemaVersion(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	result := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    false,
		Fetcher:      func(context.Context, string, string) (string, error) { return "should not call", nil },
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "0.4.6", result, "expected default schema version")
}

func TestVersion_CacheClear(t *testing.T) {
	ctx := context.Background()
	var cache sync.Map
	fetcher := func(context.Context, string, string) (string, error) {
		return "v3.0.0", nil
	}
	// First call
	result1 := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "v3.0.0", result1)
	// Clear cache
	cache = sync.Map{}
	// Second call should fetch again
	result2 := schema.VersionWithDeps(ctx, schema.VersionDeps{
		UseLatest:    true,
		Fetcher:      fetcher,
		ExitFunc:     func(int) {},
		VersionCache: &cache,
	})
	assert.Equal(t, "v3.0.0", result2)
}
