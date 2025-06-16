package tool

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestInitializeDatabaseAndHistory(t *testing.T) {
	// Create temp dir and DB path using afero.NewOsFs
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "tooldb")
	if err != nil {
		t.Fatalf("TempDir error: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "kdeps_tool.db")

	// Initialize the reader (this implicitly calls InitializeDatabase).
	reader, err := InitializeTool(dbPath)
	if err != nil {
		t.Fatalf("InitializeTool error: %v", err)
	}
	defer reader.DB.Close()

	// Manually insert a couple of history rows.
	now := time.Now().Unix()
	_, err = reader.DB.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)", "someid", "hello-1", now)
	if err != nil {
		t.Fatalf("insert history err: %v", err)
	}
	_, err = reader.DB.Exec("INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)", "someid", "hello-2", now+1)
	if err != nil {
		t.Fatalf("insert history err: %v", err)
	}

	// Request history via the reader.Read API.
	uri := url.URL{Scheme: "tool", Path: "/someid", RawQuery: "op=history"}
	out, err := reader.Read(uri)
	if err != nil {
		t.Fatalf("Read history error: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, "hello-1") || !strings.Contains(got, "hello-2") {
		t.Fatalf("unexpected history output: %s", got)
	}
}
