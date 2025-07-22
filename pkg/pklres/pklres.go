package pklres

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

var (
	globalPklresReader *PklResourceReader
	globalMutex        sync.RWMutex
	operationCounter   map[string]int // graphID -> counter
	operationMutex     sync.Mutex
)

// QueryCache stores cached query results to avoid repeated operations
type QueryCache struct {
	Results map[string]interface{} `json:"results"`
	TTL     time.Time              `json:"ttl"`
	Query   string                 `json:"query"`
}

// RelationalRow represents a row in a relational result set
type RelationalRow struct {
	Data map[string]interface{} `json:"data"`
}

// RelationalResult represents the result of a relational algebra operation
type RelationalResult struct {
	Rows    []RelationalRow `json:"rows"`
	Columns []string        `json:"columns"`
	Query   string          `json:"query"`
	TTL     time.Time       `json:"ttl"`
}

// JoinCondition defines how to join two collections
type JoinCondition struct {
	LeftCollection  string `json:"leftCollection"`
	RightCollection string `json:"rightCollection"`
	LeftKey         string `json:"leftKey"`
	RightKey        string `json:"rightKey"`
	JoinType        string `json:"joinType"` // "inner", "left", "right", "full"
}

// ProjectionCondition defines which columns to include/exclude
type ProjectionCondition struct {
	Columns []string `json:"columns"` // Empty means all columns
	Exclude []string `json:"exclude"` // Columns to exclude
}

// SelectionCondition defines filtering criteria
type SelectionCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // "eq", "ne", "gt", "lt", "gte", "lte", "contains", "in"
	Value    interface{} `json:"value"`
}

// DependencyData holds metadata for async dependency resolution
type DependencyData struct {
	ActionID     string   `json:"actionID"`
	Dependents   []string `json:"dependents"`   // Resources that depend on this
	Dependencies []string `json:"dependencies"` // Resources this depends on
	Status       string   `json:"status"`       // "pending", "processing", "completed", "error"
	ResultData   string   `json:"resultData"`   // Actual execution results
	Timestamp    int64    `json:"timestamp"`
	Error        string   `json:"error,omitempty"`
	CompletedAt  int64    `json:"completedAt,omitempty"`
}

