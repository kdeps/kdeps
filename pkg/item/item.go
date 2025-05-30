package item

import (
	"context"
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

// PklResourceReader implements the pkl.ResourceReader interface for the item scheme.
type PklResourceReader struct {
	DB       *sql.DB
	DBPath   string
	ActionID string
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

// getItemRecord retrieves an item record based on the specified operation ("prev", "next", or "current").
func (r *PklResourceReader) getItemRecord(ctx context.Context, operation string) ([]byte, error) {
	if r.ActionID == "" {
		log.Printf("Error: %s operation requires initialized actionID", operation)
		return []byte(""), nil
	}
	log.Printf("Processing %s item record query, actionID: %s", operation, r.getActionIDForLog())

	currentID, dbActionID, err := r.getMostRecentIDWithActionID()
	if err != nil {
		log.Printf("Error: Failed to get most recent item ID, actionID: %s: %v", r.getActionIDForLog(), err)
		return []byte(""), nil
	}
	if currentID == "" {
		log.Printf("No item records found for %s query, actionID: %s", operation, r.getActionIDForLog())
		return []byte(""), nil
	}

	if r.ActionID != dbActionID {
		log.Printf("Error: actionID mismatch for %s: expected %s, got %s", operation, r.getActionIDForLog(), dbActionID)
		return []byte(""), nil
	}

	var query string
	switch operation {
	case "prev":
		query = `SELECT value, action_id FROM items WHERE id < ? AND action_id = ? ORDER BY id DESC LIMIT 1`
	case "next":
		query = `SELECT value, action_id FROM items WHERE id > ? AND action_id = ? ORDER BY id ASC LIMIT 1`
	case "current":
		query = `SELECT value, action_id FROM items WHERE id = ? AND action_id = ?`
	default:
		return []byte(""), fmt.Errorf("invalid operation: %s", operation)
	}

	var value, actionID string
	err = r.DB.QueryRowContext(ctx, query, currentID, r.ActionID).Scan(&value, &actionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("Error: Failed to read %s item record for id %s, actionID: %s: %v", operation, currentID, r.getActionIDForLog(), err)
		return []byte(""), nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("No %s item record found for id: %s, actionID: %s", operation, currentID, r.getActionIDForLog())
		return []byte(""), nil
	}

	if r.ActionID != actionID {
		log.Printf("Error: actionID mismatch for %s: expected %s, got %s", operation, r.getActionIDForLog(), actionID)
		return []byte(""), nil
	}

	log.Printf("Successfully retrieved %s item record: id=%s, value=%s, actionID: %s", operation, currentID, value, r.getActionIDForLog())
	return []byte(value), nil
}

// Read handles operations for retrieving, navigating, listing, or setting item records.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	ctx := context.Background()

	// Initialize database if nil
	if r.DB == nil {
		log.Printf("Database connection is nil, attempting to initialize with path: %s", r.DBPath)
		db, err := InitializeDatabase(r.DBPath, nil)
		if err != nil {
			log.Printf("Failed to initialize database in Read, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}
		r.DB = db
		log.Printf("Database initialized successfully in Read, actionID: %s", r.getActionIDForLog())
	}

	// Extract actionID from URI path (e.g., item:/{actionID} or item:/_)
	path := uri.Path
	var uriActionID string
	if path != "" && path != "/" && path != "/_" {
		uriActionID = strings.TrimPrefix(path, "/")
	}
	log.Printf("Read called with URI: %s, uriActionID: %s, operation: %s, reader ActionID: %s",
		uri.String(), uriActionID, uri.Query().Get("op"), r.getActionIDForLog())

	query := uri.Query()
	operation := query.Get("op")

	switch operation {
	case "updateCurrent":
		newValue := query.Get("value")
		id := time.Now().Format("20060102150405.999999")
		log.Printf("Processing item record: id=%s, value=%s, actionID: %s", id, newValue, r.getActionIDForLog())

		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("Error: Failed to start transaction, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		result, err := tx.Exec(
			"INSERT OR REPLACE INTO items (id, value, action_id) VALUES (?, ?, ?)",
			id, newValue, r.ActionID,
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to execute SQL for item id %s, actionID: %s: %v", id, r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to check result for item id %s, actionID: %s: %v", id, r.getActionIDForLog(), err)
			return []byte(""), nil
		}
		if rowsAffected == 0 {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: No item record set for id %s, actionID: %s", id, r.getActionIDForLog())
			return []byte(""), nil
		}

		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to commit transaction, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		log.Printf("Successfully set item record: id=%s, value=%s, actionID: %s", id, newValue, r.getActionIDForLog())
		return []byte(newValue), nil

	case "set":
		newValue := query.Get("value")
		currentID, _, err := r.getMostRecentIDWithActionID()
		if err != nil {
			log.Printf("Error: Failed to get most recent item ID, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		resultID := fmt.Sprintf("%s-%d", time.Now().Format("20060102150405.999999"), time.Now().Nanosecond())

		tx, err := r.DB.Begin()
		if err != nil {
			log.Printf("Error: Failed to start transaction for set, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		result, err := tx.Exec(
			"INSERT INTO results (id, item_id, result_value, created_at, action_id) VALUES (?, ?, ?, ?, ?)",
			resultID, currentID, newValue, time.Now(), r.ActionID,
		)
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to execute SQL for item_id %s, actionID: %s: %v", currentID, r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to check result for item_id %s, actionID: %s: %v", currentID, r.getActionIDForLog(), err)
			return []byte(""), nil
		}
		if rowsAffected == 0 {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: No result appended for item_id %s, actionID: %s", currentID, r.getActionIDForLog())
			return []byte(""), nil
		}

		if err := tx.Commit(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error: Failed to rollback transaction for set, actionID: %s: %v", r.getActionIDForLog(), rollbackErr)
			}
			log.Printf("Error: Failed to commit transaction for set, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}

		log.Printf("Successfully appended result: item_id=%s, result_id=%s, value=%s, actionID: %s",
			currentID, resultID, newValue, r.getActionIDForLog())
		return []byte(newValue), nil

	case "values":
		log.Printf("Processing results query for uriActionID: %s, reader actionID: %s", uriActionID, r.getActionIDForLog())

		rows, err := r.DB.QueryContext(ctx, `
			SELECT r.result_value
			FROM results r
			JOIN items i ON r.item_id = i.id
			WHERE r.action_id = ?
			ORDER BY i.id, r.created_at
		`, uriActionID)
		if err != nil {
			log.Printf("Error: Failed to query results for actionID %s: %v", uriActionID, err)
			return []byte(""), nil
		}
		defer rows.Close()

		var values []string
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				log.Printf("Error: Failed to scan result value for actionID %s: %v", uriActionID, err)
				return []byte(""), nil
			}
			values = append(values, value)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error: Failed during result iteration for actionID %s: %v", uriActionID, err)
			return []byte(""), nil
		}

		result, err := json.Marshal(values)
		if err != nil {
			log.Printf("Error: Failed to marshal results to JSON for actionID %s: %v", uriActionID, err)
			return []byte(""), nil
		}

		log.Printf("Successfully retrieved %d result records for actionID %s: %s", len(values), uriActionID, result)
		return result, nil

	case "lastResult":
		if r.ActionID == "" {
			log.Printf("Error: lastResult operation requires initialized actionID")
			return []byte(""), nil
		}
		log.Printf("Processing last result query, actionID: %s", r.getActionIDForLog())

		var value, actionID string
		err := r.DB.QueryRowContext(ctx, `
			SELECT r.result_value, r.action_id
			FROM results r
			JOIN items i ON r.item_id = i.id
			WHERE r.action_id = ?
			ORDER BY i.id DESC, r.created_at DESC
			LIMIT 1
		`, r.ActionID).Scan(&value, &actionID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Error: Failed to query last result, actionID: %s: %v", r.getActionIDForLog(), err)
			return []byte(""), nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No result found for lastResult, actionID: %s", r.getActionIDForLog())
			return []byte(""), nil
		}

		if r.ActionID != actionID {
			log.Printf("Error: actionID mismatch for lastResult: expected %s, got %s", r.getActionIDForLog(), actionID)
			return []byte(""), nil
		}

		log.Printf("Successfully retrieved last result: value=%s, actionID: %s", value, r.getActionIDForLog())
		return []byte(value), nil

	case "prev", "next", "current":
		return r.getItemRecord(ctx, operation)

	default:
		log.Printf("Error: Invalid operation %s, actionID: %s", operation, r.getActionIDForLog())
		return []byte(""), nil
	}
}

// getActionIDForLog returns ActionID or "<uninitialized>" for logging.
func (r *PklResourceReader) getActionIDForLog() string {
	if r.ActionID == "" {
		return "<uninitialized>"
	}
	return r.ActionID
}

// getMostRecentIDWithActionID retrieves the ID and action_id of the most recent record.
func (r *PklResourceReader) getMostRecentIDWithActionID() (string, string, error) {
	var id, actionID string
	err := r.DB.QueryRow("SELECT id, action_id FROM items ORDER BY id DESC LIMIT 1").Scan(&id, &actionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("failed to get most recent ID: %w", err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil // No records exist
	}
	return id, actionID, nil
}

// InitializeDatabase sets up the SQLite database and creates the items and results tables.
func InitializeDatabase(dbPath string, items []string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error: Failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		log.Printf("Error: Failed to ping database: %v", err)
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create items table with action_id (nullable)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			action_id TEXT
		)
	`)
	if err != nil {
		log.Printf("Error: Failed to create items table: %v", err)
		db.Close()
		return nil, fmt.Errorf("failed to create items table: %w", err)
	}

	// Create results table with action_id (nullable)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS results (
			id TEXT PRIMARY KEY,
			item_id TEXT NOT NULL,
			result_value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			action_id TEXT,
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
			id := fmt.Sprintf("%s-%d", time.Now().Format("20060102150405.999999"), i)
			// Use empty action_id for initialization
			_, err = tx.Exec(
				"INSERT INTO items (id, value, action_id) VALUES (?, ?, ?)",
				id, itemValue, "",
			)
			if err != nil {
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					log.Printf("Error: Failed to rollback transaction for item %s: %v", itemValue, rollbackErr)
				}
				log.Printf("Error: Failed to insert item %s: %v", itemValue, err)
				db.Close()
				return nil, fmt.Errorf("failed to insert item %s: %w", itemValue, err)
			}
			log.Printf("Initialized item: id=%s, value=%s, action_id=<empty>", id, itemValue)
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
func InitializeItem(dbPath string, items []string, actionID string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath, items)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	return &PklResourceReader{DB: db, DBPath: dbPath, ActionID: actionID}, nil
}
