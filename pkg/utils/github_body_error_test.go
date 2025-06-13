package utils_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

type errBody struct{ first bool }

func (e *errBody) Read(p []byte) (int, error) {
	if e.first {
		copy(p, []byte("{")) // send partial
		e.first = false
		return 1, nil
	}
	return 0, io.ErrUnexpectedEOF
}
func (e *errBody) Close() error { return nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetLatestGitHubReleaseReadError(t *testing.T) {
	// Replace default client temporarily.
	prevClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       &errBody{first: true},
			Header:     make(http.Header),
		}
		return resp, nil
	})}
	defer func() { http.DefaultClient = prevClient }()

	_, err := utils.GetLatestGitHubRelease(context.Background(), "owner/repo", "https://api.github.com")
	if err == nil {
		t.Fatalf("expected error due to body read failure, got nil")
	}
}
