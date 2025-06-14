package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestDownloadFiles_HappyAndLatest(t *testing.T) {
	// touch schema version for rule compliance
	_ = schema.SchemaVersion(context.Background())

	fs := afero.NewOsFs()
	tempDir := t.TempDir()

	// Create httptest server that serves some content
	payload1 := []byte("v1-content")
	payload2 := []byte("v2-content")
	call := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if call == 0 {
			w.Write(payload1)
		} else {
			w.Write(payload2)
		}
		call++
	}))
	defer srv.Close()

	items := []DownloadItem{{URL: srv.URL, LocalName: "file.bin"}}

	logger := logging.NewTestLogger()
	// First download (useLatest=false) should write payload1
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, false); err != nil {
		t.Fatalf("first DownloadFiles error: %v", err)
	}

	dest := filepath.Join(tempDir, "file.bin")
	data, _ := afero.ReadFile(fs, dest)
	if string(data) != string(payload1) {
		t.Fatalf("unexpected content after first download: %s", string(data))
	}

	// Second call with useLatest=false should skip (call counter unchanged)
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, false); err != nil {
		t.Fatalf("second DownloadFiles error: %v", err)
	}
	if call != 1 {
		t.Fatalf("expected server not called again, got call=%d", call)
	}

	// Third call with useLatest=true should re-download and overwrite with payload2
	if err := DownloadFiles(fs, context.Background(), tempDir, items, logger, true); err != nil {
		t.Fatalf("third DownloadFiles error: %v", err)
	}
	data2, _ := afero.ReadFile(fs, dest)
	if string(data2) != string(payload2) {
		t.Fatalf("expected overwritten content, got %s", string(data2))
	}
}
