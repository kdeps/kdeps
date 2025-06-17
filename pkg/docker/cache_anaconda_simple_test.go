package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

// mockTransport intercepts HTTP requests to repo.anaconda.com and returns fixed HTML.
type mockHTMLTransport struct{}

func (m mockHTMLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "repo.anaconda.com" {
		html := `<html><body>
<a href="Anaconda3-2024.10-1-Linux-x86_64.sh">Anaconda3-2024.10-1-Linux-x86_64.sh</a>
<a href="Anaconda3-2024.09-1-Linux-aarch64.sh">Anaconda3-2024.09-1-Linux-aarch64.sh</a>
</body></html>`
		resp := &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(html)),
			Header:     make(http.Header),
		}
		return resp, nil
	}
	return nil, http.ErrUseLastResponse
}

func TestGetLatestAnacondaVersionsMockSimple(t *testing.T) {
	// Replace the default transport
	origTransport := http.DefaultTransport
	http.DefaultTransport = mockHTMLTransport{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	vers, err := GetLatestAnacondaVersions(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vers["x86_64"] != "2024.10-1" {
		t.Fatalf("x86_64 version mismatch, got %s", vers["x86_64"])
	}
	if vers["aarch64"] != "2024.09-1" {
		t.Fatalf("aarch64 version mismatch, got %s", vers["aarch64"])
	}
}