// PklResourceReader implements a generic key-value store scoped by graphID and actionID collection keys.
// It can store anything from shallow to deep nested data without schema restrictions.
// Storage structure: graphID -> actionID -> key -> value (JSON)
type PklResourceReader struct {
	Fs             afero.Fs
	Logger         *logging.Logger
	GraphID        string // Current graphID for scoping operations
	CurrentAgent   string // Current agent name for ActionID resolution
	CurrentVersion string // Current agent version for ActionID resolution
	KdepsPath      string // Path to kdeps directory for agent reader
	ProcessID      string // Current process ID for resource execution context

	// Generic key-value store: graphID -> actionID -> key -> value (JSON string)
	store      map[string]map[string]map[string]string
	storeMutex sync.RWMutex

	// Query cache: graphID -> queryHash -> QueryCache
	queryCache map[string]map[string]*QueryCache
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration // Default TTL for cached queries

	// Async dependency management: graphID -> actionID -> DependencyData
	dependencyStore     map[string]map[string]*DependencyData
	dependencyMutex     sync.RWMutex
	dependencyCallbacks map[string][]func(string, *DependencyData) // actionID -> callbacks
	callbackMutex       sync.RWMutex
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

// resolveActionID canonicalizes an actionID using the agent reader
// Keeps trying until it gets a canonical ID, then returns that
func (r *PklResourceReader) resolveActionID(actionID string) string {
	if r.CurrentAgent != "" && r.CurrentVersion != "" && r.KdepsPath != "" {
		// Keep trying until we get a canonical ID
		maxRetries := 10
		for attempt := 0; attempt < maxRetries; attempt++ {
			// Use agent reader to resolve the action ID
			agentReader, err := agent.GetGlobalAgentReader(r.Fs, r.KdepsPath, r.CurrentAgent, r.CurrentVersion, r.Logger)
			if err != nil {
				r.Logger.Debug("resolveActionID: failed to get agent reader, retrying", "actionID", actionID, "attempt", attempt+1, "error", err)
				continue
			}

			// Create URI for agent ID resolution
			query := url.Values{}
			query.Set("op", "resolve")
			query.Set("agent", r.CurrentAgent)
			query.Set("version", r.CurrentVersion)
			uri := url.URL{
				Scheme:   "agent",
				Path:     "/" + actionID,
				RawQuery: query.Encode(),
			}

			resolvedIDBytes, err := agentReader.Read(uri)
			if err != nil {
				r.Logger.Debug("resolveActionID: failed to resolve, retrying", "actionID", actionID, "attempt", attempt+1, "error", err)
				continue
			}

			resolvedID := string(resolvedIDBytes)
			// Check if we got a canonical ID (should start with @)
			if resolvedID != "" && resolvedID[0] == '@' {
				r.Logger.Debug("resolveActionID: resolved to canonical", "original", actionID, "canonical", resolvedID, "attempt", attempt+1)
				return resolvedID
			}

			r.Logger.Debug("resolveActionID: got non-canonical result, retrying", "actionID", actionID, "resolved", resolvedID, "attempt", attempt+1)
		}

		r.Logger.Warn("resolveActionID: failed to get canonical ID after retries, using original", "actionID", actionID, "maxRetries", maxRetries)
	} else {
		r.Logger.Debug("resolveActionID: no agent context available, using original", "actionID", actionID)
	}
	return actionID
}

// Read handles generic key-value store operations: get, set, list
// Always returns JSON format. Can store any data structure.
func (r *PklResourceReader) Read(uri url.URL) ([]byte, error) {
	// Check if receiver is nil and try to use global reader
	if r == nil {
		globalReader := GetGlobalPklresReader()
		if globalReader != nil {
			r = globalReader
		} else {
			newReader, err := InitializePklResource("default", "", "", "", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize PklResourceReader: %w", err)
			}
			r = newReader
		}
	}

	r.Logger.Debug("PklResourceReader.Read called", "uri", uri.String())

	// For global reader, ensure store is initialized
	globalReader := GetGlobalPklresReader()
	if r == globalReader {
		if r.store == nil {
			r.store = make(map[string]map[string]map[string]string)
		}
		r.Logger.Debug("using global pklres reader", "graphID", r.GraphID)
	}

	// Parse URI components
	query := uri.Query()
	op := query.Get("op")

	switch op {
	case "get":
		collectionKey := query.Get("collection")
		key := query.Get("key")
		result, err := r.getKeyValue(collectionKey, key)
		if err != nil {
			r.Logger.Error("PklResourceReader.Read: getKeyValue failed", "collection", collectionKey, "key", key, "error", err)
			return nil, fmt.Errorf("pklres get operation failed for collection=%s key=%s: %w", collectionKey, key, err)
		}
		r.Logger.Debug("PklResourceReader.Read: getKeyValue succeeded", "collection", collectionKey, "key", key, "result", string(result))
		return result, nil
	case "set":
		collectionKey := query.Get("collection")
		key := query.Get("key")
		value := query.Get("value")
		result, err := r.setKeyValue(collectionKey, key, value)
		if err != nil {
			r.Logger.Error("PklResourceReader.Read: setKeyValue failed", "collection", collectionKey, "key", key, "error", err)
			return nil, fmt.Errorf("pklres set operation failed for collection=%s key=%s: %w", collectionKey, key, err)
		}
		r.Logger.Debug("PklResourceReader.Read: setKeyValue succeeded", "collection", collectionKey, "key", key, "result", string(result))
		return result, nil
	case "list":
		collectionKey := query.Get("collection")
		return r.listKeys(collectionKey)
	case "async_resolve":
		actionID := query.Get("actionID")
		return r.asyncResolve(actionID)
	case "async_status":
		actionID := query.Get("actionID")
		return r.getAsyncStatus(actionID)
	case "relationalSelect":
		collectionKey := query.Get("collection")
		conditionsJson := query.Get("conditions")
		return r.handleRelationalSelect(collectionKey, conditionsJson)
	case "relationalProject":
		collectionKey := query.Get("collection")
		conditionJson := query.Get("condition")
		return r.handleRelationalProject(collectionKey, conditionJson)
	case "relationalJoin":
		conditionJson := query.Get("condition")
		return r.handleRelationalJoin(conditionJson)
	case "clearCache":
		return r.handleClearCache()
	case "setCacheTTL":
		ttlStr := query.Get("ttl")
		return r.handleSetCacheTTL(ttlStr)
	case "getCacheStats":
		return r.handleGetCacheStats()
	case "queryWithCache":
		queryType := query.Get("queryType")
		paramsJson := query.Get("params")
		return r.handleQueryWithCache(queryType, paramsJson)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", op)
	}
}

// asyncResolve initiates async resolution for a given actionID
func (r *PklResourceReader) asyncResolve(actionID string) ([]byte, error) {
	if actionID == "" {
		return nil, errors.New("async_resolve operation requires actionID parameter")
	}

	r.Logger.Debug("asyncResolve: initiating", "actionID", actionID, "graphID", r.GraphID)

	// Initialize dependency store if needed
	r.dependencyMutex.Lock()
	if r.dependencyStore == nil {
		r.dependencyStore = make(map[string]map[string]*DependencyData)
	}
	if r.dependencyStore[r.GraphID] == nil {
		r.dependencyStore[r.GraphID] = make(map[string]*DependencyData)
	}
	r.dependencyMutex.Unlock()

	// Create or update dependency data
	r.dependencyMutex.Lock()
	depData, exists := r.dependencyStore[r.GraphID][actionID]
	if !exists {
		depData = &DependencyData{
			ActionID:     actionID,
			Status:       "pending",
			Timestamp:    time.Now().UnixNano(),
			Dependents:   []string{},
			Dependencies: []string{},
		}
		r.dependencyStore[r.GraphID][actionID] = depData
	}
	r.dependencyMutex.Unlock()

	// Return the dependency data as JSON
	return json.Marshal(depData)
}

// getAsyncStatus returns the current status of async resolution for an actionID
func (r *PklResourceReader) getAsyncStatus(actionID string) ([]byte, error) {
	if actionID == "" {
		return nil, errors.New("async_status operation requires actionID parameter")
	}

	r.Logger.Debug("getAsyncStatus: checking", "actionID", actionID, "graphID", r.GraphID)

	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		// No dependency data exists, return pending status
		depData := &DependencyData{
			ActionID:     actionID,
			Status:       "pending",
			Timestamp:    time.Now().UnixNano(),
			Dependents:   []string{},
			Dependencies: []string{},
		}
		return json.Marshal(depData)
	}

	depData, exists := r.dependencyStore[r.GraphID][actionID]
	if !exists {
		// No dependency data for this actionID, return pending status
		depData = &DependencyData{
			ActionID:     actionID,
			Status:       "pending",
			Timestamp:    time.Now().UnixNano(),
			Dependents:   []string{},
			Dependencies: []string{},
		}
	}

	return json.Marshal(depData)
}

// PreResolveDependencies pre-resolves all pklres dependencies based on the execution order
func (r *PklResourceReader) PreResolveDependencies(executionOrder []string, resourceDependencies map[string][]string) error {
	if len(executionOrder) == 0 {
		r.Logger.Debug("PreResolveDependencies: no execution order provided")
		return nil
	}

	r.Logger.Info("PreResolveDependencies: starting", "executionOrder", executionOrder, "graphID", r.GraphID)

	// Initialize dependency store if needed
	r.dependencyMutex.Lock()
	if r.dependencyStore == nil {
		r.dependencyStore = make(map[string]map[string]*DependencyData)
	}
	if r.dependencyStore[r.GraphID] == nil {
		r.dependencyStore[r.GraphID] = make(map[string]*DependencyData)
	}
	r.dependencyMutex.Unlock()

	// Initialize callback store if needed
	r.callbackMutex.Lock()
	if r.dependencyCallbacks == nil {
		r.dependencyCallbacks = make(map[string][]func(string, *DependencyData))
	}
	r.callbackMutex.Unlock()

	// Create dependency data for each actionID in execution order
	for i, actionID := range executionOrder {
		r.dependencyMutex.Lock()
		depData := &DependencyData{
			ActionID:     actionID,
			Status:       "pending",
			Timestamp:    time.Now().UnixNano(),
			Dependents:   []string{},
			Dependencies: []string{},
		}

		// Set up dependencies based on resourceDependencies map
		if deps, exists := resourceDependencies[actionID]; exists {
			depData.Dependencies = deps
			r.Logger.Debug("PreResolveDependencies: set dependencies", "actionID", actionID, "dependencies", deps)
		}

		r.dependencyStore[r.GraphID][actionID] = depData
		r.dependencyMutex.Unlock()

		r.Logger.Debug("PreResolveDependencies: created dependency data", "actionID", actionID, "index", i)
	}

	// Set up dependents relationships (reverse of dependencies)
	for actionID, deps := range resourceDependencies {
		for _, depID := range deps {
			if depData, exists := r.dependencyStore[r.GraphID][depID]; exists {
				// Add this actionID as a dependent of depID
				depData.Dependents = append(depData.Dependents, actionID)
				r.Logger.Debug("PreResolveDependencies: set dependent", "dependencyID", depID, "dependentID", actionID)
			}
		}
	}

	r.Logger.Info("PreResolveDependencies: completed", "actionCount", len(executionOrder))
	return nil
}

// UpdateDependencyStatus updates the status of a dependency
func (r *PklResourceReader) UpdateDependencyStatus(actionID, status, resultData string, err error) error {
	if actionID == "" {
		return errors.New("actionID is required")
	}

	r.Logger.Debug("UpdateDependencyStatus: updating", "actionID", actionID, "status", status, "graphID", r.GraphID)

	r.dependencyMutex.Lock()
	defer r.dependencyMutex.Unlock()

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return errors.New("dependency store not initialized")
	}

	depData, exists := r.dependencyStore[r.GraphID][actionID]
	if !exists {
		return fmt.Errorf("dependency data not found for actionID: %s", actionID)
	}

	depData.Status = status
	depData.ResultData = resultData
	if err != nil {
		depData.Error = err.Error()
	}
	if status == "completed" || status == "error" {
		depData.CompletedAt = time.Now().UnixNano()
	}

	r.Logger.Debug("UpdateDependencyStatus: updated", "actionID", actionID, "status", status)

	// Trigger callbacks if any
	r.callbackMutex.RLock()
	callbacks, exists := r.dependencyCallbacks[actionID]
	r.callbackMutex.RUnlock()

	if exists && len(callbacks) > 0 {
		for _, callback := range callbacks {
			go callback(actionID, depData)
		}
	}

	return nil
}

// RegisterDependencyCallback registers a callback function for when a dependency status changes
func (r *PklResourceReader) RegisterDependencyCallback(actionID string, callback func(string, *DependencyData)) {
	if actionID == "" || callback == nil {
		return
	}

	r.callbackMutex.Lock()
	defer r.callbackMutex.Unlock()

	if r.dependencyCallbacks == nil {
		r.dependencyCallbacks = make(map[string][]func(string, *DependencyData))
	}

	r.dependencyCallbacks[actionID] = append(r.dependencyCallbacks[actionID], callback)
	r.Logger.Debug("RegisterDependencyCallback: registered", "actionID", actionID)
}

// GetDependencyData returns the dependency data for a given actionID
func (r *PklResourceReader) GetDependencyData(actionID string) (*DependencyData, error) {
	if actionID == "" {
		return nil, errors.New("actionID is required")
	}

	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return nil, fmt.Errorf("dependency data not found for actionID: %s", actionID)
	}

	depData, exists := r.dependencyStore[r.GraphID][actionID]
	if !exists {
		return nil, fmt.Errorf("dependency data not found for actionID: %s", actionID)
	}

	return depData, nil
}

// IsDependencyReady checks if a dependency is ready (completed or error)
func (r *PklResourceReader) IsDependencyReady(actionID string) bool {
	depData, err := r.GetDependencyData(actionID)
	if err != nil {
		return false
	}
	return depData.Status == "completed" || depData.Status == "error"
}

// AreAllDependenciesReady checks if all dependencies for a given actionID are ready
func (r *PklResourceReader) AreAllDependenciesReady(actionID string) bool {
	depData, err := r.GetDependencyData(actionID)
	if err != nil {
		return false
	}

	for _, depID := range depData.Dependencies {
		if !r.IsDependencyReady(depID) {
			return false
		}
	}
	return true
}

// WaitForDependencies waits for all dependencies of a given actionID to be ready
func (r *PklResourceReader) WaitForDependencies(actionID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if r.AreAllDependenciesReady(actionID) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for dependencies of actionID: %s", actionID)
}

// GetDependencyStatusSummary returns a summary of all dependency statuses
func (r *PklResourceReader) GetDependencyStatusSummary() map[string]string {
	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	summary := make(map[string]string)

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return summary
	}

	for actionID, depData := range r.dependencyStore[r.GraphID] {
		summary[actionID] = depData.Status
	}

	return summary
}

// GetPendingDependencies returns a list of actionIDs that are still pending
func (r *PklResourceReader) GetPendingDependencies() []string {
	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	var pending []string

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return pending
	}

	for actionID, depData := range r.dependencyStore[r.GraphID] {
		if depData.Status == "pending" {
			pending = append(pending, actionID)
		}
	}

	return pending
}

func (r *PklResourceReader) IsInDependencyGraph(actionID string) bool {
	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	// System collections with format @agentID/<graphID> are always allowed
	if strings.Contains(actionID, "/") && strings.HasPrefix(actionID, "@") {
		// This is a system collection format, always allow it
		return true
	}

	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return true
	}

	if len(r.dependencyStore[r.GraphID]) == 0 {
		return true
	}

	_, exists := r.dependencyStore[r.GraphID][actionID]
	return exists
}

// getNextOperationNumber returns the next operation number for a given graphID
func getNextOperationNumber(graphID string) int {
	operationMutex.Lock()
	defer operationMutex.Unlock()

	if operationCounter == nil {
		operationCounter = make(map[string]int)
	}

	operationCounter[graphID]++
	return operationCounter[graphID]
}

// setKeyValue stores a value in the generic key-value store
func (r *PklResourceReader) setKeyValue(collectionKey, key, value string) ([]byte, error) {
	if collectionKey == "" || key == "" {
		return nil, errors.New("set operation requires collection and key parameters")
	}
	if value == "" {
		return nil, errors.New("set operation requires a value parameter")
	}

	// Get caller information for context
	caller := "unknown"
	if pc, _, line, ok := runtime.Caller(2); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fmt.Sprintf("%s:%d", filepath.Base(fn.Name()), line)
		}
	}

	// Get operation number for this graph
	opNumber := getNextOperationNumber(r.GraphID)

	r.Logger.Debug("setKeyValue: storing",
		"collectionKey", collectionKey,
		"key", key,
		"value", value,
		"graphID", r.GraphID,
		"CurrentAgent", r.CurrentAgent,
		"CurrentVersion", r.CurrentVersion,
		"KdepsPath", r.KdepsPath,
		"caller", caller,
		"op", fmt.Sprintf("SET [%d]", opNumber),
		"processID", r.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD SET %s", r.ProcessID))

	// Handle special "current" collection - automatically resolve to system collection format
	var canonicalCollectionKey string
	if collectionKey == "current" {
		if r.CurrentAgent == "" || r.CurrentVersion == "" {
			return nil, errors.New("cannot resolve 'current' collection without CurrentAgent and CurrentVersion")
		}
		canonicalCollectionKey = fmt.Sprintf("@%s:%s", r.CurrentAgent, r.CurrentVersion)
		r.Logger.Debug("setKeyValue: resolved 'current' collection", "original", collectionKey, "canonical", canonicalCollectionKey)
	} else {
		// Canonicalize the collection key normally
		canonicalCollectionKey = r.resolveActionID(collectionKey)
	}

	r.Logger.Debug("setKeyValue: canonicalized", "canonicalCollectionKey", canonicalCollectionKey, "graphID", r.GraphID)

	// Store the value
	r.storeMutex.Lock()
	defer r.storeMutex.Unlock()

	// Initialize nested maps if they don't exist
	if r.store == nil {
		r.store = make(map[string]map[string]map[string]string)
	}
	if r.store[r.GraphID] == nil {
		r.store[r.GraphID] = make(map[string]map[string]string)
	}
	if r.store[r.GraphID][canonicalCollectionKey] == nil {
		r.store[r.GraphID][canonicalCollectionKey] = make(map[string]string)
	}

	// Store the value (value is already JSON from the URI parameter)
	r.store[r.GraphID][canonicalCollectionKey][key] = value

	r.Logger.Debug("setKeyValue: stored value",
		"collectionKey", canonicalCollectionKey,
		"key", key,
		"value", value,
		"graphID", r.GraphID,
		"caller", caller,
		"op", fmt.Sprintf("SET [%d]", opNumber),
		"processID", r.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD SET %s", r.ProcessID))

	// Return the stored value as JSON
	return json.Marshal(value)
}

// getKeyValue retrieves a value from the generic key-value store
func (r *PklResourceReader) getKeyValue(collectionKey, key string) ([]byte, error) {
	if collectionKey == "" || key == "" {
		return nil, errors.New("get operation requires collection and key parameters")
	}

	// Get caller information for context
	caller := "unknown"
	if pc, _, line, ok := runtime.Caller(2); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fmt.Sprintf("%s:%d", filepath.Base(fn.Name()), line)
		}
	}

	// Get operation number for this graph
	opNumber := getNextOperationNumber(r.GraphID)

	r.Logger.Debug("getKeyValue: retrieving",
		"collectionKey", collectionKey,
		"key", key,
		"graphID", r.GraphID,
		"CurrentAgent", r.CurrentAgent,
		"CurrentVersion", r.CurrentVersion,
		"KdepsPath", r.KdepsPath,
		"caller", caller,
		"op", fmt.Sprintf("GET [%d]", opNumber),
		"processID", r.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", r.ProcessID))

	// Handle special "current" collection - automatically resolve to system collection format
	var canonicalCollectionKey string
	if collectionKey == "current" {
		if r.CurrentAgent == "" || r.CurrentVersion == "" {
			return nil, errors.New("cannot resolve 'current' collection without CurrentAgent and CurrentVersion")
		}
		canonicalCollectionKey = fmt.Sprintf("@%s:%s", r.CurrentAgent, r.CurrentVersion)
		r.Logger.Debug("getKeyValue: resolved 'current' collection", "original", collectionKey, "canonical", canonicalCollectionKey)
	} else {
		// Canonicalize the collection key normally
		canonicalCollectionKey = r.resolveActionID(collectionKey)
	}

	r.Logger.Debug("getKeyValue: canonicalized", "canonicalCollectionKey", canonicalCollectionKey, "graphID", r.GraphID)

	// Get the value from the store
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()

	// Initialize nested maps if they don't exist
	if r.store == nil {
		r.store = make(map[string]map[string]map[string]string)
	}
	if r.store[r.GraphID] == nil {
		r.store[r.GraphID] = make(map[string]map[string]string)
	}
	if r.store[r.GraphID][canonicalCollectionKey] == nil {
		r.store[r.GraphID][canonicalCollectionKey] = make(map[string]string)
	}

	value, exists := r.store[r.GraphID][canonicalCollectionKey][key]
	r.Logger.Debug("getKeyValue: retrieved value",
		"collectionKey", canonicalCollectionKey,
		"key", key,
		"exists", exists,
		"value", value,
		"graphID", r.GraphID,
		"caller", caller,
		"op", fmt.Sprintf("GET [%d]", opNumber),
		"processID", r.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", r.ProcessID))

	if !exists {
		r.Logger.Warn("getKeyValue: key not found, returning null", "collectionKey", canonicalCollectionKey, "key", key, "graphID", r.GraphID, "caller", caller, "op", fmt.Sprintf("GET [%d]", opNumber), "processID", r.ProcessID, "prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", r.ProcessID))
		// Return null when key doesn't exist - this allows PKL to handle missing values appropriately
		return []byte("null"), nil
	}

	// Return the stored value as JSON
	return json.Marshal(value)
}

