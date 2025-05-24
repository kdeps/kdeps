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

// Scheme returns the URI scheme for this reader.
func (r *PklResourceReader) Scheme() string {
	return "item"
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

// handleRollback logs and handles rollback errors, returning the primary error.
func handleRollback(tx *sql.Tx, primaryErr error, logMsg string, args ...interface{}) error {
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		log.Printf("setRecord failed to rollback transaction: %v", rollbackErr)
	}
	log.Printf(logMsg, args...)
	return fmt.Errorf("%s: %w", logMsg, primaryErr)
}

// getMostRecentID retrieves the ID of the most recent record.
func (r *PklResourceReader) getMostRecentID() (string, error) {
	var id string
	err := r.DB.QueryRow("SELECT id FROM items ORDER BY id DESC LIMIT 1").Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil // No records exist
	}
	if err != nil {
		return "", fmt.Errorf("failed to get most recent ID: %w", err)
	}
	return id, nil
}

// updateAdjacentRecord updates a record (previous or next) with the given value using a query.
func (r *PklResourceReader) updateAdjacentRecord(tx *sql.Tx, currentID, oldValue, query, direction string) error {
	var adjID string
	err := tx.QueryRow(query, currentID).Scan(&adjID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // No adjacent record, nothing to update
	}
	if err != nil {
		return handleRollback(tx, err, "setRecord failed to find %s record for id: %s, error: %v", direction, currentID, err)
	}

	_, err = tx.Exec("UPDATE items SET value = ? WHERE id = ?", oldValue, adjID)
	if err != nil {
		return handleRollback(tx, err, "setRecord failed to update %s record for id: %s, error: %v", direction, adjID, err)
	}

	log.Printf("setRecord updated %s record id: %s with value: %s", direction, adjID, oldValue)
	return nil
}

// updateAdjacentRecordByID updates a record by its ID with the given value, if the ID is non-empty.
func (r *PklResourceReader) updateAdjacentRecordByID(tx *sql.Tx, adjID, oldValue, direction string) error {
	if adjID == "" {
		return nil // No ID provided, skip update
	}

	_, err := tx.Exec("UPDATE items SET value = ? WHERE id = ?", oldValue, adjID)
	if err != nil {
		return handleRollback(tx, err, "setRecord failed to update %s record for id: %s, error: %v", direction, adjID, err)
	}

	log.Printf("setRecord updated %s record id: %s with value: %s", direction, adjID, oldValue)
	return nil
}

// updateAdjacentRecords handles updates for previous and next records, using refs if provided, else querying.
func (r *PklResourceReader) updateAdjacentRecords(tx *sql.Tx, id, oldValue, prevID, nextID string) error {
	if prevID != "" || nextID != "" {
		// Use provided refs
		if err := r.updateAdjacentRecordByID(tx, prevID, oldValue, "previous"); err != nil {
			return err
		}
		if err := r.updateAdjacentRecordByID(tx, nextID, oldValue, "next"); err != nil {
			return err
		}
	} else {
		// Fall back to querying
		if err := r.updateAdjacentRecord(tx, id, oldValue,
			"SELECT id FROM items WHERE id < ? ORDER BY id DESC LIMIT 1", "previous"); err != nil {
			return err
		}
		if err := r.updateAdjacentRecord(tx, id, oldValue,
			"SELECT id FROM items WHERE id > ? ORDER BY id ASC LIMIT 1", "next"); err != nil {
			return err
		}
	}
	return nil
}

// fetchValues retrieves values from the items table and returns them as a JSON array.
func (r *PklResourceReader) fetchValues(operation string) ([]byte, error) {
	log.Printf("%s processing", operation)

	rows, err := r.DB.Query("SELECT value FROM items ORDER BY id")
	if err != nil {
		log.Printf("%s failed to query records: %v", operation, err)
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			log.Printf("%s failed to scan row: %v", operation, err)
			return nil, fmt.Errorf("failed to scan record value: %w", err)
		}
		values = append(values, value)
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

	log.Printf("%s succeeded, found %d records", operation, len(values))
	return result, nil
}

