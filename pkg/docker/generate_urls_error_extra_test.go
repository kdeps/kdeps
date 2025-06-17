package docker

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
)

// rtFunc already declared in another test file; reuse that type here without redefining.

func TestGenerateURLs_GitHubError(t *testing.T) {
	ctx := context.Background()

	// Save globals and transport.
	origLatest := schema.UseLatest
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		http.DefaultTransport = origTransport
	}()

	schema.UseLatest = true

	// Force GitHub API request to return HTTP 403.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return &http.Response{
				StatusCode: 403,
				Body:       ioutil.NopCloser(bytes.NewBufferString("forbidden")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx); err == nil {
		t.Fatalf("expected error when GitHub API returns forbidden")
	}
}

func TestGenerateURLs_AnacondaError(t *testing.T) {
	ctx := context.Background()

	// Save and restore globals and transport.
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		http.DefaultTransport = origTransport
	}()

	// GitHub fetch succeeds to move past first item.
	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, base string) (string, error) {
		return "0.28.1", nil
	}

	// Make Anaconda request return HTTP 500.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(bytes.NewBufferString("server error")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx); err == nil {
		t.Fatalf("expected error when Anaconda version fetch fails")
	}
}
