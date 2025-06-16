package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestCompareAndParseVersion(t *testing.T) {
	ctx := context.Background()
	assert.True(t, CompareVersions(ctx, "2.0.0", "1.9.9"))
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.1"))
	// equal
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.0"))

	got := parseVersion("1.2.3-alpha")
	assert.Equal(t, []int{1, 2, 3, 0}, got, "non numeric suffixed parts become 0")
}

func TestGenerateURLs_Static(t *testing.T) {
	schema.UseLatest = false
	items, err := GenerateURLs(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, items)
	// Ensure each local name contains arch or version placeholders replaced
	for _, it := range items {
		assert.NotContains(t, it.LocalName, "{", "template placeholders should be resolved")
		assert.NotEmpty(t, it.URL)
	}
}
