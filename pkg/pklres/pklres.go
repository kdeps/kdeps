package pklres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	_ "github.com/mattn/go-sqlite3" // SQLite driver registration
	"github.com/spf13/afero"
)

var (
	globalPklresReader *PklResourceReader
	globalMutex        sync.RWMutex
)

// getSchemaFileForType returns the appropriate schema file based on the resource type
func getSchemaFileForType(typ string) string {
	switch typ {
	case "python":
		return "Python.pkl"
	case "llm":
		return "LLM.pkl"
	case "http":
		return "HTTP.pkl"
	case "exec":
		return "Exec.pkl"
	case "data":
		return "Data.pkl"
	default:
		// Default to Resource.pkl for unknown types
		return "Resource.pkl"
	}
}

// PklResourceReader implements the pkl.ResourceReader interface for SQLite.
type PklResourceReader struct {
	DB             *sql.DB
	DBPath         string // Store dbPath for reinitialization
	Fs             afero.Fs
	Logger         *logging.Logger
	GraphID        string // Current graphID for scoping database operations
	CurrentAgent   string // Current agent name for ActionID resolution
	CurrentVersion string // Current agent version for ActionID resolution
	KdepsPath      string // Path to kdeps directory for agent reader

	// Synchronization fields
	ProcessingStatus map[string]ProcessingStatus // Track processing status per resource
	StatusMutex      sync.RWMutex                // Protect processing status
}

// ProcessingStatus tracks the processing state of a resource
type ProcessingStatus struct {
	IsProcessing bool      // Whether the resource is currently being processed
	IsFinished   bool      // Whether the resource processing is finished
	FinishedAt   time.Time // When the processing finished
	Dependencies []string  // List of dependencies that must be finished
}

// NewProcessingStatus creates a new processing status
func NewProcessingStatus(dependencies []string) ProcessingStatus {
	return ProcessingStatus{
		IsProcessing: true,
		IsFinished:   false,
		FinishedAt:   time.Time{},
		Dependencies: dependencies,
	}
}

// SetProcessingStatus sets the processing status for a resource
func (r *PklResourceReader) SetProcessingStatus(resourceID string, status ProcessingStatus) {
	r.StatusMutex.Lock()
	defer r.StatusMutex.Unlock()
	r.ProcessingStatus[resourceID] = status
	r.Logger.Debug("set processing status", "resourceID", resourceID, "isProcessing", status.IsProcessing, "isFinished", status.IsFinished, "dependencies", status.Dependencies)
}

// GetProcessingStatus gets the processing status for a resource
func (r *PklResourceReader) GetProcessingStatus(resourceID string) (ProcessingStatus, bool) {
	r.StatusMutex.RLock()
	defer r.StatusMutex.RUnlock()
	status, exists := r.ProcessingStatus[resourceID]
	return status, exists
}

// MarkResourceFinished marks a resource as finished processing
func (r *PklResourceReader) MarkResourceFinished(resourceID string) {
	r.StatusMutex.Lock()
	defer r.StatusMutex.Unlock()
	if status, exists := r.ProcessingStatus[resourceID]; exists {
		status.IsProcessing = false
		status.IsFinished = true
		status.FinishedAt = time.Now()
		r.ProcessingStatus[resourceID] = status
		r.Logger.Debug("marked resource as finished", "resourceID", resourceID)
	}
}

// CheckDependenciesFinished checks if all dependencies for a resource are finished
func (r *PklResourceReader) CheckDependenciesFinished(resourceID string) bool {
	r.StatusMutex.RLock()
	defer r.StatusMutex.RUnlock()

	status, exists := r.ProcessingStatus[resourceID]
	if !exists {
		return true // No status means no dependencies to check
	}

	for _, depID := range status.Dependencies {
		depStatus, depExists := r.ProcessingStatus[depID]
		if !depExists || !depStatus.IsFinished {
			return false
		}
	}
	return true
}

// WaitForDependencies waits for all dependencies to be finished
func (r *PklResourceReader) WaitForDependencies(resourceID string, timeout time.Duration) error {
	start := time.Now()
	for {
		if r.CheckDependenciesFinished(resourceID) {
			return nil
		}

		if timeout > 0 && time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for dependencies for resource %s", resourceID)
		}

		time.Sleep(100 * time.Millisecond)
	}
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

