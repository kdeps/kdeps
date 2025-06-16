package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

// TestGetLatestAnacondaVersions_StatusError ensures non-200 response returns error.
func TestGetLatestAnacondaVersions_StatusError(t *testing.T) {
	ctx := context.Background()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Header: make(http.Header), Body: ioutil.NopCloser(bytes.NewBufferString(""))}, nil
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected error for non-OK status")
	}
}

// TestGetLatestAnacondaVersions_NoMatches ensures HTML without matches returns error.
func TestGetLatestAnacondaVersions_NoMatches(t *testing.T) {
	ctx := context.Background()
	html := "<html><body>no versions here</body></html>"
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: ioutil.NopCloser(bytes.NewBufferString(html))}, nil
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected error when no versions found")
	}
}

// TestGetLatestAnacondaVersions_NetworkError simulates transport failure.
func TestGetLatestAnacondaVersions_NetworkError(t *testing.T) {
	ctx := context.Background()
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})
	defer func() { http.DefaultTransport = original }()

	if _, err := GetLatestAnacondaVersions(ctx); err == nil {
		t.Fatalf("expected network error")
	}
}

// TestBuildURLPlaceholders verifies placeholder interpolation.
func TestBuildURLPlaceholders(t *testing.T) {
	base := "https://repo/{version}/file-{arch}.sh"
	got := buildURL(base, "v2.0", "x86_64")
	want := "https://repo/v2.0/file-x86_64.sh"
	if got != want {
		t.Fatalf("buildURL returned %s, want %s", got, want)
	}
}
