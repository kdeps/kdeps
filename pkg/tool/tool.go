package tool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
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
		log.Printf("Warning: PklResourceReader is nil for URI: %s, initializing with DBPath", uri.String())
		newReader, err := InitializeTool(r.DBPath)
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
	case "run":
		if id == "" {
			log.Printf("runScript failed: no tool ID provided")
			return nil, errors.New("invalid URI: no tool ID provided for run operation")
		}
		script := query.Get("script")
		if script == "" {
			log.Printf("runScript failed: no script provided")
			return nil, errors.New("run operation requires a script parameter")
		}
		params := query.Get("params")

		log.Printf("runScript processing id: %s, script: %s, params: %s", id, script, params)

		// Decode URL-encoded params
		var paramList []string
		if params != "" {
			decodedParams, err := url.QueryUnescape(params)
			if err != nil {
				log.Printf("runScript failed to decode params: %v", err)
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

		log.Printf("Parsed parameters: %v", paramList)

		// Determine if script is a file path or inline script
		var output []byte
		var err error
		if _, statErr := os.Stat(script); statErr == nil {
			// Script is a file path; determine interpreter based on extension
			log.Printf("Executing file-based script: %s", script)
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
			log.Printf("Using interpreter: %s for script: %s", interpreter, script)
			logger := logging.GetLogger()
			args := append([]string{script}, paramList...)
			out, errStr, _, errExec := kdepsexec.KdepsExec(context.Background(), interpreter, args, "", false, false, logger)
			output = []byte(out + errStr)
			err = errExec
		} else {
			// Script is inline; pass script as $1 and params as $2, $3, etc.
			log.Printf("Executing inline script: %s", script)
			logger := logging.GetLogger()
			args := append([]string{"-c", script}, paramList...)
			out, errStr, _, errExec := kdepsexec.KdepsExec(context.Background(), "sh", args, "", false, false, logger)
			output = []byte(out + errStr)
			err = errExec
		}

		outputStr := string(output)
		if err != nil {
			log.Printf("runScript execution failed: %v, output: %s", err, outputStr)
			// Still store the output (which includes stderr) even if execution failed
		}

		// Store the output in the database, overwriting any existing record
		result, dbErr := r.DB.Exec(
			"INSERT OR REPLACE INTO tools (id, value) VALUES (?, ?)",
			id, outputStr,
		)
		if dbErr != nil {
			log.Printf("runScript failed to execute SQL: %v", dbErr)
			return nil, fmt.Errorf("failed to store script output: %w", dbErr)
		}

		rowsAffected, dbErr := result.RowsAffected()
		if dbErr != nil {
			log.Printf("runScript failed to check result: %v", dbErr)
			return nil, fmt.Errorf("failed to check run result: %w", dbErr)
		}
		if rowsAffected == 0 {
			log.Printf("runScript: no tool set for ID %s", id)
			return nil, fmt.Errorf("no tool set for ID %s", id)
		}

		// Append to history table
		_, dbErr = r.DB.Exec(
			"INSERT INTO history (id, value, timestamp) VALUES (?, ?, ?)",
			id, outputStr, time.Now().Unix(),
		)
		if dbErr != nil {
			log.Printf("runScript failed to append to history: %v", dbErr)
			// Note: Not failing the operation if history append fails
		}

		log.Printf("runScript succeeded for id: %s, output: %s", id, outputStr)
		return []byte(outputStr), nil

	case "history":
		if id == "" {
			log.Printf("history failed: no tool ID provided")
			return nil, errors.New("invalid URI: no tool ID provided for history operation")
		}

		log.Printf("history processing id: %s", id)

		rows, err := r.DB.Query("SELECT value, timestamp FROM history WHERE id = ? ORDER BY timestamp ASC", id)
		if err != nil {
			log.Printf("history failed to query: %v", err)
			return nil, fmt.Errorf("failed to retrieve history: %w", err)
		}
		defer rows.Close()

		var historyEntries []string
		for rows.Next() {
			var value string
			var timestamp int64
			if err := rows.Scan(&value, &timestamp); err != nil {
				log.Printf("history failed to scan row: %v", err)
				return nil, fmt.Errorf("failed to scan history row: %w", err)
			}
			formattedTime := time.Unix(timestamp, 0).Format(time.RFC3339)
			historyEntries = append(historyEntries, fmt.Sprintf("[%s] %s", formattedTime, value))
		}
		if err := rows.Err(); err != nil {
			log.Printf("history failed during row iteration: %v", err)
			return nil, fmt.Errorf("failed during history iteration: %w", err)
		}

		if len(historyEntries) == 0 {
			log.Printf("history: no entries found for id: %s", id)
			return []byte(""), nil
		}

		historyOutput := strings.Join(historyEntries, "\n")
		log.Printf("history succeeded for id: %s, entries: %d", id, len(historyEntries))
		return []byte(historyOutput), nil

	default: // getRecord (no operation specified)
		if id == "" {
			log.Printf("getRecord failed: no tool ID provided")
			return nil, errors.New("invalid URI: no tool ID provided")
		}

		log.Printf("getRecord processing id: %s", id)

		var value string
		err := r.DB.QueryRow("SELECT value FROM tools WHERE id = ?", id).Scan(&value)
		if err == sql.ErrNoRows {
			log.Printf("getRecord: no tool found for id: %s", id)
			return []byte(""), nil // Return empty string for not found
		}
		if err != nil {
			log.Printf("getRecord failed to read tool for id: %s, error: %v", id, err)
			return nil, fmt.Errorf("failed to read tool: %w", err)
		}

		log.Printf("getRecord succeeded for id: %s, value: %s", id, value)
		return []byte(value), nil
	}
}

// InitializeDatabase sets up the SQLite database and creates the tools and history tables with retries.
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

		// Create tools table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS tools (
				id TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)
		`)
		if err != nil {
			log.Printf("Attempt %d: Failed to create tools table: %v", attempt, err)
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
			log.Printf("Attempt %d: Failed to create history table: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create history table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d", dbPath, attempt)
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
