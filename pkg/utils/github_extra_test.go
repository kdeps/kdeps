package utils_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestGitHubReleaseExtra(t *testing.T) {
	ctx := context.Background()

	// Success path
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
		resp := map[string]string{"tag_name": "v1.2.3"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	v, err := utils.GetLatestGitHubRelease(ctx, "owner/repo", ts.URL)
	require.NoError(t, err)
	require.Equal(t, "1.2.3", v)

	// Unauthorized path
	ts401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts401.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts401.URL)
	require.Error(t, err)

	// Non-OK generic error path
	ts500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts500.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts500.URL)
	require.Error(t, err)

	// Forbidden path (rate limit)
	ts403 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts403.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts403.URL)
	require.Error(t, err)

	// Malformed JSON path – should error on JSON parse
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{ "tag_name": 123 }`)) // tag_name not string
	}))
	defer tsBadJSON.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", tsBadJSON.URL)
	require.Error(t, err)
}

func TestGetLatestGitHubReleaseMore(t *testing.T) {
	t.Skip("covered by TestGetLatestGitHubReleaseExtra")

	/* Keeping code for reference but skipping execution
	ctx := context.Background()

	// helper to run a single scenario
	run := func(status int, body string, expectErr bool, expectVer string) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		defer srv.Close()

		ver, err := GetLatestGitHubRelease(ctx, "kdeps/schema", srv.URL)
		if expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, expectVer, ver)
		}
	}

	// 1) success path
	run(http.StatusOK, `{"tag_name":"v1.2.3"}`, false, "1.2.3")

	// 2) unexpected status
	run(http.StatusInternalServerError, "boom", true, "")

	// 3) bad JSON
	run(http.StatusOK, `{"tag":"v0.0.1"}`, true, "")
	*/
}

// TestGetLatestGitHubReleaseWithToken verifies the Authorization header is set
// when GITHUB_TOKEN environment variable is present.
func TestGetLatestGitHubReleaseWithToken(t *testing.T) {
	ctx := context.Background()

	token := "dummy-token"
	t.Setenv("GITHUB_TOKEN", token)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		require.Equal(t, "Bearer "+token, auth)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v9.9.9"}`))
	}))
	defer srv.Close()

	ver, err := utils.GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "9.9.9", ver)
}

// TestGetLatestGitHubReleaseInvalidURL ensures that malformed URLs trigger an error
func TestGetLatestGitHubReleaseInvalidURL(t *testing.T) {
	ctx := context.Background()
	ver, err := utils.GetLatestGitHubRelease(ctx, "owner/repo", "://bad url")
	require.Error(t, err)
	assert.Empty(t, ver)
}

// TestGetLatestGitHubRelease_Success verifies the helper parses tag names and
// strips the leading 'v'.
func TestGetLatestGitHubRelease_Success(t *testing.T) {
	// Spin up mock GitHub API endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	version, err := utils.GetLatestGitHubRelease(ctx, "octocat/Hello-World", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("unexpected version: %s", version)
	}

	_ = schema.SchemaVersion(ctx)
}

// TestGetLatestGitHubRelease_Errors checks status-code error branches.
func TestGetLatestGitHubRelease_Errors(t *testing.T) {
	tests := []struct {
		code int
	}{
		{http.StatusUnauthorized},
		{http.StatusForbidden},
		{http.StatusTeapot}, // arbitrary non-200
	}

	ctx := context.Background()

	for _, tc := range tests {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
		}))
		version, err := utils.GetLatestGitHubRelease(ctx, "octocat/Hello-World", srv.URL)
		srv.Close()
		if err == nil {
			t.Fatalf("expected error for status %d, got version %s", tc.code, version)
		}
	}

	_ = schema.SchemaVersion(ctx)
}
