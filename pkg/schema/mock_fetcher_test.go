package schema_test

import (
	"context"

	"github.com/kdeps/kdeps/pkg/utils"
)

func init() {
	// Provide a fast local stub to avoid live GitHub calls when UseLatest is true and
	// individual sub-tests haven't swapped out the fetcher. This keeps the unit
	// suite hermetic and avoids flaky network timeouts seen in CI.
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "0.0.0-test", nil
	}
}