// listKeys lists all keys in a collection
func (r *PklResourceReader) listKeys(collectionKey string) ([]byte, error) {
	if collectionKey == "" {
		return nil, errors.New("list operation requires collection parameter")
	}

	r.Logger.Debug("listKeys: listing", "collectionKey", collectionKey, "graphID", r.GraphID)

	// Canonicalize the collection key
	canonicalCollectionKey := r.resolveActionID(collectionKey)

	// Get the keys from the store
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()

	// Initialize nested maps if they don't exist
	if r.store == nil {
		r.store = make(map[string]map[string]map[string]string)
	}
	if r.store[r.GraphID] == nil {
		r.store[r.GraphID] = make(map[string]map[string]string)
	}
	if r.store[r.GraphID][canonicalCollectionKey] == nil {
		r.store[r.GraphID][canonicalCollectionKey] = make(map[string]string)
	}

	// Extract keys
	keys := make([]string, 0, len(r.store[r.GraphID][canonicalCollectionKey]))
	for key := range r.store[r.GraphID][canonicalCollectionKey] {
		keys = append(keys, key)
	}

	r.Logger.Debug("listKeys: found keys", "collectionKey", canonicalCollectionKey, "count", len(keys))

	// Return keys as JSON array
	return json.Marshal(keys)
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

// UpdateGlobalPklresReaderContext updates the global pklres reader context
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

// UpdateGlobalPklresReaderProcessID updates the process ID for the global pklres reader
func UpdateGlobalPklresReaderProcessID(processID string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalPklresReader == nil {
		return errors.New("global pklres reader not initialized")
	}

	globalPklresReader.ProcessID = processID
	return nil
}

// InitializePklResource creates a new PklResourceReader with the given parameters
func InitializePklResource(graphID, currentAgent, currentVersion, kdepsPath string, fs afero.Fs) (*PklResourceReader, error) {
	reader := &PklResourceReader{
		Fs:                  fs,
		Logger:              logging.GetLogger(),
		GraphID:             graphID,
		CurrentAgent:        currentAgent,
		CurrentVersion:      currentVersion,
		KdepsPath:           kdepsPath,
		store:               make(map[string]map[string]map[string]string),
		queryCache:          make(map[string]map[string]*QueryCache),
		cacheTTL:            5 * time.Minute, // Default 5 minute cache TTL
		dependencyStore:     make(map[string]map[string]*DependencyData),
		dependencyCallbacks: make(map[string][]func(string, *DependencyData)),
	}

	SetGlobalPklresReader(reader)
	return reader, nil
}

// Relational Algebra Methods

// generateQueryHash creates a unique hash for a query to use as cache key
func (r *PklResourceReader) generateQueryHash(query interface{}) string {
	queryBytes, _ := json.Marshal(query)
	hash := md5.Sum(queryBytes)
	return hex.EncodeToString(hash[:])
}

// getCachedQuery retrieves a cached query result if it exists and is not expired
func (r *PklResourceReader) getCachedQuery(queryHash string) (*QueryCache, bool) {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	if r.queryCache == nil || r.queryCache[r.GraphID] == nil {
		return nil, false
	}

	cache, exists := r.queryCache[r.GraphID][queryHash]
	if !exists {
		return nil, false
	}

	// Check if cache is expired
	if time.Now().After(cache.TTL) {
		// Remove expired cache
		r.cacheMutex.RUnlock()
		r.cacheMutex.Lock()
		delete(r.queryCache[r.GraphID], queryHash)
		r.cacheMutex.Unlock()
		r.cacheMutex.RLock()
		return nil, false
	}

	return cache, true
}

// setCachedQuery stores a query result in the cache
func (r *PklResourceReader) setCachedQuery(queryHash string, query interface{}, results map[string]interface{}) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	if r.queryCache == nil {
		r.queryCache = make(map[string]map[string]*QueryCache)
	}
	if r.queryCache[r.GraphID] == nil {
		r.queryCache[r.GraphID] = make(map[string]*QueryCache)
	}

	queryStr, _ := json.Marshal(query)
	r.queryCache[r.GraphID][queryHash] = &QueryCache{
		Results: results,
		TTL:     time.Now().Add(r.cacheTTL),
		Query:   string(queryStr),
	}

	r.Logger.Debug("Cached query result", "queryHash", queryHash, "graphID", r.GraphID, "ttl", r.cacheTTL)
}

