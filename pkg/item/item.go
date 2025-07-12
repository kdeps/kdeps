package item

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/apple/pkl-go/pkl"
	_ "github.com/mattn/go-sqlite3"
)

// PklResourceReader implements the pkl.ResourceReader interface for the item scheme.
type PklResourceReader struct {
	DB     *sql.DB
	DBPath string // Store dbPath for reinitialization
}

// IsGlobbable indicates whether the reader supports globbing (not supported here).
func (r *PklResourceReader) IsGlobbable() bool {
	return false
}

// HasHierarchicalUris indicates whether URIs are hierarchical (not supported here).
func (r *PklResourceReader) HasHierarchicalUris() bool {
	return false
}

// ListElements is not used in this implementation.
func (r *PklResourceReader) ListElements(_ url.URL) ([]pkl.PathElement, error) {
	return nil, nil
}

// Scheme returns the URI scheme for this reader.
func (r *PklResourceReader) Scheme() string {
	return "item"
}

// fetchValues retrieves unique values from the items table and returns them as a JSON array.
func (r *PklResourceReader) fetchValues(operation string) ([]byte, error) {
	rows, err := r.DB.Query("SELECT value FROM items ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	// Use a map to ensure uniqueness and a slice to maintain order
	valueMap := make(map[string]struct{})
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan record value: %w", err)
		}
		if _, exists := valueMap[value]; !exists {
			valueMap[value] = struct{}{}
			values = append(values, value)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate records: %w", err)
	}

	// Ensure values is not nil
	if values == nil {
		values = []string{}
	}

	// Serialize values as JSON array
	result, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize record values: %w", err)
	}

	return result, nil
}

// Read handles operations for retrieving, navigating, listing, or setting item records.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Initialize database if DB is nil or closed
	if r.DB == nil {
		db, err := InitializeDatabase(r.DBPath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}
		r.DB = db
	} else {
		// Check if the database is closed and reconnect if necessary
		if err := r.DB.Ping(); err != nil {
			// Database is closed, try to reconnect
			db, err := InitializeDatabase(r.DBPath, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to reconnect to database: %w", err)
			}
			r.DB = db
		}
	}

	query := uri.Query()
	operation := query.Get("op")

	switch operation {
	case "":
		return r.getCurrentRecord()
	case "set":
		return r.setRecord(query)
	case "current":
		return r.getCurrentRecord()
	case "next":
		return r.getNextRecord()
	case "prev":
		return r.getPrevRecord()
	case "list":
		return r.listRecords()
	case "values":
		return r.getValues()
	default:
		return nil, fmt.Errorf("invalid operation: %s", operation)
	}
}

// getMostRecentID retrieves the ID of the most recent record.
func (r *PklResourceReader) getMostRecentID() (string, error) {
	var id string
	err := r.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil // No records exist
	}
	if err != nil {
		return "", fmt.Errorf("failed to get most recent ID: %w", err)
	}
	return id, nil
}

// InitializeDatabase sets up the SQLite database and creates the items table.
func InitializeDatabase(dbPath string, items []string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	// If items are provided, insert them into the database
	if len(items) > 0 {
		tx, err := db.Begin()
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to start transaction for items initialization: %w", err)
		}

		for i, itemValue := range items {
			// Generate a unique ID for each item
			id := fmt.Sprintf("%s-%d", time.Now().Format("20060102150405.999999"), i)
			_, err = tx.Exec(
				"INSERT INTO items (id, value) VALUES (?, ?)",
				id, itemValue,
			)
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					// Log rollback error but don't return it as the main error
				}
				db.Close()
				return nil, fmt.Errorf("failed to insert item %s: %w", itemValue, err)
			}
		}

		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				// Log rollback error but don't return it as the main error
			}
			db.Close()
			return nil, fmt.Errorf("failed to commit transaction for items initialization: %w", err)
		}
	}

	return db, nil
}

// InitializeItem creates a new PklResourceReader with an initialized SQLite database.
func InitializeItem(dbPath string, items []string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath, items)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}

// setRecord stores a new item record in the database
func (r *PklResourceReader) setRecord(query url.Values) ([]byte, error) {
	newValue := query.Get("value")
	if newValue == "" {
		return nil, errors.New("set operation requires a value parameter")
	}

	// Generate a new ID (e.g., timestamp-based)
	id := time.Now().Format("20060102150405.999999")

	// Start a transaction
	tx, err := r.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Set current record
	result, err := tx.Exec(
		"INSERT OR REPLACE INTO items (id, value) VALUES (?, ?)",
		id, newValue,
	)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't return it as the main error
		}
		return nil, fmt.Errorf("failed to execute SQL for current record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't return it as the main error
		}
		return nil, fmt.Errorf("failed to check result for current record: %w", err)
	}
	if rowsAffected == 0 {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't return it as the main error
		}
		return nil, fmt.Errorf("no record set for ID %s", id)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// Log rollback error but don't return it as the main error
		}
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return []byte(newValue), nil
}

// getCurrentRecord retrieves the most recent item record
func (r *PklResourceReader) getCurrentRecord() ([]byte, error) {
	currentID, err := r.getMostRecentID()
	if err != nil {
		return nil, err
	}
	if currentID == "" {
		return []byte(""), nil
	}

	var value string
	err = r.DB.QueryRow("SELECT value FROM items WHERE id = ?", currentID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return []byte(""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read record: %w", err)
	}

	return []byte(value), nil
}

// getNextRecord retrieves the next item record after the current one
func (r *PklResourceReader) getNextRecord() ([]byte, error) {
	currentID, err := r.getMostRecentID()
	if err != nil {
		return nil, err
	}
	if currentID == "" {
		return []byte(""), nil
	}

	var value string
	err = r.DB.QueryRow(`
		SELECT value FROM items
		WHERE id > ? ORDER BY id ASC LIMIT 1`, currentID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return []byte(""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read next record: %w", err)
	}

	return []byte(value), nil
}

// getPrevRecord retrieves the previous item record before the current one
func (r *PklResourceReader) getPrevRecord() ([]byte, error) {
	currentID, err := r.getMostRecentID()
	if err != nil {
		return nil, err
	}
	if currentID == "" {
		return []byte(""), nil
	}

	var value string
	err = r.DB.QueryRow(`
		SELECT value FROM items
		WHERE id < ? ORDER BY id DESC LIMIT 1`, currentID).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return []byte(""), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read previous record: %w", err)
	}

	return []byte(value), nil
}

// listRecords lists all item records
func (r *PklResourceReader) listRecords() ([]byte, error) {
	return r.fetchValues("list")
}

// getValues retrieves unique values from the items table
func (r *PklResourceReader) getValues() ([]byte, error) {
	return r.fetchValues("values")
}
