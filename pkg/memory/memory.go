package memory

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/apple/pkl-go/pkl"
	_ "github.com/mattn/go-sqlite3"
)

// PklResourceReader implements the pkl.ResourceReader interface for SQLite.
type PklResourceReader struct {
	db *sql.DB
}

// Scheme returns the URI scheme for this reader.
func (r PklResourceReader) Scheme() string {
	return "memory"
}

// IsGlobbable indicates whether the reader supports globbing (not needed here).
func (r PklResourceReader) IsGlobbable() bool {
	return false
}

// HasHierarchicalUris indicates whether URIs are hierarchical (not needed here).
func (r PklResourceReader) HasHierarchicalUris() bool {
	return false
}

// ListElements is not used in this implementation.
func (r PklResourceReader) ListElements(_ url.URL) ([]pkl.PathElement, error) {
	return nil, nil
}

// Read retrieves or updates an item in the SQLite database based on the URI.
func (r PklResourceReader) Read(uri url.URL) ([]byte, error) {
	id := strings.TrimPrefix(uri.Path, "/")
	if id == "" {
		return nil, errors.New("invalid URI: no item ID provided")
	}

	query := uri.Query()
	operation := query.Get("op")
	if operation == "update" {
		newValue := query.Get("value")
		if newValue == "" {
			return nil, errors.New("update operation requires a value parameter")
		}

		result, err := r.db.Exec(
			"INSERT OR REPLACE INTO items (id, value) VALUES (?, ?)",
			id, newValue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to update item: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("failed to check update result: %w", err)
		}
		if rowsAffected == 0 {
			return nil, fmt.Errorf("no item updated for ID %s", id)
		}

		return []byte("Updated item " + id), nil
	}

	var value string
	err := r.db.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item with ID %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read item: %w", err)
	}

	return []byte(value), nil
}

// InitializeDatabase sets up the SQLite database and creates the items table.
func InitializeDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create items table if it doesn't exist.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	return db, nil
}

func InitializeMemory(dbPath string) (PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return PklResourceReader{}, fmt.Errorf("error initializing database: %w", err)
	}
	defer db.Close()

	return PklResourceReader{db: db}, nil
}