// Select performs a selection operation (filtering) on a collection
func (r *PklResourceReader) Select(collectionKey string, conditions []SelectionCondition) (*RelationalResult, error) {
	query := map[string]interface{}{
		"operation":  "select",
		"collection": collectionKey,
		"conditions": conditions,
	}
	queryHash := r.generateQueryHash(query)

	// Check cache first
	if cache, exists := r.getCachedQuery(queryHash); exists {
		r.Logger.Debug("Using cached select result", "queryHash", queryHash, "collection", collectionKey)
		return &RelationalResult{
			Rows:    cache.Results["rows"].([]RelationalRow),
			Columns: cache.Results["columns"].([]string),
			Query:   cache.Query,
			TTL:     cache.TTL,
		}, nil
	}

	// Get all data from the collection
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()

	canonicalCollectionKey := r.resolveActionID(collectionKey)
	collection, exists := r.store[r.GraphID][canonicalCollectionKey]
	if !exists {
		return &RelationalResult{Rows: []RelationalRow{}, Columns: []string{}}, nil
	}

	var rows []RelationalRow
	var columns []string
	columnSet := make(map[string]bool)

	// Convert collection data to rows
	for key, value := range collection {
		row := RelationalRow{Data: make(map[string]interface{})}
		row.Data["key"] = key

		// Try to parse value as JSON, fallback to string
		var parsedValue interface{}
		if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
			row.Data["value"] = parsedValue
		} else {
			row.Data["value"] = value
		}

		// Apply selection conditions
		includeRow := true
		for _, condition := range conditions {
			if !r.evaluateCondition(row.Data, condition) {
				includeRow = false
				break
			}
		}

		if includeRow {
			rows = append(rows, row)
			// Track columns
			for col := range row.Data {
				columnSet[col] = true
			}
		}
	}

	// Convert column set to slice
	for col := range columnSet {
		columns = append(columns, col)
	}

	result := &RelationalResult{
		Rows:    rows,
		Columns: columns,
		Query:   fmt.Sprintf("SELECT * FROM %s WHERE %v", collectionKey, conditions),
		TTL:     time.Now().Add(r.cacheTTL),
	}

	// Cache the result
	r.setCachedQuery(queryHash, query, map[string]interface{}{
		"rows":    rows,
		"columns": columns,
	})

	r.Logger.Debug("Select operation completed", "collection", collectionKey, "rows", len(rows), "queryHash", queryHash)
	return result, nil
}

// Project performs a projection operation (column selection) on a collection
func (r *PklResourceReader) Project(collectionKey string, condition ProjectionCondition) (*RelationalResult, error) {
	query := map[string]interface{}{
		"operation":  "project",
		"collection": collectionKey,
		"condition":  condition,
	}
	queryHash := r.generateQueryHash(query)

	// Check cache first
	if cache, exists := r.getCachedQuery(queryHash); exists {
		r.Logger.Debug("Using cached project result", "queryHash", queryHash, "collection", collectionKey)
		return &RelationalResult{
			Rows:    cache.Results["rows"].([]RelationalRow),
			Columns: cache.Results["columns"].([]string),
			Query:   cache.Query,
			TTL:     cache.TTL,
		}, nil
	}

	// Get all data from the collection
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()

	canonicalCollectionKey := r.resolveActionID(collectionKey)
	collection, exists := r.store[r.GraphID][canonicalCollectionKey]
	if !exists {
		return &RelationalResult{Rows: []RelationalRow{}, Columns: []string{}}, nil
	}

	var rows []RelationalRow
	var columns []string

	// Convert collection data to rows
	for key, value := range collection {
		row := RelationalRow{Data: make(map[string]interface{})}
		row.Data["key"] = key

		// Try to parse value as JSON, fallback to string
		var parsedValue interface{}
		if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
			row.Data["value"] = parsedValue
		} else {
			row.Data["value"] = value
		}

		// Apply projection
		projectedRow := RelationalRow{Data: make(map[string]interface{})}

		if len(condition.Columns) > 0 {
			// Include only specified columns
			for _, col := range condition.Columns {
				if val, exists := row.Data[col]; exists {
					projectedRow.Data[col] = val
				}
			}
		} else {
			// Include all columns except excluded ones
			for col, val := range row.Data {
				excluded := false
				for _, excludeCol := range condition.Exclude {
					if col == excludeCol {
						excluded = true
						break
					}
				}
				if !excluded {
					projectedRow.Data[col] = val
				}
			}
		}

		rows = append(rows, projectedRow)
	}

	// Determine columns from first row
	if len(rows) > 0 {
		for col := range rows[0].Data {
			columns = append(columns, col)
		}
	}

	result := &RelationalResult{
		Rows:    rows,
		Columns: columns,
		Query:   fmt.Sprintf("PROJECT %v FROM %s", condition, collectionKey),
		TTL:     time.Now().Add(r.cacheTTL),
	}

	// Cache the result
	r.setCachedQuery(queryHash, query, map[string]interface{}{
		"rows":    rows,
		"columns": columns,
	})

	r.Logger.Debug("Project operation completed", "collection", collectionKey, "rows", len(rows), "queryHash", queryHash)
	return result, nil
}

