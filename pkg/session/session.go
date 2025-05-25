package session

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
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
		log.Printf("Warning: PklResourceReader is nil for URI: %s, initializing with DBPath", uri.String())
		newReader, err := InitializeSession(r.DBPath)
		if err != nil {
			log.Printf("Failed to initialize PklResourceReader in Read: %v", err)
			return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
		}
		r = newReader
		log.Printf("Initialized PklResourceReader with DBPath")
	}

	// Check if db is nil and initialize with retries
	if r.DB == nil {
		log.Printf("Database connection is nil, attempting to initialize with path: %s", r.DBPath)
		maxAttempts := 5
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			db, err := InitializeDatabase(r.DBPath)
			if err == nil {
				r.DB = db
				log.Printf("Database initialized successfully in Read on attempt %d", attempt)
				break
			}
			log.Printf("Attempt %d: Failed to initialize database in Read: %v", attempt, err)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to initialize database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
		}
	}

	id := strings.TrimPrefix(uri.Path, "/")
	query := uri.Query()
	operation := query.Get("op")

	log.Printf("Read called with URI: %s, operation: %s", uri.String(), operation)

	switch operation {
	case "set":
		if id == "" {
			log.Printf("setRecord failed: no record ID provided")
			return nil, errors.New("invalid URI: no record ID provided for set operation")
		}
		newValue := query.Get("value")
		if newValue == "" {
			log.Printf("setRecord failed: no value provided")
			return nil, errors.New("set operation requires a value parameter")
		}

		log.Printf("setRecord processing id: %s, value: %s", id, newValue)

		result, err := r.DB.Exec(
			"INSERT OR REPLACE INTO records (id, value) VALUES (?, ?)",
			id, newValue,
		)
		if err != nil {
			log.Printf("setRecord failed to execute SQL: %v", err)
			return nil, fmt.Errorf("failed to set record: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("setRecord failed to check result: %v", err)
			return nil, fmt.Errorf("failed to check set result: %w", err)
		}
		if rowsAffected == 0 {
			log.Printf("setRecord: no record set for ID %s", id)
			return nil, fmt.Errorf("no record set for ID %s", id)
		}

		log.Printf("setRecord succeeded for id: %s, value: %s", id, newValue)
		return []byte(newValue), nil

	case "delete":
		if id == "" {
			log.Printf("deleteRecord failed: no record ID provided")
			return nil, errors.New("invalid URI: no record ID provided for delete operation")
		}

		log.Printf("deleteRecord processing id: %s", id)

		result, err := r.DB.Exec("DELETE FROM records WHERE id = ?", id)
		if err != nil {
			log.Printf("deleteRecord failed to execute SQL: %v", err)
			return nil, fmt.Errorf("failed to delete record: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("deleteRecord failed to check result: %v", err)
			return nil, fmt.Errorf("failed to check delete result: %w", err)
		}

		log.Printf("deleteRecord succeeded for id: %s, removed %d records", id, rowsAffected)
		return []byte(fmt.Sprintf("Deleted %d record(s)", rowsAffected)), nil

	case "clear":
		if id != "_" {
			log.Printf("clear failed: invalid path, expected '/_'")
			return nil, errors.New("invalid URI: clear operation requires path '/_'")
		}

		log.Printf("clear processing")

		result, err := r.DB.Exec("DELETE FROM records")
		if err != nil {
			log.Printf("clear failed to execute SQL: %v", err)
			return nil, fmt.Errorf("failed to clear records: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("clear failed to check result: %v", err)
			return nil, fmt.Errorf("failed to check clear result: %w", err)
		}

		log.Printf("clear succeeded, removed %d records", rowsAffected)
		return []byte(fmt.Sprintf("Cleared %d records", rowsAffected)), nil

	default: // getRecord (no operation specified)
		if id == "" {
			log.Printf("getRecord failed: no record ID provided")
			return nil, errors.New("invalid URI: no record ID provided")
		}

		log.Printf("getRecord processing id: %s", id)

		var value string
		err := r.DB.QueryRow("SELECT value FROM records WHERE id = ?", id).Scan(&value)
		if err == sql.ErrNoRows {
			log.Printf("getRecord: no record found for id: %s", id)
			return []byte(""), nil // Return empty string for not found
		}
		if err != nil {
			log.Printf("getRecord failed to read record for id: %s, error: %v", id, err)
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		log.Printf("getRecord succeeded for id: %s, value: %s", id, value)
		return []byte(value), nil
	}
}

// InitializeDatabase sets up the SQLite database and creates the records table with retries.
func InitializeDatabase(dbPath string) (*sql.DB, error) {
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Attempt %d: Initializing SQLite database at %s", attempt, dbPath)
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			log.Printf("Attempt %d: Failed to open database: %v", attempt, err)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Verify connection
		if err := db.Ping(); err != nil {
			log.Printf("Attempt %d: Failed to ping database: %v", attempt, err)
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
			log.Printf("Attempt %d: Failed to create records table: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create records table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d", dbPath, attempt)
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
