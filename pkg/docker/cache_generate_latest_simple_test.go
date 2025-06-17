package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

type multiMockTransport struct{}

func (m multiMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Host {
	case "api.github.com":
		body, _ := json.Marshal(map[string]string{"tag_name": "v9.9.9"})
		return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	case "repo.anaconda.com":
		html := `<a href="Anaconda3-2025.01-0-Linux-x86_64.sh">Anaconda3-2025.01-0-Linux-x86_64.sh</a>`
		return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(bytes.NewBufferString(html)), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: 404, Body: ioutil.NopCloser(bytes.NewBuffer(nil)), Header: make(http.Header)}, nil
	}
}

func TestGenerateURLsLatestMode(t *testing.T) {
	// Enable latest mode
	schema.UseLatest = true
	defer func() { schema.UseLatest = false }()

	origTransport := http.DefaultTransport
	http.DefaultTransport = multiMockTransport{}
	defer func() { http.DefaultTransport = origTransport }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs latest failed: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected items when latest mode enabled")
	}
	// All LocalName fields should contain "latest" placeholder
	for _, it := range items {
		if it.LocalName == "" {
			t.Fatalf("missing LocalName")
		}
		if !contains(it.LocalName, "latest") {
			t.Fatalf("LocalName should reference latest: %s", it.LocalName)
		}
	}
}

func contains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }
