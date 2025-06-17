package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

type roundTripperLatest struct{}

func (roundTripperLatest) RoundTrip(req *http.Request) (*http.Response, error) {
	// Distinguish responses based on requested URL path.
	switch {
	case req.URL.Host == "api.github.com":
		// Fake GitHub release JSON.
		body, _ := json.Marshal(map[string]string{"tag_name": "v0.29.0"})
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	case req.URL.Host == "repo.anaconda.com":
		html := `<a href="Anaconda3-2024.05-0-Linux-x86_64.sh">file</a><a href="Anaconda3-2024.05-0-Linux-aarch64.sh">file</a>`
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(html))), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(""))), Header: make(http.Header)}, nil
	}
}

type nopCloser struct{ *bytes.Reader }

func (n nopCloser) Close() error { return nil }

func ioNopCloser(r *bytes.Reader) io.ReadCloser { return nopCloser{r} }

func TestGenerateURLsUseLatest(t *testing.T) {
	// Mock HTTP.
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperLatest{}
	defer func() { http.DefaultTransport = origTransport }()

	// Enable latest mode.
	origLatest := schema.UseLatest
	schema.UseLatest = true
	defer func() { schema.UseLatest = origLatest }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, itm := range items {
		if itm.LocalName == "" || itm.URL == "" {
			t.Fatalf("GenerateURLs produced empty fields: %+v", itm)
		}
		if !schema.UseLatest {
			t.Fatalf("schema.UseLatest should still be true inside loop")
		}
	}
}