// Join performs a join operation between two collections
func (r *PklResourceReader) Join(condition JoinCondition) (*RelationalResult, error) {
	query := map[string]interface{}{
		"operation": "join",
		"condition": condition,
	}
	queryHash := r.generateQueryHash(query)

	// Check cache first
	if cache, exists := r.getCachedQuery(queryHash); exists {
		r.Logger.Debug("Using cached join result", "queryHash", queryHash)
		return &RelationalResult{
			Rows:    cache.Results["rows"].([]RelationalRow),
			Columns: cache.Results["columns"].([]string),
			Query:   cache.Query,
			TTL:     cache.TTL,
		}, nil
	}

	// Get data from both collections
	r.storeMutex.RLock()
	defer r.storeMutex.RUnlock()

	leftCollectionKey := r.resolveActionID(condition.LeftCollection)
	rightCollectionKey := r.resolveActionID(condition.RightCollection)

	leftCollection, leftExists := r.store[r.GraphID][leftCollectionKey]
	rightCollection, rightExists := r.store[r.GraphID][rightCollectionKey]

	if !leftExists || !rightExists {
		return &RelationalResult{Rows: []RelationalRow{}, Columns: []string{}}, nil
	}

	var rows []RelationalRow
	var columns []string
	columnSet := make(map[string]bool)

	// Build join index for right collection
	rightIndex := make(map[string][]RelationalRow)
	for key, value := range rightCollection {
		row := RelationalRow{Data: make(map[string]interface{})}
		row.Data["key"] = key

		var parsedValue interface{}
		if err := json.Unmarshal([]byte(value), &parsedValue); err == nil {
			row.Data["value"] = parsedValue
		} else {
			row.Data["value"] = value
		}

		joinKey := fmt.Sprintf("%v", row.Data[condition.RightKey])
		rightIndex[joinKey] = append(rightIndex[joinKey], row)
	}

	// Perform join
	for leftKey, leftValue := range leftCollection {
		leftRow := RelationalRow{Data: make(map[string]interface{})}
		leftRow.Data["key"] = leftKey

		var parsedValue interface{}
		if err := json.Unmarshal([]byte(leftValue), &parsedValue); err == nil {
			leftRow.Data["value"] = parsedValue
		} else {
			leftRow.Data["value"] = leftValue
		}

		joinKey := fmt.Sprintf("%v", leftRow.Data[condition.LeftKey])
		rightRows, exists := rightIndex[joinKey]

		if exists {
			// Inner join - create joined rows
			for _, rightRow := range rightRows {
				joinedRow := RelationalRow{Data: make(map[string]interface{})}

				// Add left collection data with prefix
				for k, v := range leftRow.Data {
					joinedRow.Data[fmt.Sprintf("left_%s", k)] = v
				}

				// Add right collection data with prefix
				for k, v := range rightRow.Data {
					joinedRow.Data[fmt.Sprintf("right_%s", k)] = v
				}

				rows = append(rows, joinedRow)

				// Track columns
				for col := range joinedRow.Data {
					columnSet[col] = true
				}
			}
		} else if condition.JoinType == "left" || condition.JoinType == "full" {
			// Left join - include left row with null right values
			joinedRow := RelationalRow{Data: make(map[string]interface{})}

			for k, v := range leftRow.Data {
				joinedRow.Data[fmt.Sprintf("left_%s", k)] = v
			}

			// Add null values for right collection
			for k := range rightCollection {
				joinedRow.Data[fmt.Sprintf("right_%s", k)] = nil
			}

			rows = append(rows, joinedRow)

			for col := range joinedRow.Data {
				columnSet[col] = true
			}
		}
	}

	// Handle right join for full outer join
	if condition.JoinType == "right" || condition.JoinType == "full" {
		leftIndex := make(map[string]bool)
		for _, row := range rows {
			leftKey := fmt.Sprintf("%v", row.Data[fmt.Sprintf("left_%s", condition.LeftKey)])
			leftIndex[leftKey] = true
		}

		for _, rightRows := range rightIndex {
			for _, rightRow := range rightRows {
				rightKey := fmt.Sprintf("%v", rightRow.Data[condition.RightKey])
				if !leftIndex[rightKey] {
					// Right join - include right row with null left values
					joinedRow := RelationalRow{Data: make(map[string]interface{})}

					// Add null values for left collection
					for k := range leftCollection {
						joinedRow.Data[fmt.Sprintf("left_%s", k)] = nil
					}

					// Add right collection data
					for k, v := range rightRow.Data {
						joinedRow.Data[fmt.Sprintf("right_%s", k)] = v
					}

					rows = append(rows, joinedRow)

					for col := range joinedRow.Data {
						columnSet[col] = true
					}
				}
			}
		}
	}

	// Convert column set to slice
	for col := range columnSet {
		columns = append(columns, col)
	}

	result := &RelationalResult{
		Rows:    rows,
		Columns: columns,
		Query:   fmt.Sprintf("JOIN %s.%s = %s.%s (%s)", condition.LeftCollection, condition.LeftKey, condition.RightCollection, condition.RightKey, condition.JoinType),
		TTL:     time.Now().Add(r.cacheTTL),
	}

	// Cache the result
	r.setCachedQuery(queryHash, query, map[string]interface{}{
		"rows":    rows,
		"columns": columns,
	})

	r.Logger.Debug("Join operation completed", "left", condition.LeftCollection, "right", condition.RightCollection, "rows", len(rows), "queryHash", queryHash)
	return result, nil
}

