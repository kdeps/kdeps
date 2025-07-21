package agent

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"

	// Blank import for SQLite driver registration
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
)

// Simplified database connection pool for agent reader
var (
	agentDBPool      *sql.DB
	agentDBPoolMutex sync.Mutex
)

// Info holds agent information.
type Info struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// PklResourceReader implements the pkl.ResourceReader interface for agent ID resolution.
type PklResourceReader struct {
	DB             *sql.DB
	DBPath         string
	Fs             afero.Fs
	KdepsDir       string
	Logger         *logging.Logger
	CurrentAgent   string // Current agent name for context
	CurrentVersion string // Current agent version for context
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
	r.Logger.Debug("AgentReader.Read called", "uri", uri.String())
	query := uri.Query()
	op := query.Get("op")

	// Default to "resolve" if no operation is specified
	if op == "" {
		op = "resolve"
	}

	switch op {
	case "resolve":
		resolvedID, err := r.resolveAgentID(uri.Path, query)
		if err != nil {
			return nil, err
		}
		r.Logger.Debug("AgentReader.Read resolved", "from", uri.String(), "to", string(resolvedID))
		return resolvedID, nil
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
		// Legacy/local form: use query params for agent and version, or fall back to current context
		currentAgentName := query.Get("agent")
		currentAgentVersion := query.Get("version")

		// If query params are not provided, use the current agent context
		if currentAgentName == "" {
			currentAgentName = r.CurrentAgent
		}
		if currentAgentVersion == "" {
			currentAgentVersion = r.CurrentVersion
		}

		// If still not available, try environment variables as a last resort
		if currentAgentName == "" {
			currentAgentName = os.Getenv("KDEPS_CURRENT_AGENT")
		}
		if currentAgentVersion == "" {
			currentAgentVersion = os.Getenv("KDEPS_CURRENT_VERSION")
		}

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
			_, _ = fmt.Sscanf(aParts[i], "%d", &ai)
		}
		if i < len(bParts) {
			_, _ = fmt.Sscanf(bParts[i], "%d", &bi)
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

	agents := []Info{}

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
			agentInfo := Info{
				ID:      parts[0],
				Version: parts[1],
				Commit:  "", // No commit info in this simple list
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

	actionIDWithVersion := parts[1]

	// Extract agent ID and version from the path
	actionParts := strings.Split(actionIDWithVersion, ":")
	if len(actionParts) != 2 {
		return nil, errors.New("invalid agent ID format: expected @agentID/actionID:<semver>")
	}
	version := actionParts[1]

	agentData := Info{
		ID:      agentID,
		Version: version,
		Commit:  "", // No commit info in this simple register
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

	r.Logger.Debug("Registered agent", "agent_id", agentID)
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

	r.Logger.Debug("Unregistered agent", "agent_id", agentID, "rows_affected", rowsAffected)
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

	r.Logger.Debug("Listed all resources for agent", "agent_id", agentID)
	return jsonData, nil
}

// InitializeDatabase sets up the SQLite database and creates the agents table with retries.
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

		// Create agents table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS agents (
				id TEXT PRIMARY KEY,
				data TEXT NOT NULL,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			db.Close()
			if attempt == maxAttempts {
				return nil, fmt.Errorf("failed to create agents table after %d attempts: %w", maxAttempts, err)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		return db, nil
	}
	return nil, fmt.Errorf("failed to initialize database after %d attempts", maxAttempts)
}

// InitializeAgent creates a new PklResourceReader with an in-memory SQLite database.
func InitializeAgent(fs afero.Fs, kdepsDir string, currentAgent string, currentVersion string, logger *logging.Logger) (*PklResourceReader, error) {
	// For now, create a new database for each reader to avoid test issues
	// TODO: Optimize this to use a shared pool in production
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Create the agents table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating agents table: %w", err)
	}

	reader := &PklResourceReader{
		DB:             db,
		DBPath:         ":memory:", // Indicate this is an in-memory database
		Fs:             fs,
		KdepsDir:       kdepsDir,
		CurrentAgent:   currentAgent,
		CurrentVersion: currentVersion,
		Logger:         logger,
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

// RegisterAllAgentsAndActions scans the entire agents directory and /agent/project/resources/ and registers all
// agentID/actionID/version combinations in the database for fast lookup.
func (r *PklResourceReader) RegisterAllAgentsAndActions() error {
	// Check if filesystem is available
	if r.Fs == nil {
		r.Logger.Debug("filesystem not available, skipping agent registration")
		return nil
	}

	var actionID string

	agentsDir := filepath.Join(r.KdepsDir, "agents")
	if _, err := r.Fs.Stat(agentsDir); os.IsNotExist(err) {
		r.Logger.Debug("agents directory does not exist", "path", agentsDir)
		// Not an error, just no agents to register
	} else {
		// First scan for .kdeps files in the base agents directory
		agentFiles, err := afero.ReadDir(r.Fs, agentsDir)
		if err == nil {
			for _, file := range agentFiles {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".kdeps") {
					// Extract agent name from filename (remove .kdeps extension)
					agentName := strings.TrimSuffix(file.Name(), ".kdeps")
					r.Logger.Debug("found agent kdeps file", "agent", agentName, "file", file.Name())

					// Register this agent - we'll use default version 1.0.0 for kdeps files
					agentID := fmt.Sprintf("@%s:1.0.0", agentName)
					agentData := map[string]interface{}{
						"name":    agentName,
						"version": "1.0.0",
						"path":    filepath.Join(agentsDir, file.Name()),
					}
					agentJSON, _ := json.Marshal(agentData)
					_, err := r.DB.Exec("INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)", agentID, string(agentJSON))
					if err != nil {
						r.Logger.Debug("failed to register agent from kdeps file", "agent_id", agentID, "error", err)
					}
				}
			}
		}

		// Then scan agent directory structure
		afero.Walk(r.Fs, agentsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip if not a directory or if it's the agents directory itself
			if !info.IsDir() || path == agentsDir {
				return nil
			}

			// Extract agent name from path
			agentName := filepath.Base(path)
			if agentName == "agents" {
				return nil // Skip the agents directory itself
			}

			// Look for version directories
			versionDirs, err := afero.ReadDir(r.Fs, path)
			if err != nil {
				r.Logger.Debug("failed to read agent directory", "agent", agentName, "error", err)
				return nil // Continue with other agents
			}

			for _, versionDir := range versionDirs {
				if !versionDir.IsDir() {
					continue
				}

				version := versionDir.Name()
				agentID := fmt.Sprintf("@%s:%s", agentName, version)

				// Look for workflow.pkl file
				workflowPath := filepath.Join(path, version, "workflow.pkl")
				if _, err := r.Fs.Stat(workflowPath); os.IsNotExist(err) {
					continue // No workflow file, skip this version
				}

				// Extract action IDs from workflow
				actionIDs, err := r.extractActionIDsFromWorkflow(workflowPath)
				if err != nil {
					r.Logger.Debug("failed to extract actionIDs from workflow", "path", workflowPath, "error", err)
					continue // Continue with other agents
				}

				// Register the agent itself
				agentData := map[string]interface{}{
					"name":    agentName,
					"version": version,
					"path":    filepath.Join(path, version),
				}
				agentJSON, _ := json.Marshal(agentData)
				_, err = r.DB.Exec("INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)", agentID, string(agentJSON))
				if err != nil {
					r.Logger.Debug("failed to register agent", "agent_id", agentID, "error", err)
					continue
				}

				// Register each action
				for _, actionID := range actionIDs {
					fullActionID := fmt.Sprintf("@%s/%s:%s", agentName, actionID, version)
					actionData := map[string]interface{}{
						"agent":   agentName,
						"version": version,
						"action":  actionID,
						"path":    filepath.Join(path, version, "resources"),
					}
					actionJSON, _ := json.Marshal(actionData)
					_, err = r.DB.Exec("INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)", fullActionID, string(actionJSON))
					if err != nil {
						r.Logger.Debug("failed to register action", "action_id", fullActionID, "error", err)
					}
				}

				r.Logger.Debug("registered agent", "agent_id", agentID, "version", version, "actions", len(actionIDs))
			}

			return nil
		})
	}

	// Also scan /agent/project/resources/ for ActionID definitions
	resourcesDir := "/agent/project/resources/"
	if _, err := r.Fs.Stat(resourcesDir); err == nil {
		files, err := afero.ReadDir(r.Fs, resourcesDir)
		if err == nil {
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".pkl") {
					continue
				}
				resourcePath := filepath.Join(resourcesDir, file.Name())
				content, err := afero.ReadFile(r.Fs, resourcePath)
				if err != nil {
					r.Logger.Debug("failed to read resource file", "file", resourcePath, "error", err)
					continue
				}
				lines := strings.Split(string(content), "\n")
				var agentName, version string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "ActionID") && strings.Contains(line, "=") {
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							value := strings.TrimSpace(parts[1])
							value = strings.Trim(value, `"'`)
							if value != "" {
								actionID = value
							}
						}
					}
					if strings.HasPrefix(line, "AgentID") && strings.Contains(line, "=") {
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							value := strings.TrimSpace(parts[1])
							value = strings.Trim(value, `"'`)
							if value != "" {
								agentName = value
							}
						}
					}
					if strings.HasPrefix(line, "Version") && strings.Contains(line, "=") {
						parts := strings.SplitN(line, "=", 2)
						if len(parts) == 2 {
							value := strings.TrimSpace(parts[1])
							value = strings.Trim(value, `"'`)
							if value != "" {
								version = value
							}
						}
					}
				}
				// If ActionID is canonical, extract agent, action, version
				if actionID != "" && (agentName == "" || version == "") {
					if strings.HasPrefix(actionID, "@") {
						id := strings.TrimPrefix(actionID, "@")
						parts := strings.SplitN(id, "/", 2)
						if len(parts) == 2 {
							agentName = strings.SplitN(parts[0], ":", 2)[0]
							actionPart := parts[1]
							actionParts := strings.SplitN(actionPart, ":", 2)
							if len(actionParts) == 2 {
								version = actionParts[1]
							}
						}
					}
				}
				if actionID != "" && agentName != "" && version != "" {
					fullActionID := "@" + agentName + "/" + strings.Split(strings.TrimPrefix(actionID, "@"+agentName+"/"), ":")[0] + ":" + version
					actionData := map[string]interface{}{
						"agent":   agentName,
						"version": version,
						"action":  actionID,
						"path":    resourcePath,
					}
					actionJSON, _ := json.Marshal(actionData)
					_, err = r.DB.Exec("INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)", fullActionID, string(actionJSON))
					if err != nil {
						r.Logger.Debug("failed to register action from resources dir", "action_id", fullActionID, "error", err)
					}
				}
			}
		}
	}

	// Also scan /agent/project/agents/ for agent definition files
	agentsDir = "/agent/project/agents/"
	if _, err := r.Fs.Stat(agentsDir); err == nil {
		files, err := afero.ReadDir(r.Fs, agentsDir)
		if err == nil {
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				agentPath := filepath.Join(agentsDir, file.Name())
				content, err := afero.ReadFile(r.Fs, agentPath)
				if err != nil {
					r.Logger.Debug("failed to read agent file", "file", agentPath, "error", err)
					continue
				}

				// Parse the agent file content to extract the canonical action ID
				if strings.HasPrefix(strings.TrimSpace(string(content)), "@") {
					// This is a canonical action ID, register it directly
					actionID = strings.TrimSpace(string(content))

					// Extract agent name, action name, and version from the canonical ID
					id := strings.TrimPrefix(actionID, "@")
					parts := strings.SplitN(id, "/", 2)
					if len(parts) == 2 {
						agentName := strings.SplitN(parts[0], ":", 2)[0]
						actionPart := parts[1]
						actionParts := strings.SplitN(actionPart, ":", 2)
						actionName := actionParts[0]
						version := "1.0.0" // Default version if not specified
						if len(actionParts) == 2 {
							version = actionParts[1]
						}

						actionData := map[string]interface{}{
							"agent":   agentName,
							"version": version,
							"action":  actionName,
							"path":    agentPath,
						}
						actionJSON, _ := json.Marshal(actionData)
						_, err = r.DB.Exec("INSERT OR REPLACE INTO agents (id, data) VALUES (?, ?)", actionID, string(actionJSON))
						if err != nil {
							r.Logger.Debug("failed to register agent from agents dir", "action_id", actionID, "error", err)
						} else {
							r.Logger.Debug("registered agent from agents dir", "action_id", actionID, "file", file.Name())
						}
					}
				}
			}
		}
	}

	return nil
}

