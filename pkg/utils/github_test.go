package utils_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package variable mutexes for safe reassignment
var (
	httpDefaultTransportMutex sync.Mutex
	httpDefaultClientMutex    sync.Mutex
)

// Helper functions to safely save and restore package variables
func saveAndRestoreHTTPDefaultTransport(_ *testing.T, newTransport http.RoundTripper) func() {
	httpDefaultTransportMutex.Lock()
	original := http.DefaultTransport
	http.DefaultTransport = newTransport
	return func() {
		http.DefaultTransport = original
		httpDefaultTransportMutex.Unlock()
	}
}

func saveAndRestoreHTTPDefaultClient(_ *testing.T, newClient *http.Client) func() {
	httpDefaultClientMutex.Lock()
	original := http.DefaultClient
	http.DefaultClient = newClient
	return func() {
		http.DefaultClient = original
		httpDefaultClientMutex.Unlock()
	}
}

// testHTTPState manages test state changes for http package
type testHTTPState struct {
	origTransport http.RoundTripper
	origClient    *http.Client
}

func newTestHTTPState() *testHTTPState {
	return &testHTTPState{
		origTransport: http.DefaultTransport,
		origClient:    http.DefaultClient,
	}
}

func (ts *testHTTPState) restore() {
	http.DefaultTransport = ts.origTransport
	http.DefaultClient = ts.origClient
}

func withTestState(_ *testing.T, fn func()) {
	// Use the new helper functions instead of the old testMutex approach
	state := newTestHTTPState()
	defer state.restore()

	fn()
}

// Bridge exported functions so previous unqualified references still work.
var GetLatestGitHubRelease = utils.GetLatestGitHubRelease

func TestGetLatestGitHubRelease(t *testing.T) {
	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer srv.Close()

	ctx := context.Background()
	ver, err := GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", ver)
}

func TestGetLatestGitHubReleaseError(t *testing.T) {
	// Server returns non-200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()
	ver, err := GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	require.Error(t, err)
	assert.Empty(t, ver)
}

type mockStatusTransport struct{ status int }

func (m mockStatusTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	switch m.status {
	case http.StatusOK:
		body, _ := json.Marshal(map[string]string{"tag_name": "v1.2.3"})
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: m.status, Body: io.NopCloser(bytes.NewReader([]byte("err"))), Header: make(http.Header)}, nil
	}
}

func TestGetLatestGitHubReleaseVarious(t *testing.T) {
	ctx := context.Background()

	t.Run("Forbidden", func(t *testing.T) {
		restoreTransport := saveAndRestoreHTTPDefaultTransport(t, mockStatusTransport{status: http.StatusForbidden})
		defer restoreTransport()

		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err == nil {
			t.Fatalf("expected forbidden error")
		}
	})

	t.Run("UnexpectedStatus", func(t *testing.T) {
		restoreTransport := saveAndRestoreHTTPDefaultTransport(t, mockStatusTransport{status: http.StatusInternalServerError})
		defer restoreTransport()

		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err == nil {
			t.Fatalf("expected error for 500 status")
		}
	})

	// Ensure function respects GITHUB_TOKEN header set
	t.Run("WithToken", func(t *testing.T) {
		restoreTransport := saveAndRestoreHTTPDefaultTransport(t, mockStatusTransport{status: http.StatusOK})
		defer restoreTransport()

		t.Setenv("GITHUB_TOKEN", "dummy")
		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err != nil {
			t.Fatalf("unexpected err with token: %v", err)
		}
	})
}

func TestGetLatestGitHubRelease_AuthErrors(t *testing.T) {
	cases := []struct {
		status int
	}{
		{http.StatusUnauthorized},
		{http.StatusForbidden},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
		}))
		_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", srv.URL)
		if err == nil {
			t.Errorf("expected error for status %d", c.status)
		}
		srv.Close()
	}
}

type errBody struct{ first bool }

func (e *errBody) Read(p []byte) (int, error) {
	if e.first {
		copy(p, []byte("{")) // send partial
		e.first = false
		return 1, nil
	}
	return 0, io.ErrUnexpectedEOF
}
func (e *errBody) Close() error { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetLatestGitHubReleaseReadError(t *testing.T) {
	restoreClient := saveAndRestoreHTTPDefaultClient(t, &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       &errBody{first: true},
			Header:     make(http.Header),
		}
		return resp, nil
	})})
	defer restoreClient()

	_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
	if err == nil {
		t.Fatalf("expected error due to body read failure, got nil")
	}
}

func TestGetLatestGitHubReleaseUnauthorizedExtra(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", server.URL)
	if err == nil {
		t.Fatalf("expected error for unauthorized response, got nil")
	}
}

type staticResponseRoundTripper struct{}

