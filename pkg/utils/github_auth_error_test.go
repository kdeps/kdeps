package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestGitHubRelease_AuthErrors(t *testing.T) {
	cases := []struct {
		status int
	}{
		{http.StatusUnauthorized},
		{http.StatusForbidden},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(c.status)
		}))
		_, err := GetLatestGitHubRelease(context.Background(), "owner/repo", srv.URL)
		if err == nil {
			t.Errorf("expected error for status %d", c.status)
		}
		srv.Close()
	}
}
