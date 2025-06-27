package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBasicDockerFunctions tests basic Docker functions for coverage
func TestBasicDockerFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("CompareVersions", func(t *testing.T) {
		result := CompareVersions(ctx, "1.0.0", "2.0.0")
		assert.False(t, result)

		result2 := CompareVersions(ctx, "2.0.0", "1.0.0")
		assert.True(t, result2)
	})

	t.Run("ParseVersion", func(t *testing.T) {
		result := ParseVersion("1.2.3")
		assert.Equal(t, []int{1, 2, 3}, result)

		invalid := ParseVersion("invalid")
		assert.True(t, len(invalid) == 0 || len(invalid) == 1) // Can return empty or [0]
	})

	t.Run("BuildURL", func(t *testing.T) {
		result := BuildURL("http://example.com", "1.0", "x86")
		assert.Contains(t, result, "example.com")
	})

	t.Run("GenerateURLs", func(t *testing.T) {
		urls, err := GenerateURLs(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, urls)
	})
}
