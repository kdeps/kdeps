package utils

import (
	"context"
	"fmt"
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
