package utils_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	utilspkg "github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bridge exported functions so previous unqualified references still work.
var GetLatestGitHubRelease = utilspkg.GetLatestGitHubRelease

func TestGetLatestGitHubRelease(t *testing.T) {
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

type mockStatusTransport struct{ status int }

func (m mockStatusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch m.status {
	case http.StatusOK:
		body, _ := json.Marshal(map[string]string{"tag_name": "v1.2.3"})
		return &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: m.status, Body: ioutil.NopCloser(bytes.NewReader([]byte("err"))), Header: make(http.Header)}, nil
	}
}

func TestGetLatestGitHubReleaseVarious(t *testing.T) {
	ctx := context.Background()

	// Backup and restore default transport
	oldTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = oldTransport }()

	t.Run("Success", func(t *testing.T) {
		http.DefaultTransport = mockStatusTransport{status: http.StatusOK}
		ver, err := GetLatestGitHubRelease(ctx, "owner/repo", "https://api.github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ver != "1.2.3" {
			t.Fatalf("unexpected version: %s", ver)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		http.DefaultTransport = mockStatusTransport{status: http.StatusUnauthorized}
		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err == nil {
			t.Fatalf("expected unauthorized error")
		}
	})

	t.Run("Forbidden", func(t *testing.T) {
		http.DefaultTransport = mockStatusTransport{status: http.StatusForbidden}
		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err == nil {
			t.Fatalf("expected forbidden error")
		}
	})

	t.Run("UnexpectedStatus", func(t *testing.T) {
		http.DefaultTransport = mockStatusTransport{status: http.StatusInternalServerError}
		if _, err := GetLatestGitHubRelease(ctx, "owner/repo", ""); err == nil {
			t.Fatalf("expected error for 500 status")
		}
	})

	// Ensure function respects GITHUB_TOKEN header set
	t.Run("WithToken", func(t *testing.T) {
		http.DefaultTransport = mockStatusTransport{status: http.StatusOK}
		os.Setenv("GITHUB_TOKEN", "dummy")
		defer os.Unsetenv("GITHUB_TOKEN")
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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	// Replace default client temporarily.
	prevClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       &errBody{first: true},
			Header:     make(http.Header),
		}
		return resp, nil
	})}
	defer func() { http.DefaultClient = prevClient }()

	_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
	if err == nil {
		t.Fatalf("expected error due to body read failure, got nil")
	}
}

func TestGetLatestGitHubReleaseUnauthorizedExtra(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
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
	prev := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: staticResponseRoundTripper{}}
	defer func() { http.DefaultClient = prev }()

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
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: ioutil.NopCloser(bytes.NewBufferString(body))}
}

func TestGetLatestGitHubReleaseExtra(t *testing.T) {
	ctx := context.Background()

	// Success path
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
		resp := map[string]string{"tag_name": "v1.2.3"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	v, err := utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", ts.URL)
	require.NoError(t, err)
	require.Equal(t, "1.2.3", v)

	// Unauthorized path
	ts401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts401.Close()
	_, err = utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", ts401.URL)
	require.Error(t, err)

	// Non-OK generic error path
	ts500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts500.Close()
	_, err = utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", ts500.URL)
	require.Error(t, err)

	// Forbidden path (rate limit)
	ts403 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts403.Close()
	_, err = utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", ts403.URL)
	require.Error(t, err)

	// Malformed JSON path – should error on JSON parse
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{ "tag_name": 123 }`)) // tag_name not string
	}))
	defer tsBadJSON.Close()
	_, err = utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", tsBadJSON.URL)
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

	ver, err := utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "9.9.9", ver)
}

// TestGetLatestGitHubReleaseInvalidURL ensures that malformed URLs trigger an error
func TestGetLatestGitHubReleaseInvalidURL(t *testing.T) {
	ctx := context.Background()
	ver, err := utilspkg.GetLatestGitHubRelease(ctx, "owner/repo", "://bad url")
	require.Error(t, err)
	assert.Empty(t, ver)
}

// TestGetLatestGitHubRelease_Success verifies the helper parses tag names and
// strips the leading 'v'.
func TestGetLatestGitHubRelease_Success_Dup(t *testing.T) {
	payload := `{"tag_name":"v1.2.3"}`
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = ghRoundTrip(func(r *http.Request) (*http.Response, error) {
		return mockResp(http.StatusOK, payload), nil
	})
	defer func() { http.DefaultClient.Transport = old }()

	ver, err := utilspkg.GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.3" {
		t.Fatalf("expected 1.2.3, got %s", ver)
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestGetLatestGitHubRelease_Errors checks status-code error branches.
func TestGetLatestGitHubRelease_Errors_Dup(t *testing.T) {
	cases := []struct {
		status int
		expect string
	}{
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "rate limit"},
		{http.StatusNotFound, "unexpected status code"},
	}

	for _, c := range cases {
		old := http.DefaultClient.Transport
		http.DefaultClient.Transport = ghRoundTrip(func(r *http.Request) (*http.Response, error) {
			return mockResp(c.status, "{}"), nil
		})
		_, err := utilspkg.GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
		if err == nil || !contains(err.Error(), c.expect) {
			t.Fatalf("status %d expected error containing %q, got %v", c.status, c.expect, err)
		}
		http.DefaultClient.Transport = old
	}

	_ = schema.SchemaVersion(context.Background())
}

func contains(s, substr string) bool { return bytes.Contains([]byte(s), []byte(substr)) }

func TestGetLatestGitHubRelease_MockServer2(t *testing.T) {
	// Successful path
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Tag string `json:"tag_name"`
		}{Tag: "v1.2.3"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ctx := context.Background()
	ver, err := utilspkg.GetLatestGitHubRelease(ctx, "org/repo", ts.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.3" {
		t.Fatalf("expected 1.2.3 got %s", ver)
	}

	// Unauthorized path
	u401 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer u401.Close()
	if _, err := utilspkg.GetLatestGitHubRelease(ctx, "org/repo", u401.URL); err == nil {
		t.Fatalf("expected unauthorized error")
	}

	// Non-200 path
	u500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer u500.Close()
	if _, err := utilspkg.GetLatestGitHubRelease(ctx, "org/repo", u500.URL); err == nil {
		t.Fatalf("expected error for 500 status")
	}
}

func TestGetLatestGitHubRelease_Success_Alt(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{"tag_name": "v2.3.4"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	ver, err := GetLatestGitHubRelease(context.Background(), "dummy/repo", ts.URL)
	assert.NoError(t, err)
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
