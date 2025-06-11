package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestGitHubRelease(t *testing.T) {
	t.Parallel()

	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"tag_name": "v2.1.0"}`)
	}))
	defer server.Close()

	result, err := GetLatestGitHubRelease(context.Background(), "kdeps/schema", server.URL)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", result)
}

func TestGetLatestGitHubReleaseSuccess(t *testing.T) {
	// Create test server returning fake release
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	ver, err := GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	assert.NoError(t, err)
	assert.Equal(t, "1.2.3", ver)
}

func TestGetLatestGitHubReleaseError(t *testing.T) {
	// Server returns non-200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()
	ver, err := GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	assert.Error(t, err)
	assert.Empty(t, ver)
}
