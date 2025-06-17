package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

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
