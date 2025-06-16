package tool

import (
	"os"
	"path/filepath"
	"testing"
)

// TestInitializeDatabaseFailure exercises the retry + failure branch by pointing
// the DB path into a directory that is not writable, ensuring all attempts fail.
func TestInitializeDatabaseFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kdeps_ro")
	if err != nil {
		t.Fatalf("tempdir error: %v", err)
	}
	// make directory read-only so sqlite cannot create file inside it
	if err := os.Chmod(tmpDir, 0o555); err != nil {
		t.Fatalf("chmod error: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tool.db")
	_, err = InitializeDatabase(dbPath)
	if err == nil {
		t.Fatalf("expected error when initializing DB in read-only dir")
	}
}
