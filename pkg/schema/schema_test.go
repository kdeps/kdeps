package schema

import (
	"testing"

	"kdeps/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		UseLatest = false
		result := SchemaVersion()
		assert.Equal(t, "0.1.46", result, "expected the specified version to be returned")
	})

	t.Run("returns latest version when UseLatest is true", func(t *testing.T) {
		// Mock GitHubReleaseFetcher to return a fixed version
		utils.GitHubReleaseFetcher = func(repo, baseURL string) (string, error) {
			return "0.1.46", nil
		}
		defer func() { utils.GitHubReleaseFetcher = utils.GetLatestGitHubRelease }()

		UseLatest = true
		result := SchemaVersion()
		assert.Equal(t, "0.1.46", result, "expected the latest version to be returned")
	})
}