// Read retrieves, sets, or lists PKL records in the SQLite database based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	r.Logger.Debug("PklResourceReader.Read called", "uri", uri.String())
	// Check if receiver is nil and try to use global reader
	if r == nil {
		globalReader := GetGlobalPklresReader()
		if globalReader != nil {
			r = globalReader
		} else {
			newReader, err := InitializePklResource(":memory:", "default", "", "", "")
			if err != nil {
				return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
			}
			r = newReader
		}
	}

	// For global reader, never create new database connections - use existing one
	globalReader := GetGlobalPklresReader()
	if r == globalReader {
		if r.DB == nil {
			return nil, errors.New("global pklres reader database is nil")
		}
		r.Logger.Debug("using global pklres reader database", "dbPath", r.DBPath, "graphID", r.GraphID)

		// Verify table exists
		var tableName string
		err := r.DB.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='records'").Scan(&tableName)
		if err != nil {
			r.Logger.Debug("records table check failed", "error", err)
			// Table doesn't exist, create it
			if _, err := r.DB.Exec(`CREATE TABLE IF NOT EXISTS records (
				graph_id TEXT NOT NULL,
				id TEXT NOT NULL,
				type TEXT NOT NULL,
				key TEXT DEFAULT '',
				value TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (graph_id, id, type, key)
			)`); err != nil {
				return nil, fmt.Errorf("failed to create records table: %w", err)
			}
			r.Logger.Debug("created records table in global database")
		} else {
			r.Logger.Debug("records table exists in global database")
		}
	} else {
		// Check if db is nil or closed and initialize with retries (for non-global readers)
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
	}

	id := strings.TrimPrefix(uri.Path, "/")
	query := uri.Query()
	operation := query.Get("op")
	typ := query.Get("type")
	key := query.Get("key")

	switch operation {
	case "set":
		return r.setRecord(id, typ, key, query)
	case "list":
		return r.listRecords(typ)
	case "get":
		return r.getRecord(id, typ, key)
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

	// Check if this is a resource record that needs synchronization
	if key != "" {
		// For resource records, check if the resource is finished processing
		status, exists := r.GetProcessingStatus(id)
		if exists && status.IsProcessing && !status.IsFinished {
			r.Logger.Error("setRecord failed: resource is still processing", "id", id, "type", typ)
			return nil, fmt.Errorf("cannot set record for resource %s: resource is still processing", id)
		}
	}

	// If this is a resource record (key is not empty), we need to handle it as a mapping update
	if key != "" {
		return r.setResourceRecord(id, typ, key, value)
	}

	// For non-resource records (empty key), store the value directly
	return r.setDirectRecord(id, typ, value)
}

// setResourceRecord handles storing resource records in PKL mappings
func (r *PklResourceReader) setResourceRecord(id, typ, key, value string) ([]byte, error) {
	// Get existing mapping content
	existingContent, err := r.getRecord(id, typ, "")
	if err != nil {
		r.Logger.Debug("setResourceRecord: no existing content, creating new mapping", "id", id, "type", typ)
		existingContent = []byte("")
	}

	// Parse existing content or create new mapping structure
	var mappingContent string
	if len(existingContent) > 0 {
		mappingContent = string(existingContent)
	} else {
		// Create new mapping structure based on resource type
		blockName := "Resources"
		schemaFile := getSchemaFileForType(typ)
		if typ == "data" {
			blockName = "Files"
		}
		// Use proper schema import path with dynamic version
		schemaPath := schema.ImportPath(context.Background(), schemaFile)
		mappingContent = fmt.Sprintf("extends \"%s\"\n\n%s = new Mapping {}\n", schemaPath, blockName)
	}

	// Update the mapping with the new record
	updatedContent, err := r.updateMappingContent(mappingContent, key, value, typ)
	if err != nil {
		r.Logger.Error("setResourceRecord failed to update mapping", "error", err)
		return nil, fmt.Errorf("failed to update mapping: %w", err)
	}

	// Store the updated content
	query := url.Values{}
	query.Set("value", updatedContent)
	return r.setDirectRecord(id, typ, updatedContent)
}

