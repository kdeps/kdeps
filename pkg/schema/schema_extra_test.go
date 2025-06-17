package schema

import (
	"context"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

// TestSchemaVersionSpecified verifies that when UseLatest is false the
// function returns the compile-time specifiedVersion without making any
// external fetch calls.
func TestSchemaVersionSpecifiedExtra(t *testing.T) {
	// Ensure we start from a clean slate.
	UseLatest = false
	versionCache = sync.Map{}

	got := SchemaVersion(context.Background())
	if got != specifiedVersion {
		t.Fatalf("expected specifiedVersion %s, got %s", specifiedVersion, got)
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
	UseLatest = true
	versionCache = sync.Map{}

	ctx := context.Background()
	first := SchemaVersion(ctx)
	second := SchemaVersion(ctx)

	if first != "1.2.3" || second != "1.2.3" {
		t.Fatalf("unexpected versions returned: %s and %s", first, second)
	}
	if fetchCount != 1 {
		t.Fatalf("GitHubReleaseFetcher should be called once, got %d", fetchCount)
	}
}
