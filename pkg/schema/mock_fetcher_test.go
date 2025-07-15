package schema_test

import (
	"context"

	"github.com/kdeps/kdeps/pkg/utils"
)

func init() {
	// Mock the GitHubReleaseFetcher for testing
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "1.2.3", nil
	}
}
