package tool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
)

// PklResourceReader implements the pkl.ResourceReader interface for SQLite.
type PklResourceReader struct {
	DB     *sql.DB
	DBPath string // Store dbPath for reinitialization
}

// Scheme returns the URI scheme for this reader.
func (r *PklResourceReader) Scheme() string {
	return "tool"
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

// Read retrieves, runs, or retrieves history of script outputs in the SQLite database based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Check if receiver is nil and initialize with fixed DBPath
	if r == nil {
		newReader, err := InitializeTool(r.DBPath)
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
	case "run":
		return r.runScript(id, query)
	case "get":
		return r.getRecord(id)
	case "set":
		return r.setRecord(id, query)
	case "delete":
		return r.deleteRecord(id)
	case "clear":
		return r.clearRecords()
	case "history":
		return r.getHistory(id)
	case "list":
		return r.listRecords()
	default: // getRecord (no operation specified)
		return r.getRecord(id)
	}
}

// InitializeDatabase sets up the SQLite database and creates the tools and history tables with retries.
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

		// Create tools table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS tools (
				id TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create tools table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Create history table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS history (
				id TEXT NOT NULL,
				value TEXT NOT NULL,
				timestamp INTEGER NOT NULL
			)
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create history table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeTool creates a new PklResourceReader with an initialized SQLite database.
func InitializeTool(dbPath string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	return &PklResourceReader{DB: db, DBPath: dbPath}, nil
}

// runScript executes a script and stores the output
func (r *PklResourceReader) runScript(id string, query url.Values) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no tool ID provided for run operation")
	}
	script := query.Get("script")
	if script == "" {
		return nil, errors.New("run operation requires a script parameter")
	}
	params := query.Get("params")

	// Decode URL-encoded params
	var paramList []string
	if params != "" {
		decodedParams, err := url.QueryUnescape(params)
		if err != nil {
			return nil, fmt.Errorf("failed to decode params: %w", err)
		}
		// Split params by spaces and trim whitespace
		for _, p := range strings.Split(decodedParams, " ") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				paramList = append(paramList, trimmed)
			}
		}
	}

	// Determine if script is a file path or inline script
	var output []byte
	var err error
	if _, statErr := os.Stat(script); statErr == nil {
		// Script is a file path; determine interpreter based on extension
		extension := strings.ToLower(filepath.Ext(script))
		var interpreter string
		switch extension {
		case ".py":
			interpreter = "python3"
		case ".ts":
			interpreter = "ts-node"
		case ".js":
			interpreter = "node"
		case ".rb":
			interpreter = "ruby"
		default:
			interpreter = "sh"
		}
		logger := logging.GetLogger()
		args := append([]string{script}, paramList...)
		out, errStr, _, errExec := kdepsexec.KdepsExec(context.Background(), interpreter, args, "", false, false, logger)
		output = []byte(out + errStr)
		err = errExec
	} else {
		// Script is inline; pass script as $1 and params as $2, $3, etc.
		logger := logging.GetLogger()
		args := append([]string{"-c", script}, paramList...)
		out, errStr, _, errExec := kdepsexec.KdepsExec(context.Background(), "sh", args, "", false, false, logger)
		output = []byte(out + errStr)
		err = errExec
	}

	outputStr := string(output)
	if err != nil {
		// Still store the output (which includes stderr) even if execution failed
	}

	// Store the output in the database, overwriting any existing record
	result, dbErr := r.DB.Exec(
		"INSERT OR REPLACE INTO tools (id, value) VALUES (?, ?)",
		id, outputStr,
	)
	if dbErr != nil {
		return nil, fmt.Errorf("failed to store script output: %w", dbErr)
	}

	rowsAffected, dbErr := result.RowsAffected()
	if dbErr != nil {
		return nil, fmt.Errorf("failed to check run result: %w", dbErr)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no tool set for ID %s", id)
	}

	// Append to history table
	_, dbErr = r.DB.Exec(
		"INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)",
		id, outputStr, time.Now().Unix(),
	)
	if dbErr != nil {
		// Note: Not failing the operation if history append fails
	}

	return []byte(outputStr), nil
}

// getRecord retrieves a tool record from the database
func (r *PklResourceReader) getRecord(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no tool ID provided")
	}

	var value string
	err := r.DB.QueryRow("SELECT value FROM tools WHERE id = ?", id).Scan(&value)
	if err == sql.ErrNoRows {
		return []byte(""), nil // Return empty string for not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read tool: %w", err)
	}

	return []byte(value), nil
}

// setRecord stores a tool record in the database
func (r *PklResourceReader) setRecord(id string, query url.Values) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no tool ID provided for set operation")
	}
	newValue := query.Get("value")
	if newValue == "" {
		return nil, errors.New("set operation requires a value parameter")
	}

	result, err := r.DB.Exec(
		"INSERT OR REPLACE INTO tools (id, value) VALUES (?, ?)",
		id, newValue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check set result: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("no tool set for ID %s", id)
	}

	return []byte(newValue), nil
}

// deleteRecord removes a tool record from the database
func (r *PklResourceReader) deleteRecord(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no tool ID provided for delete operation")
	}

	result, err := r.DB.Exec("DELETE FROM tools WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete tool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check delete result: %w", err)
	}

	return []byte(fmt.Sprintf("Deleted %d tool(s)", rowsAffected)), nil
}

// clearRecords removes all tool records from the database
func (r *PklResourceReader) clearRecords() ([]byte, error) {
	result, err := r.DB.Exec("DELETE FROM tools")
	if err != nil {
		return nil, fmt.Errorf("failed to clear tools: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check clear result: %w", err)
	}

	return []byte(fmt.Sprintf("Cleared %d tools", rowsAffected)), nil
}

// getHistory retrieves the execution history for a tool
func (r *PklResourceReader) getHistory(id string) ([]byte, error) {
	if id == "" {
		return nil, errors.New("invalid URI: no tool ID provided for history operation")
	}

	rows, err := r.DB.Query("SELECT value, timestamp FROM history WHERE id = ? ORDER BY timestamp ASC", id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve history: %w", err)
	}
	defer rows.Close()

	var historyEntries []string
	for rows.Next() {
		var value string
		var timestamp int64
		if err := rows.Scan(&value, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		formattedTime := time.Unix(timestamp, 0).Format(time.RFC3339)
		historyEntries = append(historyEntries, fmt.Sprintf("[%s] %s", formattedTime, value))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed during history iteration: %w", err)
	}

	if len(historyEntries) == 0 {
		return []byte(""), nil
	}

	historyOutput := strings.Join(historyEntries, "\n")
	return []byte(historyOutput), nil
}

// listRecords lists all tool records in the database
func (r *PklResourceReader) listRecords() ([]byte, error) {
	rows, err := r.DB.Query("SELECT id FROM tools ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	defer rows.Close()

	var toolIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan tool ID: %w", err)
		}
		toolIDs = append(toolIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed during tool iteration: %w", err)
	}

	return []byte(strings.Join(toolIDs, "\n")), nil
}
