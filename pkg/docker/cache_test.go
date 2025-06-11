package docker

import (
	"context"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentArchitecture(t *testing.T) {
	ctx := context.Background()

	var expected string
	if archMap, ok := archMappings["apple/pkl"]; ok {
		if mapped, exists := archMap[runtime.GOARCH]; exists {
			expected = mapped
		}
	}
	// Fallback to default mapping only if apple/pkl did not contain entry
	if expected == "" {
		if defaultMap, ok := archMappings["default"]; ok {
			if mapped, exists := defaultMap[runtime.GOARCH]; exists {
				expected = mapped
			}
		}
	}
	if expected == "" {
		expected = runtime.GOARCH
	}

	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	assert.Equal(t, expected, arch)
}

func TestCompareVersions(t *testing.T) {
	ctx := context.Background()

	assert.True(t, CompareVersions(ctx, "2.0.0", "1.9.9"))
	assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.0"))
	assert.False(t, CompareVersions(ctx, "1.2.3", "1.2.4"))
	// Mixed length versions
	assert.True(t, CompareVersions(ctx, "1.2.3", "1.2"))
	assert.False(t, CompareVersions(ctx, "1.2", "1.2.3"))
}

func TestParseVersion(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		parts := parseVersion("1.2.3")
		assert.Equal(t, []int{1, 2, 3}, parts)
	})

	t.Run("WithHyphen", func(t *testing.T) {
		parts := parseVersion("1-2-3")
		assert.Equal(t, []int{1, 2, 3}, parts)
	})
}

func TestBuildURL(t *testing.T) {
	base := "https://example.com/download/{version}/app-{arch}"
	url := buildURL(base, "1.0.0", "x86_64")
	assert.Equal(t, "https://example.com/download/1.0.0/app-x86_64", url)
}

func TestGenerateURLs_DefaultVersion(t *testing.T) {
	// Ensure we are not in latest mode to avoid network calls
	schemaUseLatestBackup := schema.UseLatest
	schema.UseLatest = false
	defer func() { schema.UseLatest = schemaUseLatestBackup }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx)
	assert.NoError(t, err)
	assert.Greater(t, len(items), 0)

	// verify each item has URL and LocalName populated
	for _, item := range items {
		assert.NotEmpty(t, item.URL)
		assert.NotEmpty(t, item.LocalName)
	}
}