// evaluateCondition evaluates a selection condition against row data
func (r *PklResourceReader) evaluateCondition(rowData map[string]interface{}, condition SelectionCondition) bool {
	fieldValue, exists := rowData[condition.Field]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "eq":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", condition.Value)
	case "ne":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", condition.Value)
	case "gt":
		return r.compareValues(fieldValue, condition.Value) > 0
	case "lt":
		return r.compareValues(fieldValue, condition.Value) < 0
	case "gte":
		return r.compareValues(fieldValue, condition.Value) >= 0
	case "lte":
		return r.compareValues(fieldValue, condition.Value) <= 0
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", condition.Value))
	case "in":
		if slice, ok := condition.Value.([]interface{}); ok {
			for _, val := range slice {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", val) {
					return true
				}
			}
		}
		return false
	default:
		return false
	}
}

// compareValues compares two values for ordering operations
func (r *PklResourceReader) compareValues(a, b interface{}) int {
	// Try numeric comparison first
	if aNum, aOk := a.(float64); aOk {
		if bNum, bOk := b.(float64); bOk {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// Fallback to string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.Compare(aStr, bStr)
}

// ClearCache clears the query cache for the current graph
func (r *PklResourceReader) ClearCache() {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	if r.queryCache != nil {
		delete(r.queryCache, r.GraphID)
	}
	r.Logger.Debug("Cleared query cache", "graphID", r.GraphID)
}

// SetCacheTTL sets the default TTL for cached queries
func (r *PklResourceReader) SetCacheTTL(ttl time.Duration) {
	r.cacheTTL = ttl
	r.Logger.Debug("Set cache TTL", "ttl", ttl)
}

// GetCacheStats returns statistics about the query cache
func (r *PklResourceReader) GetCacheStats() map[string]interface{} {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	if r.queryCache == nil || r.queryCache[r.GraphID] == nil {
		return map[string]interface{}{
			"total_queries":   0,
			"expired_queries": 0,
			"active_queries":  0,
		}
	}

	total := len(r.queryCache[r.GraphID])
	expired := 0
	active := 0

	for _, cache := range r.queryCache[r.GraphID] {
		if time.Now().After(cache.TTL) {
			expired++
		} else {
			active++
		}
	}

	return map[string]interface{}{
		"total_queries":   total,
		"expired_queries": expired,
		"active_queries":  active,
		"cache_ttl":       r.cacheTTL,
	}
}

// URI Handler Methods for Relational Algebra Operations

// handleRelationalSelect handles the relationalSelect URI operation
func (r *PklResourceReader) handleRelationalSelect(collectionKey, conditionsJson string) ([]byte, error) {
	if collectionKey == "" || conditionsJson == "" {
		return nil, errors.New("relationalSelect requires collection and conditions parameters")
	}

	var conditions []SelectionCondition
	if err := json.Unmarshal([]byte(conditionsJson), &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse conditions JSON: %w", err)
	}

	result, err := r.Select(collectionKey, conditions)
	if err != nil {
		return nil, fmt.Errorf("relationalSelect operation failed: %w", err)
	}

	return json.Marshal(result)
}

// handleRelationalProject handles the relationalProject URI operation
func (r *PklResourceReader) handleRelationalProject(collectionKey, conditionJson string) ([]byte, error) {
	if collectionKey == "" || conditionJson == "" {
		return nil, errors.New("relationalProject requires collection and condition parameters")
	}

	var condition ProjectionCondition
	if err := json.Unmarshal([]byte(conditionJson), &condition); err != nil {
		return nil, fmt.Errorf("failed to parse condition JSON: %w", err)
	}

	result, err := r.Project(collectionKey, condition)
	if err != nil {
		return nil, fmt.Errorf("relationalProject operation failed: %w", err)
	}

	return json.Marshal(result)
}

// handleRelationalJoin handles the relationalJoin URI operation
func (r *PklResourceReader) handleRelationalJoin(conditionJson string) ([]byte, error) {
	if conditionJson == "" {
		return nil, errors.New("relationalJoin requires condition parameter")
	}

	var condition JoinCondition
	if err := json.Unmarshal([]byte(conditionJson), &condition); err != nil {
		return nil, fmt.Errorf("failed to parse condition JSON: %w", err)
	}

	result, err := r.Join(condition)
	if err != nil {
		return nil, fmt.Errorf("relationalJoin operation failed: %w", err)
	}

	return json.Marshal(result)
}

// handleClearCache handles the clearCache URI operation
func (r *PklResourceReader) handleClearCache() ([]byte, error) {
	r.ClearCache()
	return json.Marshal("Cache cleared successfully")
}

// handleSetCacheTTL handles the setCacheTTL URI operation
func (r *PklResourceReader) handleSetCacheTTL(ttlStr string) ([]byte, error) {
	if ttlStr == "" {
		return nil, errors.New("setCacheTTL requires ttl parameter")
	}

	ttlSeconds, err := fmt.Sscanf(ttlStr, "%d", new(int))
	if err != nil {
		return nil, fmt.Errorf("failed to parse TTL: %w", err)
	}

	r.SetCacheTTL(time.Duration(ttlSeconds) * time.Second)
	return json.Marshal(fmt.Sprintf("Cache TTL set to %d seconds", ttlSeconds))
}

// handleGetCacheStats handles the getCacheStats URI operation
func (r *PklResourceReader) handleGetCacheStats() ([]byte, error) {
	stats := r.GetCacheStats()
	return json.Marshal(stats)
}

// handleQueryWithCache handles the queryWithCache URI operation
func (r *PklResourceReader) handleQueryWithCache(queryType, paramsJson string) ([]byte, error) {
	if queryType == "" || paramsJson == "" {
		return nil, errors.New("queryWithCache requires queryType and params parameters")
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJson), &params); err != nil {
		return nil, fmt.Errorf("failed to parse params JSON: %w", err)
	}

	// This would need to be implemented in PklresHelper
	// For now, return an error indicating this needs to be handled differently
	return nil, errors.New("queryWithCache operation not yet implemented in URI handler")
}
