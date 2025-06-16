package docker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
)

type stubRoundTrip func(*http.Request) (*http.Response, error)

func (f stubRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGenerateURLs_UseLatestWithStubsLow(t *testing.T) {
	// Stub GitHub release fetcher to avoid network
	origFetcher := utils.GitHubReleaseFetcher
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "99.99.99", nil
	}
	defer func() { utils.GitHubReleaseFetcher = origFetcher }()

	// Intercept HTTP requests for both Anaconda archive and GitHub API
	origTransport := http.DefaultTransport
	http.DefaultTransport = stubRoundTrip(func(req *http.Request) (*http.Response, error) {
		var body string
		if strings.Contains(req.URL.Host, "repo.anaconda.com") {
			body = `Anaconda3-2024.10-1-Linux-x86_64.sh Anaconda3-2024.10-1-Linux-aarch64.sh`
		} else {
			body = `{"tag_name":"v99.99.99"}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})
	defer func() { http.DefaultTransport = origTransport }()

	schema.UseLatest = true
	defer func() { schema.UseLatest = false }()

	items, err := GenerateURLs(context.Background())
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected non-empty items")
	}
	for _, it := range items {
		if !strings.Contains(it.LocalName, "latest") {
			t.Fatalf("expected LocalName to contain latest, got %s", it.LocalName)
		}
	}
}
