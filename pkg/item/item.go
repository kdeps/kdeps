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
	case "updateCurrent":
		newValue := query.Get("value")
		if newValue == "" {
			log.Printf("Error: set operation requires a value parameter")
			return nil, errors.New("set operation requires a value parameter")
		}

		// Generate a new ID (e.g., timestamp-based)
		id := time.Now().Format("20060102150405.999999")
		log.Printf("Processing item record: id=%s, value=%s", id, newValue)

		// Start transaction
		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("Error: Failed to start transaction: %v", err)
			return nil, fmt.Errorf("failed to start transaction: %w", err)
		}

		// Set current record
		result, err := tx.Exec(
			"INSERT OR REPLACE INTO items (id, value) VALUES (?, ?)",
			id, newValue,
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("Error: Failed to execute SQL for item id %s: %v", id, err)
			return nil, fmt.Errorf("failed to execute SQL for item: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("Error: Failed to check result for item id %s: %v", id, err)
			return nil, fmt.Errorf("failed to check result for item: %w", err)
		}
		if rowsAffected == 0 {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("Error: No item record set for id %s", id)
			return nil, fmt.Errorf("no item record set for id %s", id)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction: %v", rollbackErr)
			}
			log.Printf("Error: Failed to commit transaction: %v", err)
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("Successfully set item record: id=%s, value=%s", id, newValue)
		return []byte(newValue), nil

	case "set":
		newValue := query.Get("value")
		if newValue == "" {
			log.Printf("Error: set operation requires a value parameter")
			return nil, errors.New("set operation requires a value parameter")
		}

		// Get the most recent item ID
		currentID, err := r.getMostRecentID()
		if err != nil {
			log.Printf("Error: Failed to get most recent item ID: %v", err)
			return nil, err
		}
		if currentID == "" {
			log.Printf("Error: No item records found for set")
			return nil, errors.New("set: no current record exists")
		}

		// Generate a unique ID for the result entry
		resultID := fmt.Sprintf("%s-%d", time.Now().Format("20060102150405.999999"), time.Now().Nanosecond())

		// Start transaction
		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("Error: Failed to start transaction for set: %v", err)
			return nil, fmt.Errorf("failed to start transaction: %w", err)
		}

		// Append new result value for the current item
		result, err := tx.Exec(
			"INSERT INTO results (id, item_id, result_value, created_at) VALUES (?, ?, ?, ?)",
			resultID, currentID, newValue, time.Now(),
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set: %v", rollbackErr)
			}
			log.Printf("Error: Failed to execute SQL for item_id %s: %v", currentID, err)
			return nil, fmt.Errorf("failed to execute SQL for result: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set: %v", rollbackErr)
			}
			log.Printf("Error: Failed to check result for item_id %s: %v", currentID, err)
			return nil, fmt.Errorf("failed to check result for result: %w", err)
		}
		if rowsAffected == 0 {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set: %v", rollbackErr)
			}
			log.Printf("Error: No result appended for item_id %s", currentID)
			return nil, fmt.Errorf("no result appended for item_id %s", currentID)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set: %v", rollbackErr)
			}
			log.Printf("Error: Failed to commit transaction for set: %v", err)
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("Successfully appended result: item_id=%s, result_id=%s, value=%s", currentID, resultID, newValue)
		return []byte(newValue), nil

	case "results":
		log.Printf("Processing results query")

		rows, err := r.DB.Query(`
			SELECT r.result_value
			FROM results r
			JOIN items i ON r.item_id = i.id
			ORDER BY i.id, r.created_at
		`)
		if err != nil {
			log.Printf("Error: Failed to query results: %v", err)
			return nil, fmt.Errorf("failed to query results: %w", err)
		}
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				log.Printf("Error: Failed to scan result value: %v", err)
				return nil, fmt.Errorf("failed to scan result value: %w", err)
			}
			values = append(values, value)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error: Failed during result iteration: %v", err)
			return nil, fmt.Errorf("failed to iterate results: %w", err)
		}

		// Debug: Log values before marshaling
		log.Printf("Results before marshaling: %v", values)

		// Ensure values is not nil
		if values == nil {
			values = []string{}
		}

		// Serialize values as JSON array
		result, err := json.Marshal(values)
		if err != nil {
			log.Printf("Error: Failed to marshal results to JSON: %v", err)
			return nil, fmt.Errorf("failed to serialize result values: %w", err)
		}

		log.Printf("Successfully retrieved %d result records", len(values))
		return result, nil

	case "lastResult":
		log.Printf("Processing last result query")

		var value string
		err := r.DB.QueryRow(`
			SELECT r.result_value
			FROM results r
			JOIN items i ON r.item_id = i.id
			ORDER BY i.id DESC, r.created_at DESC
			LIMIT 1
		`).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No result found for lastResult")
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("Error: Failed to query last result: %v", err)
			return nil, fmt.Errorf("failed to read last result: %w", err)
		}

		log.Printf("Successfully retrieved last result: value=%s", value)
		return []byte(value), nil

	case "prev":
		log.Printf("Processing previous item record query")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("No item records found for prev query")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow(`
			SELECT value FROM items
			WHERE id < ? ORDER BY id DESC LIMIT 1`, currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No previous item record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("Error: Failed to read previous item record for id %s: %v", currentID, err)
			return nil, fmt.Errorf("failed to read previous record: %w", err)
		}

		log.Printf("Successfully retrieved previous item record: id=%s, value=%s", currentID, value)
		return []byte(value), nil

	case "next":
		log.Printf("Processing next item record query")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("No item records found for next query")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow(`
			SELECT value FROM items
			WHERE id > ? ORDER BY id ASC LIMIT 1`, currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No next item record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("Error: Failed to read next item record for id %s: %v", currentID, err)
			return nil, fmt.Errorf("failed to read next record: %w", err)
		}

		log.Printf("Successfully retrieved next item record: id=%s, value=%s", currentID, value)
		return []byte(value), nil

	case "current":
		log.Printf("Processing current item record query")

		currentID, err := r.getMostRecentID()
		if err != nil {
			return nil, err
		}
		if currentID == "" {
			log.Printf("No item records found for current query")
			return []byte(""), nil
		}

		var value string
		err = r.DB.QueryRow("SELECT value FROM items WHERE id = ?", currentID).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No item record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("Error: Failed to read item record for id %s: %v", currentID, err)
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		log.Printf("Successfully retrieved current item record: id=%s, value=%s", currentID, value)
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

