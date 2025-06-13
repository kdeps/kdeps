package utils

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

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
