package docker

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

// TestGenerateURLsDefault verifies that GenerateURLs returns the expected
// download items when schema.UseLatest is false.
func TestGenerateURLsDefault(t *testing.T) {
	ctx := context.Background()

	// Ensure we are testing the static version path.
	original := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = original }()

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}

	// We expect exactly two download targets (PKL + Anaconda).
	if len(items) != 2 {
		t.Fatalf("expected 2 download items, got %d", len(items))
	}

	// Basic sanity checks on the returned structure.
	for _, itm := range items {
		if !strings.HasPrefix(itm.URL, "https://") {
			t.Errorf("URL does not start with https: %s", itm.URL)
		}
		if itm.LocalName == "" {
			t.Errorf("LocalName should not be empty for item %+v", itm)
		}
	}

	// Reference the schema version as required by testing rules.
	_ = schema.SchemaVersion(ctx)
}
