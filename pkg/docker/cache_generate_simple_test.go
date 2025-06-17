package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

func TestGenerateURLsBasic(t *testing.T) {
	ctx := context.Background()
	// Ensure deterministic behaviour
	schema.UseLatest = false

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("GenerateURLs returned no items")
	}
	for _, it := range items {
		if it.URL == "" {
			t.Fatalf("item has empty URL")
		}
		if it.LocalName == "" {
			t.Fatalf("item has empty LocalName")
		}
	}
}