// extractActionIDsFromWorkflow parses a workflow.pkl file to extract all actionIDs
func (r *PklResourceReader) extractActionIDsFromWorkflow(workflowPath string) ([]string, error) {
	// Check if filesystem is available
	if r.Fs == nil {
		return nil, fmt.Errorf("filesystem not available")
	}

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

	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating over rows: %w", err)
	}

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

// getAgentDB returns a database connection from the agent pool
func getAgentDB() *sql.DB {
	agentDBPoolMutex.Lock()
	defer agentDBPoolMutex.Unlock()

	if agentDBPool == nil {
		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			return nil
		}

		// Create the agents table
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			db.Close()
			return nil
		}

		agentDBPool = db
		return agentDBPool
	}

	// Test the connection to ensure it's still valid
	if err := agentDBPool.Ping(); err != nil {
		// Connection is closed, create a new one
		agentDBPool.Close()

		db, err := sql.Open("sqlite3", ":memory:")
		if err != nil {
			return nil
		}

		// Create the agents table
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS agents (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		if err != nil {
			db.Close()
			return nil
		}

		agentDBPool = db
	}

	return agentDBPool
}

// GetGlobalAgentReader returns the singleton agent reader instance.
// This is now simplified to use the unified context system.
func GetGlobalAgentReader(fs afero.Fs, kdepsDir string, currentAgent string, currentVersion string, logger *logging.Logger) (*PklResourceReader, error) {
	// For now, create a new instance each time - this will be replaced by the unified context
	return InitializeAgent(fs, kdepsDir, currentAgent, currentVersion, logger)
}

// CloseGlobalAgentReader closes the global agent reader and resets the singleton.
// This should be called during application shutdown.
func CloseGlobalAgentReader() error {
	// This will be handled by the unified context system
	return nil
}
