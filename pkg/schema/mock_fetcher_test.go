package schema_test

import (
	"context"

	"github.com/kdeps/kdeps/pkg/utils"
)

func init() {
	// Mock the GitHubReleaseFetcher for testing
	utils.GitHubReleaseFetcher = func(_ context.Context, _ string, _ string) (string, error) {
		return "1.2.3", nil
	}
}