// InitializeDatabase sets up the SQLite database and creates the items and results tables.
func InitializeDatabase(dbPath string, items []string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error: Failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Printf("Error: Failed to ping database: %v", err)
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
		log.Printf("Error: Failed to create items table: %v", err)
		db.Close()
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	// Create results table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			result_value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (item_id) REFERENCES items(id)
		)
	`)
	if err != nil {
		log.Printf("Error: Failed to create results table: %v", err)
		db.Close()
		return nil, fmt.Errorf("failed to create results table: %w", err)
	}

	// If items are provided, insert them into the database
	if len(items) > 0 {
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error: Failed to start transaction for items initialization: %v", err)
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
					log.Printf("Error: Failed to rollback transaction for item %s: %v", itemValue, rollbackErr)
				}
				log.Printf("Error: Failed to insert item %s: %v", itemValue, err)
				db.Close()
				return nil, fmt.Errorf("failed to insert item %s: %w", itemValue, err)
			}
			log.Printf("Initialized item: id=%s, value=%s", id, itemValue)
		}

		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for items initialization: %v", rollbackErr)
			}
			log.Printf("Error: Failed to commit transaction for items initialization: %v", err)
			db.Close()
			return nil, fmt.Errorf("failed to commit transaction for items initialization: %w", err)
		}
	}

	log.Printf("Successfully initialized SQLite database at %s with %d items", dbPath, len(items))
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
