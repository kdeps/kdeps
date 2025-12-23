package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestOllamaVersion(t *testing.T) {
	t.Run("Success - Returns latest semantic version", func(t *testing.T) {
		// Mock Docker Hub response
		mockResponse := map[string]interface{}{
			"results": []map[string]string{
				{"name": "0.13.5"},
				{"name": "0.13.4"},
				{"name": "0.13.3"},
				{"name": "latest"},
				{"name": "0.13.2-rc1"},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResponse)
		}))
		defer server.Close()

		// Temporarily replace the URL in the function
		// Since we can't easily inject URL, we'll test the real API
		ctx := context.Background()
		version, err := GetLatestOllamaVersion(ctx)

		require.NoError(t, err)
		assert.NotEmpty(t, version)
		assert.Regexp(t, `^\d+\.\d+\.\d+$`, version, "Version should be in semver format")
	})

	t.Run("Error - Invalid context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := GetLatestOllamaVersion(ctx)
		assert.Error(t, err)
	})
}

func TestCompareSemanticVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"v1 greater - major", "2.0.0", "1.0.0", 1},
		{"v1 greater - minor", "1.2.0", "1.1.0", 1},
		{"v1 greater - patch", "1.0.2", "1.0.1", 1},
		{"v2 greater - major", "1.0.0", "2.0.0", -1},
		{"v2 greater - minor", "1.1.0", "1.2.0", -1},
		{"v2 greater - patch", "1.0.1", "1.0.2", -1},
		{"equal versions", "1.2.3", "1.2.3", 0},
		{"complex comparison", "0.13.5", "0.13.4", 1},
		{"large version numbers", "10.20.30", "9.20.30", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSemanticVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected [3]int
	}{
		{"simple version", "1.2.3", [3]int{1, 2, 3}},
		{"zeros", "0.0.0", [3]int{0, 0, 0}},
		{"large numbers", "10.20.30", [3]int{10, 20, 30}},
		{"latest ollama", "0.13.5", [3]int{0, 13, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSemVer(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test - only runs with network access
func TestGetLatestOllamaVersionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	version, err := GetLatestOllamaVersion(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, version)
	assert.Regexp(t, `^\d+\.\d+\.\d+$`, version)

	// Version should be at least 0.13.0 (as of Dec 2024)
	parts := parseSemVer(version)
	assert.True(t, parts[0] >= 0, "Major version should be >= 0")
	assert.True(t, parts[1] >= 13, "Minor version should be >= 13")
}
