package memory

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
	return "memory"
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

// Read retrieves, sets, or clears items in the SQLite database based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Check if receiver is nil and initialize with fixed DBPath
	if r == nil {
		log.Printf("Warning: PklResourceReader is nil for URI: %s, initializing with DBPath", uri.String())
		newReader, err := InitializeMemory(r.DBPath)
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
			log.Printf("setItem failed: no item ID provided")
			return nil, errors.New("invalid URI: no item ID provided for set operation")
		}
		newValue := query.Get("value")
		if newValue == "" {
			log.Printf("setItem failed: no value provided")
			return nil, errors.New("set operation requires a value parameter")
		}

		log.Printf("setItem processing id: %s, value: %s", id, newValue)

		result, err := r.DB.Exec(
			"INSERT OR REPLACE INTO items (id, value) VALUES (?, ?)",
			id, newValue,
		)
		if err != nil {
			log.Printf("setItem failed to execute SQL: %v", err)
			return nil, fmt.Errorf("failed to set item: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("setItem failed to check result: %v", err)
			return nil, fmt.Errorf("failed to check set result: %w", err)
		}
		if rowsAffected == 0 {
			log.Printf("setItem: no item set for ID %s", id)
			return nil, fmt.Errorf("no item set for ID %s", id)
		}

		log.Printf("setItem succeeded for id: %s, value: %s", id, newValue)
		return []byte(newValue), nil

	case "clear":
		if id != "_" {
			log.Printf("clear failed: invalid path, expected '/_'")
			return nil, errors.New("invalid URI: clear operation requires path '/_'")
		}

		log.Printf("clear processing")

		result, err := r.DB.Exec("DELETE FROM items")
		if err != nil {
			log.Printf("clear failed to execute SQL: %v", err)
			return nil, fmt.Errorf("failed to clear items: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("clear failed to check result: %v", err)
			return nil, fmt.Errorf("failed to check clear result: %w", err)
		}

		log.Printf("clear succeeded, removed %d items", rowsAffected)
		return []byte(fmt.Sprintf("Cleared %d items", rowsAffected)), nil

	default: // getItem (no operation specified)
		if id == "" {
			log.Printf("getItem failed: no item ID provided")
			return nil, errors.New("invalid URI: no item ID provided")
		}

		log.Printf("getItem processing id: %s", id)

		var value string
		err := r.DB.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&value)
		if err == sql.ErrNoRows {
			log.Printf("getItem: no item found for id: %s", id)
			return []byte(""), nil // Return empty string for not found
		}
		if err != nil {
			log.Printf("getItem failed to read item for id: %s, error: %v", id, err)
			return nil, fmt.Errorf("failed to read item: %w", err)
		}

		log.Printf("getItem succeeded for id: %s, value: %s", id, value)
		return []byte(value), nil
	}
}

// InitializeDatabase sets up the SQLite database and creates the items table with retries.
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

		// Create items table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS items (
				id TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)
		`)
		if err != nil {
			log.Printf("Attempt %d: Failed to create items table: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create items table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d", dbPath, attempt)
		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeMemory creates a new PklResourceReader with an initialized SQLite database.
func InitializeMemory(dbPath string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}
