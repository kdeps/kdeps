package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLatestGitHubRelease(t *testing.T) {
	var ctx context.Context

	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"tag_name": "v2.1.0"}`)
	}))
	defer server.Close()

	result, err := GetLatestGitHubRelease(ctx, "kdeps/schema", server.URL)
	assert.NoError(t, err)
	assert.Equal(t, "2.1.0", result)
}
