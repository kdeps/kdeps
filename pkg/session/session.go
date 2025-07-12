package session

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	_ "github.com/mattn/go-sqlite3"
)

// PklResourceReader implements the pkl.ResourceReader interface for SQLite.
type PklResourceReader struct {
	DB     *sql.DB
	DBPath string // Store dbPath for reinitialization
}

// Scheme returns the URI scheme for this reader.
func (r *PklResourceReader) Scheme() string {
	return "session"
}

// IsGlobbable indicates whether the reader supports globbing (not needed here).
func (r *PklResourceReader) IsGlobbable() bool {
	return false
}

// HasHierarchicalUris indicates whether URIs are hierarchical (not needed here).
func (r *PklResourceReader) HasHierarchicalUris() bool {
	return false
}

// ListElements is not used in this implementation.
func (r *PklResourceReader) ListElements(_ url.URL) ([]pkl.PathElement, error) {
	return nil, nil
}

// Read retrieves, sets, deletes, or clears records in the SQLite database based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Check if receiver is nil and initialize with fixed DBPath
	if r == nil {
		newReader, err := InitializeSession(r.DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
		}
		r = newReader
	}

	// Check if db is nil and initialize with retries
	if r.DB == nil {
		maxAttempts := 5
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			db, err := InitializeDatabase(r.DBPath)
			if err == nil {
				r.DB = db
				break
			}
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to initialize database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
		}
	}

	id := strings.TrimPrefix(uri.Path, "/")
	query := uri.Query()
	operation := query.Get("op")

	switch operation {
	case "set":
		return r.setRecord(id, query)
	case "delete":
		return r.deleteRecord(id)
	case "clear":
		return r.clearRecords(id)
	default: // getRecord (no operation specified)
		return r.getRecord(id)
	}
}

// InitializeDatabase sets up the SQLite database and creates the records table with retries.
func InitializeDatabase(dbPath string) (*sql.DB, error) {
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Verify connection
		if err := db.Ping(); err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Create records table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS records (
				id TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create records table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeSession creates a new PklResourceReader with an initialized SQLite database.
func InitializeSession(dbPath string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}

// setRecord stores a session record in the database
func (r *PklResourceReader) setRecord(id string, query url.Values) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no record ID provided for set operation")
	}
	newValue := query.Get("value")
	if newValue == "" {
		return nil, errors.New("set operation requires a value parameter")
	}

	result, err := r.DB.Exec(
		"INSERT OR REPLACE INTO records (id, value) VALUES (?, ?)",
		id, newValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check set result: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no record set for ID %s", id)
	}

	return []byte(newValue), nil
}

// deleteRecord removes a session record from the database
func (r *PklResourceReader) deleteRecord(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no record ID provided for delete operation")
	}

	result, err := r.DB.Exec("DELETE FROM records WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check delete result: %w", err)
	}

	return []byte(fmt.Sprintf("Deleted %d record(s)", rowsAffected)), nil
}

// clearRecords removes all session records from the database
func (r *PklResourceReader) clearRecords(id string) ([]byte, error) {
	if id != "_" {
		return nil, errors.New("clear operation requires path '/_'")
	}

	result, err := r.DB.Exec("DELETE FROM records")
	if err != nil {
		return nil, fmt.Errorf("failed to clear records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check clear result: %w", err)
	}

	return []byte(fmt.Sprintf("Cleared %d records", rowsAffected)), nil
}

// getRecord retrieves a session record from the database
func (r *PklResourceReader) getRecord(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no record ID provided")
	}

	var value string
	err := r.DB.QueryRow("SELECT value FROM records WHERE id = ?", id).Scan(&value)
	if err == sql.ErrNoRows {
		return []byte(""), nil // Return empty string for not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read record: %w", err)
	}

	return []byte(value), nil
}
