package docker

import (
	"context"
	"testing"
)

func TestGenerateURLs(t *testing.T) {
	ctx := context.Background()

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("unexpected error generating urls: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one download item")
	}

	for _, itm := range items {
		if itm.URL == "" || itm.LocalName == "" {
			t.Errorf("item fields should not be empty: %+v", itm)
		}
	}
}
