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
