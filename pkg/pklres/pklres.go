package pklres

import (
	"database/sql"
	"encoding/json"
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
	return "pklres"
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

// Read retrieves, sets, deletes, clears, or lists PKL records in the SQLite database based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Check if receiver is nil and initialize with fixed DBPath
	if r == nil {
		log.Printf("Warning: PklResourceReader is nil for URI: %s, initializing with DBPath", uri.String())
		newReader, err := InitializePklResource(r.DBPath)
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
	typ := query.Get("type")
	key := query.Get("key")

	log.Printf("Read called with URI: %s, operation: %s, type: %s, key: %s", uri.String(), operation, typ, key)

	switch operation {
	case "set":
		return r.setRecord(id, typ, key, query)
	case "delete":
		return r.deleteRecord(id, typ, key)
	case "clear":
		return r.clearRecords(typ)
	case "list":
		return r.listRecords(typ)
	default: // getRecord (no operation specified)
		return r.getRecord(id, typ, key)
	}
}

// setRecord stores a PKL record in the database
func (r *PklResourceReader) setRecord(id, typ, key string, query url.Values) ([]byte, error) {
	if id == "" || typ == "" {
		log.Printf("setRecord failed: id or type not provided")
		return nil, errors.New("set operation requires id and type parameters")
	}

	value := query.Get("value")
	if value == "" {
		log.Printf("setRecord failed: no value provided")
		return nil, errors.New("set operation requires a value parameter")
	}

	log.Printf("setRecord processing id: %s, type: %s, key: %s, value: %s", id, typ, key, value)

	var err error

	keyValue := key
	if keyValue == "" {
		keyValue = "" // Use empty string instead of NULL
	}

	// Use INSERT OR REPLACE with consistent key handling
	_, err = r.DB.Exec(
		"INSERT OR REPLACE INTO records (id, type, key, value) VALUES (?, ?, ?, ?)",
		id, typ, keyValue, value,
	)

	if err != nil {
		log.Printf("setRecord failed to execute SQL: %v", err)
		return nil, fmt.Errorf("failed to set record: %w", err)
	}

	log.Printf("setRecord succeeded for id: %s, type: %s, key: %s, value: %s", id, typ, key, value)
	return []byte(value), nil
}

// getRecord retrieves a PKL record from the database
func (r *PklResourceReader) getRecord(id, typ, key string) ([]byte, error) {
	if id == "" || typ == "" {
		log.Printf("getRecord failed: id or type not provided")
		return nil, errors.New("get operation requires id and type parameters")
	}

	log.Printf("getRecord processing id: %s, type: %s, key: %s", id, typ, key)

	var value string
	var err error

	keyValue := key
	if keyValue == "" {
		keyValue = "" // Use empty string instead of NULL
	}

	// Query with consistent key handling
	err = r.DB.QueryRow("SELECT value FROM records WHERE id = ? AND type = ? AND key = ?", id, typ, keyValue).Scan(&value)

	if err == sql.ErrNoRows {
		log.Printf("getRecord: no record found for id: %s, type: %s, key: %s", id, typ, key)
		return []byte(""), nil // Return empty string for not found
	}
	if err != nil {
		log.Printf("getRecord failed to read record for id: %s, type: %s, key: %s, error: %v", id, typ, key, err)
		return nil, fmt.Errorf("failed to read record: %w", err)
	}

	log.Printf("getRecord succeeded for id: %s, type: %s, key: %s, value: %s", id, typ, key, value)
	return []byte(value), nil
}

// deleteRecord removes a PKL record or key from the database
func (r *PklResourceReader) deleteRecord(id, typ, key string) ([]byte, error) {
	if id == "" || typ == "" {
		log.Printf("deleteRecord failed: id or type not provided")
		return nil, errors.New("delete operation requires id and type parameters")
	}

	log.Printf("deleteRecord processing id: %s, type: %s, key: %s", id, typ, key)

	var result sql.Result
	var err error

	if key != "" {
		// Delete specific key from record
		result, err = r.DB.Exec("DELETE FROM records WHERE id = ? AND type = ? AND key = ?", id, typ, key)
	} else {
		// Delete entire record (all keys for this id and type)
		result, err = r.DB.Exec("DELETE FROM records WHERE id = ? AND type = ?", id, typ)
	}

	if err != nil {
		log.Printf("deleteRecord failed to execute SQL: %v", err)
		return nil, fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("deleteRecord failed to check result: %v", err)
		return nil, fmt.Errorf("failed to check delete result: %w", err)
	}

	log.Printf("deleteRecord succeeded for id: %s, type: %s, key: %s, removed %d records", id, typ, key, rowsAffected)
	return []byte(fmt.Sprintf("Deleted %d record(s)", rowsAffected)), nil
}

// clearRecords removes all records of a specific type or all records
func (r *PklResourceReader) clearRecords(typ string) ([]byte, error) {
	if typ == "" {
		log.Printf("clearRecords failed: type not provided")
		return nil, errors.New("clear operation requires type parameter")
	}

	log.Printf("clearRecords processing type: %s", typ)

	var result sql.Result
	var err error

	if typ == "_" {
		// Clear all records regardless of type
		result, err = r.DB.Exec("DELETE FROM records")
	} else {
		// Clear only records of specific type
		result, err = r.DB.Exec("DELETE FROM records WHERE type = ?", typ)
	}

	if err != nil {
		log.Printf("clearRecords failed to execute SQL: %v", err)
		return nil, fmt.Errorf("failed to clear records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("clearRecords failed to check result: %v", err)
		return nil, fmt.Errorf("failed to check clear result: %w", err)
	}

	log.Printf("clearRecords succeeded for type: %s, removed %d records", typ, rowsAffected)
	return []byte(fmt.Sprintf("Cleared %d records", rowsAffected)), nil
}

// listRecords returns all record IDs of a specific type
func (r *PklResourceReader) listRecords(typ string) ([]byte, error) {
	if typ == "" {
		log.Printf("listRecords failed: type not provided")
		return nil, errors.New("list operation requires type parameter")
	}

	log.Printf("listRecords processing type: %s", typ)

	rows, err := r.DB.Query("SELECT DISTINCT id FROM records WHERE type = ? ORDER BY id", typ)
	if err != nil {
		log.Printf("listRecords failed to query records: %v", err)
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("listRecords failed to scan row: %v", err)
			return nil, fmt.Errorf("failed to scan record ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		log.Printf("listRecords failed during row iteration: %v", err)
		return nil, fmt.Errorf("failed to iterate records: %w", err)
	}

	// Ensure ids is not nil
	if ids == nil {
		ids = []string{}
	}

	// Serialize IDs as JSON array
	result, err := json.Marshal(ids)
	if err != nil {
		log.Printf("listRecords failed to marshal JSON: %v", err)
		return nil, fmt.Errorf("failed to serialize record IDs: %w", err)
	}

	log.Printf("listRecords succeeded for type: %s, found %d records", typ, len(ids))
	return result, nil
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

		// Create records table with type and optional key support
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS records (
				id TEXT NOT NULL,
				type TEXT NOT NULL,
				key TEXT DEFAULT '',
				value TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (id, type, key)
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

		// Create indexes for better performance
		_, err = db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_records_type ON records(type);
			CREATE INDEX IF NOT EXISTS idx_records_id_type ON records(id, type);
		`)
		if err != nil {
			log.Printf("Attempt %d: Failed to create indexes: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create indexes after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d", dbPath, attempt)
		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializePklResource creates a new PklResourceReader with an initialized SQLite database.
func InitializePklResource(dbPath string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}
