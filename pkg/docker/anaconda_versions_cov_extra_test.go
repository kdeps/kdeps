package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

// mockRoundTripper implements http.RoundTripper to stub external calls made by
// GetLatestAnacondaVersions. It always returns a fixed HTML listing that
// contains multiple Anaconda installer filenames so that the version parsing
// logic is fully exercised.

type mockRoundTripper struct{}

func (m mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Minimal HTML directory index with two entries for different archs.
	body := `
<html><body>
<a href="Anaconda3-2024.05-0-Linux-x86_64.sh">Anaconda3-2024.05-0-Linux-x86_64.sh</a><br>
<a href="Anaconda3-2024.10-1-Linux-aarch64.sh">Anaconda3-2024.10-1-Linux-aarch64.sh</a><br>
</body></html>`

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestGetLatestAnacondaVersionsMocked(t *testing.T) {
	// Swap the default transport for our mock and restore afterwards.
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockRoundTripper{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// We expect to get both architectures with their respective versions.
	if versions["x86_64"] != "2024.05-0" {
		t.Fatalf("expected x86_64 version '2024.05-0', got %s", versions["x86_64"])
	}
	if versions["aarch64"] != "2024.10-1" {
		t.Fatalf("expected aarch64 version '2024.10-1', got %s", versions["aarch64"])
	}
}

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

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetLatestAnacondaVersionsMock(t *testing.T) {
	ctx := context.Background()

	// HTML snippet with two architectures
	html := `<!DOCTYPE html><html><body>
    <a href="Anaconda3-2024.10-1-Linux-x86_64.sh">x</a>
    <a href="Anaconda3-2024.05-1-Linux-aarch64.sh">y</a>
    </body></html>`

	// Save original transport and replace
	orig := http.DefaultTransport
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       ioutil.NopCloser(bytes.NewBufferString(html)),
			}, nil
		}
		return orig.RoundTrip(r)
	})
	defer func() { http.DefaultTransport = orig }()

	versions, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if versions["x86_64"] == "" || versions["aarch64"] == "" {
		t.Fatalf("expected versions for both architectures: %+v", versions)
	}
}
