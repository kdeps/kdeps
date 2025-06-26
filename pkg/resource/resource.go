package resource

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	_ "github.com/mattn/go-sqlite3"
)

// ResourceType defines the type of resource
type ResourceType string

const (
	TypeExec   ResourceType = "exec"
	TypePython ResourceType = "python"
	TypeHTTP   ResourceType = "http"
	TypeLLM    ResourceType = "llm"
	TypeData   ResourceType = "data"
)

// BaseResource represents common resource fields
type BaseResource struct {
	ID              string                 `json:"id"`
	RequestID       string                 `json:"request_id"`
	Type            ResourceType           `json:"type"`
	Timestamp       time.Time              `json:"timestamp"`
	TimeoutDuration *time.Duration         `json:"timeout_duration,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ExecResource represents execution resources
type ExecResource struct {
	BaseResource
	Command string            `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	Stdout  string            `json:"stdout,omitempty"`
	Stderr  string            `json:"stderr,omitempty"`
	File    string            `json:"file,omitempty"`
}

// PythonResource represents Python execution resources
type PythonResource struct {
	BaseResource
	Script string            `json:"script"`
	Env    map[string]string `json:"env,omitempty"`
	Stdout string            `json:"stdout,omitempty"`
	Stderr string            `json:"stderr,omitempty"`
	File   string            `json:"file,omitempty"`
}

// HTTPResource represents HTTP request resources
type HTTPResource struct {
	BaseResource
	Method   string            `json:"method"`
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers,omitempty"`
	Data     []string          `json:"data,omitempty"`
	Params   map[string]string `json:"params,omitempty"`
	Response *HTTPResponse     `json:"response,omitempty"`
	File     string            `json:"file,omitempty"`
}

// HTTPResponse represents HTTP response data
type HTTPResponse struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// LLMResource represents LLM/Chat resources
type LLMResource struct {
	BaseResource
	Model            string            `json:"model,omitempty"`
	Prompt           string            `json:"prompt,omitempty"`
	Role             string            `json:"role,omitempty"`
	JSONResponse     bool              `json:"json_response,omitempty"`
	JSONResponseKeys []string          `json:"json_response_keys,omitempty"`
	Scenario         string            `json:"scenario,omitempty"`
	Files            map[string]string `json:"files,omitempty"`
	Tools            []LLMTool         `json:"tools,omitempty"`
	Response         string            `json:"response,omitempty"`
	File             string            `json:"file,omitempty"`
}

// LLMTool represents LLM tool definition
type LLMTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// DataResource represents data file resources
type DataResource struct {
	BaseResource
	Files map[string]map[string]string `json:"files"` // [agentName][baseFilename] = filePath
}

// PklResourceReader implements the pkl.ResourceReader interface for SQLite resource storage
type PklResourceReader struct {
	DB        *sql.DB
	DBPath    string
	RequestID string
}

// Scheme returns the URI scheme for this reader
func (r *PklResourceReader) Scheme() string {
	return "resource"
}

// IsGlobbable indicates whether the reader supports globbing
func (r *PklResourceReader) IsGlobbable() bool {
	return false
}

// HasHierarchicalUris indicates whether URIs are hierarchical
func (r *PklResourceReader) HasHierarchicalUris() bool {
	return false
}

// ListElements is not used in this implementation
func (r *PklResourceReader) ListElements(_ url.URL) ([]pkl.PathElement, error) {
	return nil, nil
}

// Read handles resource operations based on URI
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	ctx := context.Background()
	log.Printf("Processing resource URI: %s", uri.String())

	// Parse the URI path to get resource ID and type
	path := strings.Trim(uri.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid URI path format: expected /type/id")
	}

	resourceType := ResourceType(parts[0])
	resourceID := parts[1]
	params := uri.Query()
	operation := params.Get("op")

	switch operation {
	case "get", "":
		return r.getResource(ctx, resourceType, resourceID)
	case "list":
		return r.listResources(ctx, resourceType)
	case "delete":
		return r.deleteResource(ctx, resourceType, resourceID)
	case "clear":
		return r.clearResources(ctx, resourceType)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}
}

// StoreExecResource stores an execution resource
func (r *PklResourceReader) StoreExecResource(resource *ExecResource) error {
	ctx := context.Background()
	return r.storeResource(ctx, resource)
}

