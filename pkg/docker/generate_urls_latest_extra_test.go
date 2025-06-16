package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
)

func TestGenerateURLsLatestUsesFetcher(t *testing.T) {
	ctx := context.Background()

	// Save globals and restore afterwards
	orig := schema.UseLatest
	fetchOrig := utils.GitHubReleaseFetcher
	defer func() {
		schema.UseLatest = orig
		utils.GitHubReleaseFetcher = fetchOrig
	}()

	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "0.99.0", nil
	}

	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	found := false
	for _, it := range items {
		if it.LocalName == "pkl-linux-latest-"+GetCurrentArchitecture(ctx, "apple/pkl") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pkl latest local name element, got %+v", items)
	}
}
