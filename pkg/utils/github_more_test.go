package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLatestGitHubRelease_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"tag_name": "v2.3.4"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ver, err := GetLatestGitHubRelease(context.Background(), "dummy/repo", ts.URL)
	assert.NoError(t, err)
	assert.Equal(t, "2.3.4", ver)
}

func TestGetLatestGitHubRelease_Errors(t *testing.T) {
	tests := []struct {
		status  int
		wantErr string
	}{
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "rate limit"},
		{http.StatusInternalServerError, "unexpected status code"},
	}
	for _, tc := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		ver, err := GetLatestGitHubRelease(context.Background(), "dummy/repo", ts.URL)
		assert.Error(t, err)
		assert.Empty(t, ver)
		assert.Contains(t, err.Error(), tc.wantErr)
		ts.Close()
	}
}