// Read handles operations for retrieving, navigating, listing, or setting item records.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Handle nil receiver by initializing with DBPath
	if r == nil {
		log.Printf("Warning: PklResourceReader is nil for URI: %s, initializing with DBPath", uri.String())
		newReader, err := InitializeItem(r.DBPath, nil)
		if err != nil {
			log.Printf("Failed to initialize PklResourceReader in Read: %v", err)
			return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
		}
		r = newReader
		log.Printf("Initialized PklResourceReader with DBPath")
	}

	// Initialize database if DB is nil
	if r.DB == nil {
		log.Printf("Database connection is nil, attempting to initialize with path: %s", r.DBPath)
		maxAttempts := 5
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			db, err := InitializeDatabase(r.DBPath, nil)
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

		// Parse refs parameter for prev, current, next IDs
		var prevID, refCurrentID, nextID string
		isLoop := 0 // Default to false (0 in SQLite)
		if refs := query.Get("refs"); refs != "" {
			var refArray []string
			if err := json.Unmarshal([]byte(refs), &refArray); err != nil {
				log.Printf("setRecord failed to parse refs parameter: %v", err)
				return nil, fmt.Errorf("setRecord failed to parse refs parameter: %w", err)
			}
			if len(refArray) != 3 {
				log.Printf("setRecord failed: refs must contain exactly 3 elements")
				return nil, errors.New("setRecord failed: refs must contain exactly 3 elements")
			}
			prevID, refCurrentID, nextID = refArray[0], refArray[1], refArray[2]
			if refCurrentID == id {
				isLoop = 1 // Set isLoop to true (1 in SQLite)
			}
			log.Printf("setRecord refs parsed: prev=%s, current=%s, next=%s, isLoop=%d", prevID, refCurrentID, nextID, isLoop)
		}

		// Start a transaction
		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("setRecord failed to start transaction: %v", err)
			return nil, fmt.Errorf("failed to start transaction: %w", err)
		}

		// Get current value (if it exists)
		var oldValue string
		err = tx.QueryRow("SELECT value FROM items WHERE id = ?", id).Scan(&oldValue)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, handleRollback(tx, err, "setRecord failed to read current value for id: %s, error: %v", id, err)
		}

		// Set current record with isLoop flag
		result, err := tx.Exec(
			"INSERT OR REPLACE INTO items (id, value, isLoop) VALUES (?, ?, ?)",
			id, newValue, isLoop,
		)
		if err != nil {
			return nil, handleRollback(tx, err, "setRecord failed to execute SQL for current record: %v", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, handleRollback(tx, err, "setRecord failed to check result for current record: %v", err)
		}
		if rowsAffected == 0 {
			return nil, handleRollback(tx, fmt.Errorf("no record set for ID %s", id), "setRecord: no record set for ID %s", id)
		}

		// Update previous/next records if oldValue exists
		if oldValue != "" {
			if err := r.updateAdjacentRecords(tx, id, oldValue, prevID, nextID); err != nil {
				return nil, err
			}
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return nil, handleRollback(tx, err, "setRecord failed to commit transaction: %v", err)
		}

		log.Printf("setRecord succeeded for id: %s, value: %s, isLoop: %d", id, newValue, isLoop)
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
		if err == sql.ErrNoRows {
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
		if err == sql.ErrNoRows {
			log.Printf("nextRecord: no next record found for id: %s", currentID)
			return []byte(""), nil
		}
		if err != nil {
			log.Printf("nextRecord failed to read record for id: %s, error: %v", currentID, err)
			return nil, fmt.Errorf("failed to read next record: %w", err)
		}

		log.Printf("nextRecord succeeded for id: %s, value: %s", currentID, value)
		return []byte(value), nil

	case "list":
		return r.fetchValues("list")

	case "values":
		return r.fetchValues("values")

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
		if err == sql.ErrNoRows {
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

// InitializeDatabase sets up the SQLite database, creates the items table, and optionally populates it with initial items.
func InitializeDatabase(dbPath string, items []string) (*sql.DB, error) {
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

		// Create items table with isLoop column
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS items (
				id TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				isLoop INTEGER DEFAULT 0
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

		// If items are provided, insert them into the database
		if len(items) > 0 {
			tx, err := db.Begin()
			if err != nil {
				log.Printf("Attempt %d: Failed to start transaction for items initialization: %v", attempt, err)
				db.Close()
				if attempt == maxAttempts {
					return nil, fmt.Errorf("failed to start transaction for items initialization: %w", err)
				}
				time.Sleep(1 * time.Second)
				continue
			}

			for i, itemValue := range items {
				// Generate a unique ID for each item (e.g., timestamp-based with index)
				id := fmt.Sprintf("%s-%d", time.Now().Format("20060102150405.999999"), i)
				_, err = tx.Exec(
					"INSERT INTO items (id, value, isLoop) VALUES (?, ?, ?)",
					id, itemValue, 0, // isLoop set to 0 for initial items
				)
				if err != nil {
					tx.Rollback()
					log.Printf("Attempt %d: Failed to insert item %s: %v", attempt, itemValue, err)
					db.Close()
					if attempt == maxAttempts {
						return nil, fmt.Errorf("failed to insert item %s: %w", itemValue, err)
					}
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("Initialized item with id: %s, value: %s", id, itemValue)
			}

			if err := tx.Commit(); err != nil {
				tx.Rollback()
				log.Printf("Attempt %d: Failed to commit transaction for items initialization: %v", attempt, err)
				db.Close()
				if attempt == maxAttempts {
					return nil, fmt.Errorf("failed to commit transaction for items initialization: %w", err)
				}
				time.Sleep(1 * time.Second)
				continue
			}
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d with %d items", dbPath, attempt, len(items))
		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeItem creates a new PklResourceReader with an initialized SQLite database, optionally populated with items.
func InitializeItem(dbPath string, items []string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath, items)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}
