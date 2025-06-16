package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
)

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
