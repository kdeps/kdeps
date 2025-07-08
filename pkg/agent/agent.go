package agent

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
)

// Global singleton instance
var (
	globalAgentReader *PklResourceReader
	globalAgentMutex  sync.RWMutex
	globalAgentOnce   sync.Once
)

// AgentInfo represents information about an installed agent
type AgentInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// PklResourceReader implements the pkl.ResourceReader interface for agent ID resolution.
type PklResourceReader struct {
	DB       *sql.DB
	DBPath   string
	Fs       afero.Fs
	KdepsDir string
	Logger   *logging.Logger
}

// Scheme returns the URI scheme for this reader.
func (r *PklResourceReader) Scheme() string {
	return "agent"
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

// Read handles agent ID resolution operations based on the URI.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	query := uri.Query()
	op := query.Get("op")

	// Default to "resolve" if no operation is specified
	if op == "" {
		op = "resolve"
	}

	switch op {
	case "resolve":
		return r.resolveAgentID(uri.Path, query)
	case "list":
		// If no path is provided, treat as list-installed
		if uri.Path == "" || uri.Path == "/" {
			return r.listInstalledAgents()
		}
		return r.listAgentResources(uri.Path, query)
	case "list-installed":
		return r.listInstalledAgents()
	case "register":
		return r.registerAgent(uri.Path, query)
	case "unregister":
		return r.unregisterAgent(uri.Path)
	default:
		return nil, fmt.Errorf("unknown operation: %s", op)
	}
}

// resolveAgentID resolves a local action ID to a fully qualified agent ID
func (r *PklResourceReader) resolveAgentID(actionID string, query url.Values) ([]byte, error) {
	actionID = strings.TrimPrefix(actionID, "/")

	if actionID == "" {
		return nil, errors.New("no action ID provided")
	}

	if !strings.HasPrefix(actionID, "@") {
		// Legacy/local form: use query params for agent and version
		currentAgentName := query.Get("agent")
		currentAgentVersion := query.Get("version")
		if currentAgentName == "" || currentAgentVersion == "" {
			return nil, errors.New("agent name and version required for ID resolution")
		}
		resolvedID := fmt.Sprintf("@%s/%s:%s", currentAgentName, actionID, currentAgentVersion)
		return []byte(resolvedID), nil
	}

	// Canonical @ form: support all four forms and latest version lookup
	// Parse forms:
	// @agentID
	// @agentID:version
	// @agentID/actionID
	// @agentID/actionID:version
	var agentID, actID, version string
	var hasAction bool

	id := strings.TrimPrefix(actionID, "@")
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		hasAction = true
		agentID = parts[0]
		actionPart := parts[1]
		actionParts := strings.SplitN(actionPart, ":", 2)
		actID = actionParts[0]
		if len(actionParts) == 2 {
			version = actionParts[1]
		}
	} else {
		agentParts := strings.SplitN(parts[0], ":", 2)
		agentID = agentParts[0]
		if len(agentParts) == 2 {
			version = agentParts[1]
		}
	}

	if agentID == "" {
		return nil, errors.New("agent ID required")
	}

	if version == "" {
		// Look up the latest version from the database
		latestVersion, err := r.findLatestVersionFromDB(agentID)
		if err != nil {
			return nil, fmt.Errorf("could not find any versions for agent %s: %w", agentID, err)
		}
		version = latestVersion
	}

	var resolvedID string
	if hasAction {
		resolvedID = fmt.Sprintf("@%s/%s:%s", agentID, actID, version)
	} else {
		resolvedID = fmt.Sprintf("@%s:%s", agentID, version)
	}
	return []byte(resolvedID), nil
}

// compareSemver returns 1 if a > b, -1 if a < b, 0 if equal (simple semver, not prerelease aware)
func compareSemver(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var ai, bi int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &ai)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bi)
		}
		if ai > bi {
			return 1
		} else if ai < bi {
			return -1
		}
	}
	return 0
}

