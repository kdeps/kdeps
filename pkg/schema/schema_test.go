package schema

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const mockLockedVersion = "0.2.14" // Define the version once and reuse it
	const mockVersion = "0.2.14"       // Define the version once and reuse it

	// Save the original value of UseLatest to avoid test interference
	originalUseLatest := UseLatest
	defer func() { UseLatest = originalUseLatest }()

	t.Run("returns specified version when UseLatest is false", func(t *testing.T) {
		t.Parallel()

		// Ensure UseLatest is set to false and mock behavior is consistent
		UseLatest = false
		result := SchemaVersion(ctx)

		assert.Equal(t, mockVersion, result, "expected the specified version to be returned when UseLatest is false")
	})

	t.Run("returns latest version when UseLatest is true", func(t *testing.T) {
		t.Parallel()

		// Mock GitHubReleaseFetcher to return a specific version for testing
		originalFetcher := utils.GitHubReleaseFetcher
		utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
			return mockLockedVersion, nil // Use the reusable constant
		}
		defer func() { utils.GitHubReleaseFetcher = originalFetcher }()

		UseLatest = true
		result := SchemaVersion(ctx)

		assert.Equal(t, mockLockedVersion, result, "expected the latest version to be returned when UseLatest is true")
	})
}
