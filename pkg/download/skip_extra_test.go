package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestDownloadFile_SkipWhenExists verifies that DownloadFile returns nil and does not re-download
// when the target file already exists and useLatest is false.
func TestDownloadFile_SkipWhenExists(t *testing.T) {
	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "data.txt")

	// Pre-create non-empty file to trigger skip logic.
	if err := afero.WriteFile(fs, dest, []byte("cached"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// URL is irrelevant because we expect early return.
	err := DownloadFile(fs, context.Background(), "http://example.com/irrelevant", dest, logging.NewTestLogger(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestDownloadFile_InvalidStatus exercises the non-200 status code branch.
func TestDownloadFile_InvalidStatus(t *testing.T) {
	fs := afero.NewOsFs()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "out.txt")

	// Spin up a server that returns 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	err := DownloadFile(fs, context.Background(), srv.URL, dest, logging.NewTestLogger(), true)
	if err == nil {
		t.Fatalf("expected error on 500 status")
	}
}
