package pklres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/agent"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
)

// PklResourceReader implements the pkl.ResourceReader interface for SQLite.
type PklResourceReader struct {
	DB      *sql.DB
	DBPath  string // Store dbPath for reinitialization
	Fs      afero.Fs
	Logger  *slog.Logger
	GraphID string // Current graphID for scoping database operations
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
		newReader, err := InitializePklResource(r.DBPath, "default")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
		}
		r = newReader
	}

	// Check if db is nil or closed and initialize with retries
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
	} else {
		// Check if the database is closed and reconnect if necessary
		if err := r.DB.Ping(); err != nil {
			// Database is closed, try to reconnect
			maxAttempts := 5
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				db, err := InitializeDatabase(r.DBPath)
				if err == nil {
					r.DB = db
					break
				}
				if attempt == maxAttempts {
					return nil, fmt.Errorf("failed to reconnect to database after %d attempts: %w", maxAttempts, err)
				}
				time.Sleep(1 * time.Second)
			}
		}
	}

	id := strings.TrimPrefix(uri.Path, "/")
	query := uri.Query()
	operation := query.Get("op")
	typ := query.Get("type")
	key := query.Get("key")

	switch operation {
	case "set":
		return r.setRecord(id, typ, key, query)
	case "delete":
		return r.deleteRecord(id, typ, key)
	case "clear":
		return r.clearRecords(typ)
	case "list":
		return r.listRecords(typ)
	case "output":
		return r.getResourceOutput(id, typ)
	case "eval":
		return r.evaluateResource(id, typ)
	default: // getRecord (no operation specified)
		return r.getRecord(id, typ, key)
	}
}