// setDirectRecord stores a direct value in the database
func (r *PklResourceReader) setDirectRecord(id, typ, value string) ([]byte, error) {
	// Ensure table exists before attempting to set record
	if _, err := r.DB.Exec(`CREATE TABLE IF NOT EXISTS records (
		graph_id TEXT NOT NULL,
		id TEXT NOT NULL,
		type TEXT NOT NULL,
		key TEXT DEFAULT '',
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (graph_id, id, type, key)
	)`); err != nil {
		r.Logger.Error("setDirectRecord failed to create table", "error", err)
		return nil, fmt.Errorf("failed to create records table: %w", err)
	}

	// Start a transaction for atomic operation
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Error("setDirectRecord failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Error("setDirectRecord: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	// Use INSERT OR REPLACE with graphID and consistent key handling
	_, err = tx.Exec(
		"INSERT OR REPLACE INTO records (graph_id, id, type, key, value) VALUES (?, ?, ?, ?, ?)",
		r.GraphID, id, typ, "", value,
	)
	if err != nil {
		r.Logger.Error("setDirectRecord failed to execute SQL", "error", err)
		return nil, fmt.Errorf("failed to set record: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Error("setDirectRecord failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("setDirectRecord succeeded", "id", id, "type", typ, "value", value)
	return []byte(value), nil
}

// updateMappingContent updates a PKL mapping with a new key-value pair
func (r *PklResourceReader) updateMappingContent(content, key, value, typ string) (string, error) {
	// Find the Resources or Files block
	blockName := "Resources"
	if typ == "data" {
		blockName = "Files"
	}

	// Look for either "BlockName {" or "BlockName = new Mapping {"
	blockPattern1 := blockName + " {"
	blockPattern2 := blockName + " = new Mapping {"

	blockIndex := strings.Index(content, blockPattern1)
	patternUsed := blockPattern1
	if blockIndex == -1 {
		blockIndex = strings.Index(content, blockPattern2)
		patternUsed = blockPattern2
	}

	if blockIndex == -1 {
		return "", errors.New(blockName + " block not found in content")
	}

	// Find the closing brace of the block
	braceCount := 0
	closingBraceIndex := -1
	for i := blockIndex; i < len(content); i++ {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
			if braceCount == 0 {
				closingBraceIndex = i
				break
			}
		}
	}
	if closingBraceIndex == -1 {
		return "", errors.New(blockName + " block not properly closed")
	}

	// Parse existing entries into a map
	blockBody := content[blockIndex+len(patternUsed) : closingBraceIndex]
	lines := strings.Split(blockBody, "\n")
	entries := map[string]string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "[") {
			continue
		}
		// Parse ["key"] = value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if strings.HasPrefix(k, "[") && strings.HasSuffix(k, "]") {
			k = strings.TrimPrefix(k, "[")
			k = strings.TrimSuffix(k, "]")
			k = strings.Trim(k, "\"")
			entries[k] = v
		}
	}
	// Add or update the entry
	entries[key] = value

	// Write back the collection block, sorted by key
	var newBlock strings.Builder
	newBlock.WriteString(blockName + " = new Mapping {\n")
	var keys []string
	for k := range entries {
		keys = append(keys, k)
	}
	// Sort keys for consistent output
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		newBlock.WriteString(fmt.Sprintf("  [\"%s\"] = %s\n", k, entries[k]))
	}
	newBlock.WriteString("}\n")

	// Replace the old block in the content
	before := content[:blockIndex]
	after := content[closingBraceIndex+1:]
	return before + newBlock.String() + after, nil
}

