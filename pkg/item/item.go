package item

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/apple/pkl-go/pkl"
	_ "github.com/mattn/go-sqlite3"
)

// ItemReader is the interface for reading resources.
type ItemReader interface {
	pkl.ResourceReader
	Read(uri url.URL) ([]byte, error)
}

// PklResourceReader implements the ItemReader interface for the item scheme.
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
	log.Printf("%s processing", operation)

	rows, err := r.DB.Query("SELECT value FROM items ORDER BY id")
	if err != nil {
		log.Printf("%s failed to query records: %v", operation, err)
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	// Use a map to ensure uniqueness and a slice to maintain order
	valueMap := make(map[string]struct{})
	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			log.Printf("%s failed to scan row: %v", operation, err)
			return nil, fmt.Errorf("failed to scan record value: %w", err)
		}
		if _, exists := valueMap[value]; !exists {
			valueMap[value] = struct{}{}
			values = append(values, value)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("%s failed during row iteration: %v", operation, err)
		return nil, fmt.Errorf("failed to iterate records: %w", err)
	}

	// Debug: Log values before marshaling
	log.Printf("%s values before marshal: %v (nil: %t)", operation, values, values == nil)

	// Ensure values is not nil
	if values == nil {
		values = []string{}
	}

	// Serialize values as JSON array
	result, err := json.Marshal(values)
	if err != nil {
		log.Printf("%s failed to marshal JSON: %v", operation, err)
		return nil, fmt.Errorf("failed to serialize record values: %w", err)
	}

	log.Printf("%s succeeded, found %d unique records", operation, len(values))
	return result, nil
}

// Read handles operations for retrieving, navigating, listing, or setting item records.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Initialize database if DB is nil
	if r.DB == nil {
		log.Printf("Database connection is nil, attempting to initialize with path: %s", r.DBPath)
		db, err := InitializeDatabase(r.DBPath, nil)
		if err != nil {
			log.Printf("Failed to initialize database in Read: %v", err)
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}
		r.DB = db
		log.Printf("Database initialized successfully in Read")
	}

	query := uri.Query()
	operation := query.Get("op")

	log.Printf("Read called with URI: %s, operation: %s", uri.String(), operation)

	switch operation {
	case "set":
		newValue := query.Get("value")
		if newValue == "" {
			log.Printf("setRecord failed: no value provided")
			return nil, errors.New("set operation requires a value parameter")
		}

		// Generate a new ID (e.g., timestamp-based)
		id := time.Now().Format("20060102150405.999999")
		log.Printf("setRecord processing id: %s, value: %s", id, newValue)

		// Start a transaction
		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("setRecord failed to start transaction: %v", err)
			return nil, fmt.Errorf("failed to start transaction: %w", err)
		}

		// Set current record
		result, err := tx.Exec(
			"INSERT OR REPLACE INTO items (id, value) VALUES (?, ?)",
			id, newValue,
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("setRecord failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("setRecord failed to execute SQL for current record: %v", err)
			return nil, fmt.Errorf("setRecord failed to execute SQL for current record: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("setRecord failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("setRecord failed to check result for current record: %v", err)
			return nil, fmt.Errorf("setRecord failed to check result for current record: %w", err)
		}
		if rowsAffected == 0 {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("setRecord failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("setRecord: no record set for ID %s", id)
			return nil, fmt.Errorf("setRecord: no record set for ID %s", id)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("setRecord failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("setRecord failed to commit transaction: %v", err)
			return nil, fmt.Errorf("setRecord failed to commit transaction: %w", err)
		}

		log.Printf("setRecord succeeded for id: %s, value: %s", id, newValue)
		return []byte(newValue), nil

	case "prev":
		log.Printf("prevRecord processing")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("prevRecord: no records found")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow(`
			SELECT value FROM items
			WHERE id < ? ORDER BY id DESC LIMIT 1`, currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("prevRecord: no previous record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("prevRecord failed to read record for id: %s, error: %v", currentID, err)
			return nil, fmt.Errorf("failed to read previous record: %w", err)
		}

		log.Printf("prevRecord succeeded for id: %s, value: %s", currentID, value)
		return []byte(value), nil

	case "next":
		log.Printf("nextRecord processing")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("nextRecord: no records found")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow(`
			SELECT value FROM items
			WHERE id > ? ORDER BY id ASC LIMIT 1`, currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("nextRecord: no next record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("nextRecord failed to read record for id: %s, error: %v", currentID, err)
			return nil, fmt.Errorf("failed to read next record: %w", err)
		}

		log.Printf("nextRecord succeeded for id: %s, value: %s", currentID, value)
		return []byte(value), nil

	case "list", "values":
		return r.fetchValues(operation)

	case "current":
		log.Printf("getRecord processing")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("getRecord: no records found")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow("SELECT value FROM items WHERE id = ?", currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("getRecord: no record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("getRecord failed to read record for id: %s, error: %v", currentID, err)
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		log.Printf("getRecord succeeded for id: %s, value: %s", currentID, value)
		return []byte(value), nil

	default:
		return nil, errors.New("invalid operation")
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
		log.Printf("Failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Printf("Failed to ping database: %v", err)
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
		log.Printf("Failed to create items table: %v", err)
		db.Close()
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	// If items are provided, insert them into the database
	if len(items) > 0 {
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Failed to start transaction for items initialization: %v", err)
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
					log.Printf("Failed to rollback transaction for item %s: %v", itemValue, rollbackErr)
				}
				log.Printf("Failed to insert item %s: %v", itemValue, err)
				db.Close()
				return nil, fmt.Errorf("failed to insert item %s: %w", itemValue, err)
			}
			log.Printf("Initialized item with id: %s, value: %s", id, itemValue)
		}

		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Failed to rollback transaction for items initialization: %v", rollbackErr)
			}
			log.Printf("Failed to commit transaction for items initialization: %v", err)
			db.Close()
			return nil, fmt.Errorf("failed to commit transaction for items initialization: %w", err)
		}
	}

	log.Printf("SQLite database initialized successfully at %s with %d items", dbPath, len(items))
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
