package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentArchitecture(t *testing.T) {
	ctx := context.Background()

	t.Run("DefaultMapping", func(t *testing.T) {
		arch := GetCurrentArchitecture(ctx, "unknown")
		assert.NotEmpty(t, arch)
	})

	t.Run("ApplePKLMapping", func(t *testing.T) {
		arch := GetCurrentArchitecture(ctx, "apple/pkl")
		assert.NotEmpty(t, arch)
	})
}

func TestCompareVersions(t *testing.T) {
	ctx := context.Background()

	t.Run("Greater", func(t *testing.T) {
		assert.True(t, CompareVersions(ctx, "2.0.0", "1.0.0"))
	})

	t.Run("Less", func(t *testing.T) {
		assert.False(t, CompareVersions(ctx, "1.0.0", "2.0.0"))
	})

	t.Run("Equal", func(t *testing.T) {
		assert.False(t, CompareVersions(ctx, "1.0.0", "1.0.0"))
	})
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

func TestGetLatestAnacondaVersions(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<a href="Anaconda3-2024.10-1-Linux-x86_64.sh">Anaconda3-2024.10-1-Linux-x86_64.sh</a>`))
		}))
		defer server.Close()

		versions, err := GetLatestAnacondaVersions(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, versions)
	})

	t.Run("NoVersions", func(t *testing.T) {
		t.Skip("Skipping test that requires HTTP client mocking")
	})
}

func TestBuildURL(t *testing.T) {
	url := buildURL("https://example.com/{version}/{arch}", "1.0.0", "amd64")
	assert.Equal(t, "https://example.com/1.0.0/amd64", url)
}

func TestGenerateURLs(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		items, err := GenerateURLs(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, items)
	})

	t.Run("UseLatest", func(t *testing.T) {
		schema.UseLatest = true
		defer func() { schema.UseLatest = false }()

		items, err := GenerateURLs(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, items)
	})
}
