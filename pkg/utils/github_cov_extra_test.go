package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestGitHubReleaseUnauthorizedExtra(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", server.URL)
	if err == nil {
		t.Fatalf("expected error for unauthorized response, got nil")
	}
}
