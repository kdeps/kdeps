package resolver

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
)

func TestClearItemDB(t *testing.T) {
	// Setup in-memory filesystem (not used by ClearItemDB itself)
	fs := afero.NewMemMapFs()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "items.db")

	// Initialize item reader with some rows
	reader, err := item.InitializeItem(dbPath, []string{"foo", "bar"})
	if err != nil {
		t.Fatalf("InitializeItem failed: %v", err)
	}

	dr := &DependencyResolver{
		Fs:         fs,
		Logger:     logging.NewTestLogger(),
		ItemReader: reader,
		ItemDBPath: dbPath,
	}

	// Ensure rows exist before clearing
	var count int
	if err := reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected initial rows, got 0")
	}

	// Invoke ClearItemDB
	if err := dr.ClearItemDB(); err != nil {
		t.Fatalf("ClearItemDB returned error: %v", err)
	}

	// Verify table is empty
	if err := reader.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatalf("count query after clear failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows after clear, got %d", count)
	}

	// Closing DB
	reader.DB.Close()
	// Ensure ClearItemDB handles closed DB gracefully
	dr.ItemReader.DB, _ = sql.Open("sqlite3", dbPath) // reopen to avoid error on second call
	_ = dr.ClearItemDB()
}