// getRecord retrieves a PKL record from the database
func (r *PklResourceReader) getRecord(id, typ, key string) ([]byte, error) {
	if id == "" || typ == "" {
		return nil, errors.New("get operation requires id and type parameters")
	}

	// Canonicalize the id if it's not already canonical
	canonicalID := id
	r.Logger.Debug("getRecord: ActionID resolution", "id", id, "agent", r.CurrentAgent, "version", r.CurrentVersion, "kdepsPath", r.KdepsPath)
	if r.CurrentAgent != "" && r.CurrentVersion != "" && r.KdepsPath != "" {
		// Use agent reader to resolve the action ID
		agentReader, err := agent.GetGlobalAgentReader(r.Fs, r.KdepsPath, r.CurrentAgent, r.CurrentVersion, r.Logger)
		if err == nil {
			// Create URI for agent ID resolution
			query := url.Values{}
			query.Set("op", "resolve")
			query.Set("agent", r.CurrentAgent)
			query.Set("version", r.CurrentVersion)
			uri := url.URL{
				Scheme:   "agent",
				Path:     "/" + id,
				RawQuery: query.Encode(),
			}

			resolvedIDBytes, err := agentReader.Read(uri)
			if err == nil {
				canonicalID = string(resolvedIDBytes)
				r.Logger.Debug("getRecord: resolved ActionID", "original", id, "canonical", canonicalID)
			} else {
				r.Logger.Debug("getRecord: failed to resolve ActionID, using original", "id", id, "error", err)
			}
		} else {
			r.Logger.Debug("getRecord: failed to get agent reader, using original ID", "id", id, "error", err)
		}
	}

	// Check if this is a resource record that needs dependency synchronization
	if key != "" {
		// For resource records, check if dependencies are finished
		if !r.CheckDependenciesFinished(canonicalID) {
			r.Logger.Info("getRecord: dependencies not finished, waiting", "id", canonicalID, "type", typ)
			if err := r.WaitForDependencies(canonicalID, 5*time.Minute); err != nil {
				return nil, err
			}
		}
		// For resource records, also wait for the resource itself to be finished
		waitStart := time.Now()
		for {
			status, exists := r.GetProcessingStatus(canonicalID)
			if exists && status.IsFinished {
				break
			}
			if time.Since(waitStart) > 5*time.Minute {
				return nil, fmt.Errorf("timeout waiting for resource %s to finish", canonicalID)
			}
			r.Logger.Info("getRecord: resource not finished, waiting", "id", canonicalID, "type", typ)
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Ensure table exists before attempting to get record
	if _, err := r.DB.Exec(`CREATE TABLE IF NOT EXISTS records (
		graph_id TEXT NOT NULL,
		id TEXT NOT NULL,
		type TEXT NOT NULL,
		key TEXT DEFAULT '',
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (graph_id, id, type, key)
	)`); err != nil {
		r.Logger.Error("getRecord failed to create table", "error", err)
		return nil, fmt.Errorf("failed to create records table: %w", err)
	}

	// Start a transaction for consistent read
	tx, err := r.DB.Begin()
	if err != nil {
		r.Logger.Error("getRecord failed to start transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.Logger.Error("getRecord: failed to rollback transaction", "error", rollbackErr)
			}
		}
	}()

	// Query for the record
	var value string
	err = tx.QueryRow("SELECT value FROM records WHERE graph_id = ? AND id = ? AND type = ? AND key = ?",
		r.GraphID, canonicalID, typ, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Logger.Debug("getRecord: no record found", "id", canonicalID, "type", typ, "key", key)
			return []byte(""), nil // Return empty byte slice for no record found
		}
		r.Logger.Error("getRecord failed to query record", "error", err)
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		r.Logger.Error("getRecord failed to commit transaction", "error", err)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.Logger.Debug("getRecord succeeded", "id", canonicalID, "type", typ, "key", key, "value", value)
	return []byte(value), nil
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

// SetGlobalPklresReader sets the global pklres reader instance
func SetGlobalPklresReader(reader *PklResourceReader) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalPklresReader = reader
}

// GetGlobalPklresReader returns the global pklres reader instance
func GetGlobalPklresReader() *PklResourceReader {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalPklresReader
}

// UpdateGlobalPklresReaderContext updates the global pklres reader with new context
func UpdateGlobalPklresReaderContext(graphID, currentAgent, currentVersion, kdepsPath string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalPklresReader == nil {
		return errors.New("global pklres reader not initialized")
	}

	globalPklresReader.GraphID = graphID
	globalPklresReader.CurrentAgent = currentAgent
	globalPklresReader.CurrentVersion = currentVersion
	globalPklresReader.KdepsPath = kdepsPath

	return nil
}

// InitializePklResource creates a new PklResourceReader with an initialized SQLite database.
func InitializePklResource(dbPath, graphID, currentAgent, currentVersion, kdepsPath string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	// Do NOT close db here; caller will manage closing
	reader := &PklResourceReader{
		DB:               db,
		DBPath:           dbPath,
		GraphID:          graphID,
		CurrentAgent:     currentAgent,
		CurrentVersion:   currentVersion,
		KdepsPath:        kdepsPath,
		Logger:           logging.GetLogger(),
		ProcessingStatus: make(map[string]ProcessingStatus),
		StatusMutex:      sync.RWMutex{},
	}

	// Set as global reader if none exists
	if GetGlobalPklresReader() == nil {
		SetGlobalPklresReader(reader)
	}

	return reader, nil
}