// listInstalledAgents returns a list of all installed agents
func (r *PklResourceReader) listInstalledAgents() ([]byte, error) {
	agentsDir := filepath.Join(r.KdepsDir, "agents")

	agents := []AgentInfo{}

	// Walk through the agents directory
	err := afero.Walk(r.Fs, agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the root agents directory
		if !info.IsDir() || path == agentsDir {
			return nil
		}

		// Check if this is an agent directory (should contain workflow.pkl)
		workflowPath := filepath.Join(path, "workflow.pkl")
		if exists, _ := afero.Exists(r.Fs, workflowPath); !exists {
			return nil
		}

		// Extract agent name and version from path
		relPath, err := filepath.Rel(agentsDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) == 2 {
			agentInfo := AgentInfo{
				Name:    parts[0],
				Version: parts[1],
				Path:    path,
			}
			agents = append(agents, agentInfo)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan agents directory: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(agents)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agents to JSON: %w", err)
	}

	return jsonData, nil
}

// registerAgent registers a new agent in the database
func (r *PklResourceReader) registerAgent(agentID string, query url.Values) ([]byte, error) {
	agentID = strings.TrimPrefix(agentID, "/")
	if agentID == "" {
		return nil, errors.New("invalid URI: no agent ID provided")
	}

	agentPath := query.Get("path")
	if agentPath == "" {
		return nil, errors.New("agent path required for registration")
	}

	// Parse agent ID to extract agentID and version (ignore actionID)
	// Format: @agentID/actionID:<semver>
	parts := strings.Split(strings.TrimPrefix(agentID, "@"), "/")
	if len(parts) != 2 {
		return nil, errors.New("invalid agent ID format: expected @agentID/actionID:<semver>")
	}

	agentIDPart := parts[0]
	actionIDWithVersion := parts[1]

	// Split actionID and version
	actionParts := strings.Split(actionIDWithVersion, ":")
	if len(actionParts) != 2 {
		return nil, errors.New("invalid agent ID format: expected @agentID/actionID:<semver>")
	}

	version := actionParts[1]

	agentData := AgentInfo{
		Name:    agentIDPart,
		Version: version,
		Path:    agentPath,
	}

	jsonData, err := json.Marshal(agentData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent data: %w", err)
	}

	_, err = r.DB.Exec(
		"INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)",
		agentID, string(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	log.Printf("Registered agent: %s", agentID)
	return []byte(fmt.Sprintf("Registered agent: %s", agentID)), nil
}

// unregisterAgent removes an agent from the database
func (r *PklResourceReader) unregisterAgent(agentID string) ([]byte, error) {
	agentID = strings.TrimPrefix(agentID, "/")
	if agentID == "" {
		return nil, errors.New("invalid URI: no agent ID provided")
	}

	result, err := r.DB.Exec("DELETE FROM agents WHERE id = ?", agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to unregister agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check unregister result: %w", err)
	}

	log.Printf("Unregistered agent: %s (removed %d records)", agentID, rowsAffected)
	return []byte(fmt.Sprintf("Unregistered agent: %s", agentID)), nil
}

// listAgentResources returns a list of all resources from a specific agent
func (r *PklResourceReader) listAgentResources(agentPath string, query url.Values) ([]byte, error) {
	agentID := strings.TrimPrefix(agentPath, "/")
	if agentID == "" {
		return nil, errors.New("invalid URI: no agent ID provided")
	}

	// Get current agent context from query parameters
	currentAgentName := query.Get("agent")
	currentAgentVersion := query.Get("version")

	if currentAgentName == "" || currentAgentVersion == "" {
		return nil, errors.New("agent name and version required for resource listing")
	}

	// For now, return a special marker indicating all resources should be copied
	// In a full implementation, this would scan the agent's resources directory
	response := map[string]interface{}{
		"agent":     agentID,
		"operation": "copy_all_resources",
		"message":   fmt.Sprintf("All resources from agent %s will be copied", agentID),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	log.Printf("Listed all resources for agent: %s", agentID)
	return jsonData, nil
}

// InitializeDatabase sets up the SQLite database and creates the agents table with retries.
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

		// Create agents table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS agents (
				id TEXT PRIMARY KEY,
				data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			log.Printf("Attempt %d: Failed to create agents table: %v", attempt, err)
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create agents table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("SQLite database initialized successfully at %s on attempt %d", dbPath, attempt)
		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeAgent creates a new PklResourceReader with an in-memory SQLite database.
func InitializeAgent(fs afero.Fs, kdepsDir string, logger *logging.Logger) (*PklResourceReader, error) {
	// Use in-memory database for better performance
	db, err := InitializeDatabase(":memory:")
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}

	// Do NOT close db here; caller will manage closing
	reader := &PklResourceReader{
		DB:       db,
		DBPath:   ":memory:", // Indicate this is an in-memory database
		Fs:       fs,
		KdepsDir: kdepsDir,
		Logger:   logger,
	}

	// Register all agents and actions from the agents directory
	if err := reader.RegisterAllAgentsAndActions(); err != nil {
		logger.Warn("Failed to register all agents and actions", "error", err)
		// Don't fail initialization, just log a warning
	}

	return reader, nil
}

// Close is a helper method for cleanup that closes the database.
func (r *PklResourceReader) Close() error {
	var errs []error

	// Close the database
	if r.DB != nil {
		if err := r.DB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close database: %w", err))
		}
	}

	// Only remove file if it's not an in-memory database
	if r.DBPath != "" && r.DBPath != ":memory:" {
		if err := os.Remove(r.DBPath); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove temporary file %s: %w", r.DBPath, err))
		}
	}

	// Return the first error if any occurred
	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// RegisterAllAgentsAndActions scans the entire agents directory and registers all
// agentID/actionID/version combinations in the database for fast lookup.
func (r *PklResourceReader) RegisterAllAgentsAndActions() error {
	agentsDir := filepath.Join(r.KdepsDir, "agents")

	// Walk through the agents directory
	err := afero.Walk(r.Fs, agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or if it's the root agents directory
		if !info.IsDir() || path == agentsDir {
			return nil
		}

		// Check if this is an agent directory (should contain workflow.pkl)
		workflowPath := filepath.Join(path, "workflow.pkl")
		if exists, _ := afero.Exists(r.Fs, workflowPath); !exists {
			return nil
		}

		// Extract agent name and version from path
		relPath, err := filepath.Rel(agentsDir, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) != 2 {
			return nil // Skip if not agent/version structure
		}

		agentID := parts[0]
		version := parts[1]

		// Register the agent itself: @agentID:version
		agentOnlyID := fmt.Sprintf("@%s:%s", agentID, version)
		agentData := AgentInfo{
			Name:    agentID,
			Version: version,
			Path:    path,
		}
		jsonData, err := json.Marshal(agentData)
		if err != nil {
			return fmt.Errorf("failed to marshal agent data: %w", err)
		}
		_, err = r.DB.Exec(
			"INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)",
			agentOnlyID, string(jsonData),
		)
		if err != nil {
			return fmt.Errorf("failed to register agent %s: %w", agentOnlyID, err)
		}

		// Parse workflow.pkl to find all actionIDs
		actionIDs, err := r.extractActionIDsFromWorkflow(workflowPath)
		if err != nil {
			log.Printf("Warning: failed to extract actionIDs from %s: %v", workflowPath, err)
			return nil // Continue with other agents
		}

		// Register each actionID: @agentID/actionID:version
		for _, actionID := range actionIDs {
			actionIDWithVersion := fmt.Sprintf("@%s/%s:%s", agentID, actionID, version)
			actionData := AgentInfo{
				Name:    agentID,
				Version: version,
				Path:    path,
			}
			jsonData, err := json.Marshal(actionData)
			if err != nil {
				return fmt.Errorf("failed to marshal action data: %w", err)
			}
			_, err = r.DB.Exec(
				"INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)",
				actionIDWithVersion, string(jsonData),
			)
			if err != nil {
				return fmt.Errorf("failed to register action %s: %w", actionIDWithVersion, err)
			}
		}

		log.Printf("Registered agent %s v%s with %d actions", agentID, version, len(actionIDs))
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan agents directory: %w", err)
	}

	log.Printf("Completed registration of all agents and actions")
	return nil
}

// extractActionIDsFromWorkflow parses a workflow.pkl file to extract all actionIDs
func (r *PklResourceReader) extractActionIDsFromWorkflow(workflowPath string) ([]string, error) {
	content, err := afero.ReadFile(r.Fs, workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Parse the workflow.pkl content to find actionIDs
	// This is a simplified parser - in a real implementation, you might want to use PKL evaluation
	actionIDs := []string{}
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for ActionID = "..." patterns
		if strings.HasPrefix(line, "ActionID") && strings.Contains(line, "=") {
			// Extract the actionID value
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				// Remove quotes
				value = strings.Trim(value, `"'`)
				if value != "" {
					actionIDs = append(actionIDs, value)
				}
			}
		}
	}

	return actionIDs, nil
}

// findLatestVersionFromDB finds the latest version of an agent in the database
func (r *PklResourceReader) findLatestVersionFromDB(agentID string) (string, error) {
	// Query for all entries that start with @agentID:
	rows, err := r.DB.Query("SELECT id FROM agents WHERE id LIKE ?", fmt.Sprintf("@%s:%%", agentID))
	if err != nil {
		return "", fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()

	var latestVersion string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", fmt.Errorf("failed to scan database: %w", err)
		}

		// Extract version from the ID (format: @agentID:version)
		parts := strings.SplitN(id, ":", 2)
		if len(parts) == 2 {
			version := parts[1]
			if latestVersion == "" || compareSemver(version, latestVersion) > 0 {
				latestVersion = version
			}
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no versions found for agent %s", agentID)
	}

	return latestVersion, nil
}

// GetGlobalAgentReader returns the singleton agent reader instance.
// If it doesn't exist, it creates one with the provided parameters.
func GetGlobalAgentReader(fs afero.Fs, kdepsDir string, logger *logging.Logger) (*PklResourceReader, error) {
	globalAgentMutex.RLock()
	if globalAgentReader != nil {
		globalAgentMutex.RUnlock()
		return globalAgentReader, nil
	}
	globalAgentMutex.RUnlock()

	globalAgentMutex.Lock()
	defer globalAgentMutex.Unlock()

	// Double-check pattern
	if globalAgentReader != nil {
		return globalAgentReader, nil
	}

	// Create the singleton instance
	reader, err := InitializeAgent(fs, kdepsDir, logger)
	if err != nil {
		return nil, err
	}

	globalAgentReader = reader
	return globalAgentReader, nil
}

// CloseGlobalAgentReader closes the global agent reader and resets the singleton.
// This should be called during application shutdown.
func CloseGlobalAgentReader() error {
	globalAgentMutex.Lock()
	defer globalAgentMutex.Unlock()

	if globalAgentReader != nil {
		err := globalAgentReader.Close()
		globalAgentReader = nil
		return err
	}
	return nil
}