func (staticResponseRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Expect default base URL
	if !strings.Contains(r.URL.Host, "api.github.com") {
		return nil, http.ErrUseLastResponse
	}
	body := io.NopCloser(strings.NewReader(`{"tag_name":"v0.0.1"}`))
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Header:     make(http.Header),
	}, nil
}

func TestGetLatestGitHubRelease_DefaultBaseURL(t *testing.T) {
	restoreClient := saveAndRestoreHTTPDefaultClient(t, &http.Client{Transport: staticResponseRoundTripper{}})
	defer restoreClient()

	ver, err := GetLatestGitHubRelease(context.Background(), "kdeps/schema", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "0.0.1" {
		t.Fatalf("unexpected version: %s", ver)
	}
}

type ghRoundTrip func(*http.Request) (*http.Response, error)

func (f ghRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockResp(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(body))}
}

func TestGetLatestGitHubReleaseExtra(t *testing.T) {
	ctx := context.Background()

	// Success path
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
		resp := map[string]string{"tag_name": "v1.2.3"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	v, err := utils.GetLatestGitHubRelease(ctx, "owner/repo", ts.URL)
	require.NoError(t, err)
	require.Equal(t, "1.2.3", v)

	// Unauthorized path
	ts401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts401.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts401.URL)
	require.Error(t, err)

	// Non-OK generic error path
	ts500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts500.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts500.URL)
	require.Error(t, err)

	// Forbidden path (rate limit)
	ts403 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts403.Close()
	_, err = utils.GetLatestGitHubRelease(ctx, "owner/repo", ts403.URL)
	require.Error(t, err)

	// Malformed JSON path – should error on JSON parse
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		assert.Equal(t, "Bearer "+token, auth)

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
func TestGetLatestGitHubRelease_Success_Dup(t *testing.T) {
	withTestState(t, func() {
		payload := `{"tag_name":"v1.2.3"}`
		http.DefaultClient.Transport = ghRoundTrip(func(_ *http.Request) (*http.Response, error) {
			return mockResp(http.StatusOK, payload), nil
		})

		ver, err := utils.GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ver != "1.2.3" {
			t.Fatalf("expected 1.2.3, got %s", ver)
		}

		// _ = schema.Version(context.Background()) // This line was removed as per the edit hint
	})
}

// TestGetLatestGitHubRelease_Errors checks status-code error branches.
func TestGetLatestGitHubRelease_Errors_Dup(t *testing.T) {
	withTestState(t, func() {
		cases := []struct {
			status int
			expect string
		}{
			{http.StatusUnauthorized, "unauthorized"},
			{http.StatusForbidden, "rate limit"},
			{http.StatusNotFound, "unexpected status code"},
		}

		for _, c := range cases {
			http.DefaultClient.Transport = ghRoundTrip(func(_ *http.Request) (*http.Response, error) {
				return mockResp(c.status, "{}"), nil
			})
			_, err := utils.GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
			if err == nil || !contains(err.Error(), c.expect) {
				t.Fatalf("status %d expected error containing %q, got %v", c.status, c.expect, err)
			}
		}

		// _ = schema.Version(context.Background()) // This line was removed as per the edit hint
	})
}

func contains(s, substr string) bool { return bytes.Contains([]byte(s), []byte(substr)) }

func TestGetLatestGitHubRelease_MockServer2(t *testing.T) {
	// Successful path
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := struct {
			Tag string `json:"tag_name"`
		}{Tag: "v1.2.3"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ctx := context.Background()
	ver, err := utils.GetLatestGitHubRelease(ctx, "org/repo", ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.3" {
		t.Fatalf("expected 1.2.3 got %s", ver)
	}

	// Unauthorized path
	u401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer u401.Close()
	if _, err := utils.GetLatestGitHubRelease(ctx, "org/repo", u401.URL); err == nil {
		t.Fatalf("expected unauthorized error")
	}

	// Non-200 path
	u500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer u500.Close()
	if _, err := utils.GetLatestGitHubRelease(ctx, "org/repo", u500.URL); err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestGetLatestGitHubRelease_Success_Alt(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]string{"tag_name": "v2.3.4"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ver, err := GetLatestGitHubRelease(context.Background(), "dummy/repo", ts.URL)
	require.NoError(t, err)
	assert.Equal(t, "2.3.4", ver)
}

func TestGetLatestGitHubRelease_Errors_Alt(t *testing.T) {
	tests := []struct {
		status  int
		wantErr string
	}{
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "rate limit"},
		{http.StatusInternalServerError, "unexpected status code"},
	}
	for _, tc := range tests {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
		}))
		ver, err := GetLatestGitHubRelease(context.Background(), "dummy/repo", ts.URL)
		require.Error(t, err)
		assert.Empty(t, ver)
		assert.Contains(t, err.Error(), tc.wantErr)
		ts.Close()
	}
}
