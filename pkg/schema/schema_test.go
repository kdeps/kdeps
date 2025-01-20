package schema

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	var ctx context.Context

	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		UseLatest = false
		result := SchemaVersion(ctx)
		assert.Equal(t, "0.1.46", result, "expected the specified version to be returned")
	})

	t.Run("returns latest version when UseLatest is true", func(t *testing.T) {
		// Mock GitHubReleaseFetcher to return a fixed version
		utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
			return "0.1.46", nil
		}
		defer func() { utils.GitHubReleaseFetcher = utils.GetLatestGitHubRelease }()

		UseLatest = true
		result := SchemaVersion(ctx)
		assert.Equal(t, "0.1.46", result, "expected the latest version to be returned")
	})
}