// StorePythonResource stores a Python resource
func (r *PklResourceReader) StorePythonResource(resource *PythonResource) error {
	ctx := context.Background()
	return r.storeResource(ctx, resource)
}

// StoreHTTPResource stores an HTTP resource
func (r *PklResourceReader) StoreHTTPResource(resource *HTTPResource) error {
	ctx := context.Background()
	return r.storeResource(ctx, resource)
}

// StoreLLMResource stores an LLM resource
func (r *PklResourceReader) StoreLLMResource(resource *LLMResource) error {
	ctx := context.Background()
	return r.storeResource(ctx, resource)
}

// StoreDataResource stores a data resource
func (r *PklResourceReader) StoreDataResource(resource *DataResource) error {
	ctx := context.Background()
	return r.storeResource(ctx, resource)
}

// storeResource stores any resource type
func (r *PklResourceReader) storeResource(ctx context.Context, resource interface{}) error {
	var baseRes *BaseResource
	var resourceData []byte
	var err error

	// Extract base resource and serialize specific data
	switch res := resource.(type) {
	case *ExecResource:
		baseRes = &res.BaseResource
		resourceData, err = json.Marshal(res)
	case *PythonResource:
		baseRes = &res.BaseResource
		resourceData, err = json.Marshal(res)
	case *HTTPResource:
		baseRes = &res.BaseResource
		resourceData, err = json.Marshal(res)
	case *LLMResource:
		baseRes = &res.BaseResource
		resourceData, err = json.Marshal(res)
	case *DataResource:
		baseRes = &res.BaseResource
		resourceData, err = json.Marshal(res)
	default:
		return fmt.Errorf("unsupported resource type: %T", resource)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal resource: %w", err)
	}

	now := time.Now()
	baseRes.CreatedAt = now
	baseRes.UpdatedAt = now

	query := `
		INSERT OR REPLACE INTO resources (
			id, request_id, type, timestamp, timeout_duration, 
			metadata, resource_data, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	var timeoutDurationMillis *int64
	if baseRes.TimeoutDuration != nil {
		millis := baseRes.TimeoutDuration.Milliseconds()
		timeoutDurationMillis = &millis
	}

	metadataJSON, err := json.Marshal(baseRes.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, query,
		baseRes.ID,
		baseRes.RequestID,
		string(baseRes.Type),
		baseRes.Timestamp.UnixNano(),
		timeoutDurationMillis,
		metadataJSON,
		resourceData,
		now.UnixNano(),
		now.UnixNano(),
	)

	if err != nil {
		log.Printf("Failed to store resource %s: %v", baseRes.ID, err)
		return fmt.Errorf("failed to store resource: %w", err)
	}

	log.Printf("Successfully stored %s resource: %s", baseRes.Type, baseRes.ID)
	return nil
}

// getResource retrieves a specific resource
func (r *PklResourceReader) getResource(ctx context.Context, resourceType ResourceType, resourceID string) ([]byte, error) {
	query := `
		SELECT resource_data FROM resources 
		WHERE id = ? AND type = ? AND request_id = ?`

	var resourceData []byte
	err := r.DB.QueryRowContext(ctx, query, resourceID, string(resourceType), r.RequestID).Scan(&resourceData)
	if err == sql.ErrNoRows {
		log.Printf("Resource not found: type=%s, id=%s, requestID=%s", resourceType, resourceID, r.RequestID)
		return []byte("{}"), nil
	}
	if err != nil {
		log.Printf("Failed to get resource %s/%s: %v", resourceType, resourceID, err)
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	log.Printf("Successfully retrieved %s resource: %s", resourceType, resourceID)
	return resourceData, nil
}

// listResources retrieves all resources of a specific type
func (r *PklResourceReader) listResources(ctx context.Context, resourceType ResourceType) ([]byte, error) {
	query := `
		SELECT id, resource_data FROM resources 
		WHERE type = ? AND request_id = ? 
		ORDER BY created_at ASC`

	rows, err := r.DB.QueryContext(ctx, query, string(resourceType), r.RequestID)
	if err != nil {
		log.Printf("Failed to list resources of type %s: %v", resourceType, err)
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	defer rows.Close()

	resources := make(map[string]json.RawMessage)
	for rows.Next() {
		var id string
		var resourceData []byte
		if err := rows.Scan(&id, &resourceData); err != nil {
			log.Printf("Failed to scan resource row: %v", err)
			return nil, fmt.Errorf("failed to scan resource: %w", err)
		}
		resources[id] = json.RawMessage(resourceData)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating resource rows: %v", err)
		return nil, fmt.Errorf("error iterating resources: %w", err)
	}

	result, err := json.Marshal(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource list: %w", err)
	}

	log.Printf("Successfully listed %d resources of type %s", len(resources), resourceType)
	return result, nil
}

// deleteResource deletes a specific resource
func (r *PklResourceReader) deleteResource(ctx context.Context, resourceType ResourceType, resourceID string) ([]byte, error) {
	query := `DELETE FROM resources WHERE id = ? AND type = ? AND request_id = ?`

	result, err := r.DB.ExecContext(ctx, query, resourceID, string(resourceType), r.RequestID)
	if err != nil {
		log.Printf("Failed to delete resource %s/%s: %v", resourceType, resourceID, err)
		return nil, fmt.Errorf("failed to delete resource: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	message := fmt.Sprintf("Deleted %d resource(s)", rowsAffected)
	log.Printf("Successfully deleted %s resource %s: %d rows affected", resourceType, resourceID, rowsAffected)
	return []byte(message), nil
}

// clearResources deletes all resources of a specific type
func (r *PklResourceReader) clearResources(ctx context.Context, resourceType ResourceType) ([]byte, error) {
	query := `DELETE FROM resources WHERE type = ? AND request_id = ?`

	result, err := r.DB.ExecContext(ctx, query, string(resourceType), r.RequestID)
	if err != nil {
		log.Printf("Failed to clear resources of type %s: %v", resourceType, err)
		return nil, fmt.Errorf("failed to clear resources: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	message := fmt.Sprintf("Cleared %d resource(s) of type %s", rowsAffected, resourceType)
	log.Printf("Successfully cleared %d resources of type %s", rowsAffected, resourceType)
	return []byte(message), nil
}

// InitializeDatabase creates the resources table
func InitializeDatabase(dbPath string) (*sql.DB, error) {
	const maxAttempts = 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Attempt %d: Initializing resource database at %s", attempt, dbPath)

		db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
		if err != nil {
			log.Printf("Attempt %d: Failed to open database: %v", attempt, err)
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		if err := db.Ping(); err != nil {
			log.Printf("Attempt %d: Failed to ping database: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Create resources table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS resources (
				id TEXT NOT NULL,
				request_id TEXT NOT NULL,
				type TEXT NOT NULL,
				timestamp INTEGER NOT NULL,
				timeout_duration INTEGER NULL,
				metadata TEXT NOT NULL DEFAULT '{}',
				resource_data TEXT NOT NULL,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				PRIMARY KEY (id, request_id, type)
			)
		`)
		if err != nil {
			log.Printf("Attempt %d: Failed to create resources table: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create resources table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Create indexes for performance
		indexes := []string{
			"CREATE INDEX IF NOT EXISTS idx_resources_request_type ON resources(request_id, type)",
			"CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type)",
			"CREATE INDEX IF NOT EXISTS idx_resources_timestamp ON resources(timestamp)",
			"CREATE INDEX IF NOT EXISTS idx_resources_created_at ON resources(created_at)",
		}

		for _, indexSQL := range indexes {
			if _, err := db.Exec(indexSQL); err != nil {
				log.Printf("Attempt %d: Failed to create index: %v", attempt, err)
				db.Close()
				if attempt == maxAttempts {
					return nil, fmt.Errorf("failed to create index after %d attempts: %w", maxAttempts, err)
				}
				time.Sleep(1 * time.Second)
				continue
			}
		}

		log.Printf("Resource database initialized successfully at %s on attempt %d", dbPath, attempt)
		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeResource creates a new PklResourceReader with an initialized SQLite database
func InitializeResource(dbPath string, requestID string) (*PklResourceReader, error) {
	db, err := InitializeDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	return &PklResourceReader{DB: db, DBPath: dbPath, RequestID: requestID}, nil
}

// Close closes the database connection
func (r *PklResourceReader) Close() error {
	if r.DB != nil {
		err := r.DB.Close()
		r.DB = nil
		if err != nil {
			log.Printf("Error closing resource database: %v", err)
			return fmt.Errorf("failed to close database: %w", err)
		}
		log.Printf("Resource database closed successfully")
	}
	return nil
}
