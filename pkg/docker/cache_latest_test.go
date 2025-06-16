package docker

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// mockTransport intercepts HTTP requests and serves canned responses.
type mockTransport struct{}

func (m mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	if strings.Contains(req.URL.Path, "/releases/latest") { // GitHub API
		body = `{"tag_name":"v1.2.3"}`
	} else { // Anaconda archive listing
		body = `Anaconda3-2024.05-0-Linux-x86_64.sh
Anaconda3-2024.05-0-Linux-aarch64.sh`
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	return resp, nil
}

func TestGenerateURLs_UseLatest(t *testing.T) {
	// Save and restore globals we mutate.
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		http.DefaultTransport = origTransport
	}()

	schema.UseLatest = true

	// Stub GitHub release fetcher.
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, baseURL string) (string, error) {
		return "v9.9.9", nil
	}

	// Intercept Anaconda archive request.
	http.DefaultTransport = mockTransport{}

	items, err := GenerateURLs(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, items)

	// Ensure an item for pkl latest and anaconda latest exist.
	var gotPkl, gotAnaconda bool
	for _, it := range items {
		if strings.Contains(it.LocalName, "pkl-linux-latest") {
			gotPkl = true
		}
		if strings.Contains(it.LocalName, "anaconda-linux-latest") {
			gotAnaconda = true
		}
	}
	assert.True(t, gotPkl, "expected pkl latest item")
	assert.True(t, gotAnaconda, "expected anaconda latest item")
}