// setRecord stores a PKL record in the database
func (r *PklResourceReader) setRecord(id, typ, key string, query url.Values) ([]byte, error) {
	if id == "" || typ == "" {
		r.Logger.Error("setRecord failed: id or type not provided")
		return nil, errors.New("set operation requires id and type parameters")
	}

	value := query.Get("value")
	if value == "" {
		r.Logger.Error("setRecord failed: no value provided")
		return nil, errors.New("set operation requires a value parameter")
	}

	r.Logger.Debug("setRecord processing", "id", id, "type", typ, "key", key, "value", value)

	keyValue := key
	if keyValue == "" {
		keyValue = "" // Use empty string instead of NULL
	}

	// Start a transaction for atomic operation
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Error("setRecord failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Error("setRecord: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	// Use INSERT OR REPLACE with graphID and consistent key handling
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO records (graph_id, id, type, key, value) VALUES (?, ?, ?, ?, ?)",
		r.GraphID, id, typ, keyValue, value,
	)
	if err != nil {
		r.Logger.Error("setRecord failed to execute SQL", "error", err)
		return nil, fmt.Errorf("failed to set record: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Error("setRecord failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("setRecord succeeded", "id", id, "type", typ, "key", key, "value", value)
	return []byte(value), nil
}

// getRecord retrieves a PKL record from the database
func (r *PklResourceReader) getRecord(id, typ, key string) ([]byte, error) {
	if id == "" || typ == "" {
		return nil, errors.New("get operation requires id and type parameters")
	}

	// Canonicalize the key if it looks like an ActionID (not already canonical)
	canonicalKey := key
	if key != "" && !strings.HasPrefix(key, "@") && !strings.Contains(key, "/") {
		// Only try to resolve if we have a valid agent context (KDEPS_PATH set)
		if kdepsPath := os.Getenv("KDEPS_PATH"); kdepsPath != "" {
			agentReader, err := agent.GetGlobalAgentReader(nil, kdepsPath, "", "", nil)
			if err == nil && agentReader != nil {
				uri, err := url.Parse(fmt.Sprintf("agent:///%s", key))
				if err == nil {
					data, err := agentReader.Read(*uri)
					if err == nil && len(data) > 0 {
						canonicalKey = string(data)
						r.Logger.Debug("getRecord: resolved short ActionID", "key", key, "canonicalKey", canonicalKey, "agent", os.Getenv("KDEPS_CURRENT_AGENT"), "version", os.Getenv("KDEPS_CURRENT_VERSION"))
					} else {
						r.Logger.Debug("getRecord: failed to resolve short ActionID", "key", key, "agent", os.Getenv("KDEPS_CURRENT_AGENT"), "version", os.Getenv("KDEPS_CURRENT_VERSION"))
					}
				}
			}
		} else {
			r.Logger.Debug("getRecord: no agent reader available, using short ActionID as-is", "key", key, "agent", os.Getenv("KDEPS_CURRENT_AGENT"), "version", os.Getenv("KDEPS_CURRENT_VERSION"))
		}
	}

	// Ensure key is always an empty string, not NULL
	if canonicalKey == "" {
		canonicalKey = ""
	}

	r.Logger.Info("getRecord: querying for id, type, key", "graphID", r.GraphID, "id", id, "type", typ, "key", canonicalKey, "dbPath", r.DBPath)

	// Start a transaction for consistent read
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Debug("getRecord failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Debug("getRecord: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	var value string
	err = tx.QueryRow("SELECT value FROM records WHERE graph_id = ? AND id = ? AND type = ? AND key = ?", r.GraphID, id, typ, canonicalKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		// Commit the transaction even for no rows found
		if commitErr := tx.Commit(); commitErr != nil {
			r.Logger.Debug("getRecord failed to commit transaction", "error", commitErr)
			return nil, fmt.Errorf("failed to commit transaction: %w", commitErr)
		}
		committed = true
		r.Logger.Debug("getRecord: no record found", "id", id, "type", typ, "key", canonicalKey)
		return []byte(""), nil
	}
	if err != nil {
		r.Logger.Debug("getRecord failed to scan row", "error", err)
		return nil, fmt.Errorf("failed to read record: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Debug("getRecord failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	r.Logger.Debug("getRecord: retrieved value", "id", id, "type", typ, "key", canonicalKey, "contentLength", len(value))
	return []byte(value), nil
}

// deleteRecord removes a PKL record or key from the database
func (r *PklResourceReader) deleteRecord(id, typ, key string) ([]byte, error) {
	if id == "" || typ == "" {
		r.Logger.Debug("deleteRecord failed: id or type not provided")
		return nil, errors.New("delete operation requires id and type parameters")
	}

	r.Logger.Debug("deleteRecord processing", "id", id, "type", typ, "key", key)

	canonicalKey := key
	if key != "" && !strings.HasPrefix(key, "@") && !strings.Contains(key, "/") {
		if kdepsPath := os.Getenv("KDEPS_PATH"); kdepsPath != "" {
			agentReader, err := agent.GetGlobalAgentReader(nil, kdepsPath, "", "", nil)
			if err == nil && agentReader != nil {
				uri, err := url.Parse(fmt.Sprintf("agent:///%s", key))
				if err == nil {
					data, err := agentReader.Read(*uri)
					if err == nil && len(data) > 0 {
						canonicalKey = string(data)
						r.Logger.Debug("deleteRecord: resolved short ActionID", "key", key, "canonicalKey", canonicalKey)
					} else {
						r.Logger.Debug("deleteRecord: failed to resolve short ActionID, using as-is", "key", key)
					}
				}
			} else {
				r.Logger.Debug("deleteRecord: no agent reader available, using short ActionID as-is", "key", key)
			}
		}
	}

	// Start a transaction for atomic operation
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Debug("deleteRecord failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Debug("deleteRecord: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	result, err := tx.Exec("DELETE FROM records WHERE graph_id = ? AND id = ? AND type = ? AND key = ?", r.GraphID, id, typ, canonicalKey)
	if err != nil {
		r.Logger.Debug("deleteRecord failed to execute SQL", "error", err)
		return nil, fmt.Errorf("failed to delete record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.Logger.Debug("deleteRecord failed to check result", "error", err)
		return nil, fmt.Errorf("failed to check delete result: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Debug("deleteRecord failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("deleteRecord succeeded", "id", id, "type", typ, "key", canonicalKey, "removed", rowsAffected)
	return []byte(fmt.Sprintf("Deleted %d record(s)", rowsAffected)), nil
}

// clearRecords removes all records of a specific type or all records
func (r *PklResourceReader) clearRecords(typ string) ([]byte, error) {
	if typ == "" {
		r.Logger.Debug("clearRecords failed: type not provided")
		return nil, errors.New("clear operation requires type parameter")
	}

	r.Logger.Debug("clearRecords processing", "type", typ)

	// Start a transaction for atomic operation
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Debug("clearRecords failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Debug("clearRecords: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	var result sql.Result
	// Handle special case where type="_" means clear all records for this graphID
	if typ == "_" {
		result, err = tx.Exec("DELETE FROM records WHERE graph_id = ?", r.GraphID)
	} else {
		result, err = tx.Exec("DELETE FROM records WHERE graph_id = ? AND type = ?", r.GraphID, typ)
	}

	if err != nil {
		r.Logger.Debug("clearRecords failed to execute SQL", "error", err)
		return nil, fmt.Errorf("failed to clear records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.Logger.Debug("clearRecords failed to check result", "error", err)
		return nil, fmt.Errorf("failed to check clear result: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Debug("clearRecords failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("clearRecords succeeded", "type", typ, "removed", rowsAffected)
	return []byte(fmt.Sprintf("Cleared %d records", rowsAffected)), nil
}

// listRecords returns all record IDs of a specific type
func (r *PklResourceReader) listRecords(typ string) ([]byte, error) {
	if typ == "" {
		r.Logger.Debug("listRecords failed: type not provided")
		return nil, errors.New("list operation requires type parameter")
	}

	r.Logger.Debug("listRecords processing", "type", typ)

	// Start a transaction for consistent read
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Debug("listRecords failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Debug("listRecords: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	rows, err := tx.Query("SELECT DISTINCT id FROM records WHERE graph_id = ? AND type = ? ORDER BY id", r.GraphID, typ)
	if err != nil {
		r.Logger.Debug("listRecords failed to query records", "error", err)
		return nil, fmt.Errorf("failed to list records: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			r.Logger.Debug("listRecords failed to scan row", "error", err)
			return nil, fmt.Errorf("failed to scan record ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		r.Logger.Debug("listRecords failed during row iteration", "error", err)
		return nil, fmt.Errorf("failed to iterate records: %w", err)
	}

	// If no records, return []
	if len(ids) == 0 {
		return []byte("[]"), nil
	}

	result, err := json.Marshal(ids)
	if err != nil {
		r.Logger.Debug("listRecords failed to marshal JSON", "error", err)
		return nil, fmt.Errorf("failed to serialize record IDs: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Debug("listRecords failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("listRecords succeeded", "type", typ, "count", len(ids))
	return result, nil
}

// getResourceOutput retrieves the output file content for a specific resource
func (r *PklResourceReader) getResourceOutput(id, typ string) ([]byte, error) {
	r.Logger.Debug("getResourceOutput called", "id", id, "type", typ)
	if id == "" || typ == "" {
		return nil, errors.New("getResourceOutput requires id and type parameters")
	}

	// The request ID should be available in the resource ID or through the call context
	// For now, scan temp directory for matching files
	outputDir := "/tmp/action/files"

	// Look for files matching the pattern: {requestID}_{resourcename}
	// Convert canonical ID format: @whois2/llmResource:1.0.0 -> whois2_llmResource_1.0.0
	resourceName := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(id, "@", ""), "/", "_"), ":", "_")
	r.Logger.Debug("getResourceOutput: searching for pattern", "pattern", "*_"+resourceName, "directory", outputDir)

	files, err := filepath.Glob(filepath.Join(outputDir, "*_"+resourceName))
	if err != nil {
		r.Logger.Error("getResourceOutput: glob search failed", "error", err)
		return nil, fmt.Errorf("failed to search for output files: %w", err)
	}

	r.Logger.Debug("getResourceOutput: found matching files", "count", len(files), "files", files)
	if len(files) == 0 {
		return []byte(""), nil // Return empty content if file not found
	}

	// Use the most recent file if multiple matches
	var newestFile string
	var newestTime time.Time
	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			if info.ModTime().After(newestTime) {
				newestTime = info.ModTime()
				newestFile = file
			}
		}
	}

	if newestFile == "" {
		return []byte(""), nil
	}

	// Read the file content
	content, err := os.ReadFile(newestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file %s: %w", newestFile, err)
	}

	r.Logger.Debug("getResourceOutput: successfully read file", "file", newestFile, "contentLength", len(content))
	return content, nil
}

// evaluateResource retrieves the PKL record from database and evaluates it
func (r *PklResourceReader) evaluateResource(id, typ string) ([]byte, error) {
	r.Logger.Debug("evaluateResource called", "id", id, "type", typ)
	if id == "" || typ == "" {
		return nil, errors.New("evaluateResource requires id and type parameters")
	}

	// For LLM resources, try to get the output file first
	if typ == "llm" {
		outputData, err := r.getResourceOutput(id, typ)
		if err == nil && len(outputData) > 0 {
			r.Logger.Debug("evaluateResource: found LLM output file", "id", id, "dataLength", len(outputData))
			return outputData, nil
		}
	}

	// Get the PKL record from database
	pklRecord, err := r.getRecord(id, typ, "")
	if err != nil {
		r.Logger.Error("evaluateResource: failed to get PKL record", "error", err)
		return nil, fmt.Errorf("failed to get PKL record: %w", err)
	}

	if len(pklRecord) == 0 {
		r.Logger.Debug("evaluateResource: PKL record is empty", "id", id, "type", typ)
		return []byte(""), nil
	}

	r.Logger.Debug("evaluateResource: retrieved PKL record", "id", id, "type", typ, "recordLength", len(pklRecord))

	// Try to evaluate the PKL record using the PKL evaluator
	evaluator, err := pkl.NewEvaluator(context.Background(), pkl.PreconfiguredOptions)
	if err != nil {
		r.Logger.Error("evaluateResource: failed to create PKL evaluator", "error", err)
		return pklRecord, nil
	}
	defer evaluator.Close()

	// Create a temporary file for the PKL content
	tempFile, err := os.CreateTemp("", "pklres_eval_*.pkl")
	if err != nil {
		r.Logger.Error("evaluateResource: failed to create temp file", "error", err)
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write the PKL record to the temp file
	if _, err := tempFile.Write(pklRecord); err != nil {
		r.Logger.Error("evaluateResource: failed to write PKL record to temp file", "error", err)
		return nil, fmt.Errorf("failed to write PKL record: %w", err)
	}

	// Close the file so it can be read by the evaluator
	tempFile.Close()

	// Evaluate the PKL file
	result, err := evaluator.EvaluateOutputText(context.Background(), pkl.FileSource(tempFile.Name()))
	if err != nil {
		r.Logger.Error("evaluateResource: failed to evaluate PKL record", "error", err)
		// If evaluation fails, return the raw PKL record as fallback
		return pklRecord, nil
	}

	r.Logger.Debug("evaluateResource: successfully evaluated PKL record", "id", id, "type", typ, "resultLength", len(result))
	return []byte(result), nil
}

// InitializeDatabase sets up the SQLite database and creates the records table with retries.
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

		// Create records table with graphID, type and optional key support
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS records (
				graph_id TEXT NOT NULL,
				id TEXT NOT NULL,
				type TEXT NOT NULL,
				key TEXT DEFAULT '',
				value TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (graph_id, id, type, key)
			)
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create records table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Create indexes for better performance with graphID
		_, err = db.Exec(`
			CREATE INDEX IF NOT EXISTS idx_records_graph_type ON records(graph_id, type);
			CREATE INDEX IF NOT EXISTS idx_records_graph_id_type ON records(graph_id, id, type);
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create indexes after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializePklResource creates a new PklResourceReader with an initialized SQLite database.
func InitializePklResource(dbPath, graphID string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	return &PklResourceReader{DB: db, DBPath: dbPath, GraphID: graphID, Logger: slog.Default()}, nil
}
