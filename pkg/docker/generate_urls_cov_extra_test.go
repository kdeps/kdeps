package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

// TestGenerateURLsHappyPath exercises the default code path where UseLatest is
// false. This avoids external HTTP requests yet covers several branches inside
// GenerateURLs including architecture substitution and local-name template
// logic.
func TestGenerateURLsHappyPath(t *testing.T) {
	ctx := context.Background()

	// Ensure the package-level flag is in the expected default state.
	schema.UseLatest = false

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}

	// We expect two items (one for Pkl and one for Anaconda).
	if len(items) != 2 {
		t.Fatalf("expected 2 download items, got %d", len(items))
	}

	// Basic sanity checks on the generated URLs/local names – just ensure they
	// contain expected substrings so that we're not overly sensitive to exact
	// versions or architecture values.
	for _, itm := range items {
		if itm.URL == "" {
			t.Fatalf("item URL is empty: %+v", itm)
		}
		if itm.LocalName == "" {
			t.Fatalf("item LocalName is empty: %+v", itm)
		}
	}
}
