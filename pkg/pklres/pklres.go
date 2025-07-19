package pklres

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
)

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

// PklResourceReader implements the pkl.ResourceReader interface for a simple key-value store.
// Collection key is the CanonicalID from Agent reader, keys are template attributes.
// Scoping is by GraphID and set internally in the backend.
// Always returns JSON format.
type PklResourceReader struct {
	Fs             afero.Fs
	Logger         *logging.Logger
	GraphID        string // Current graphID for scoping operations (set internally)
	CurrentAgent   string // Current agent name for ActionID resolution
	CurrentVersion string // Current agent version for ActionID resolution
	KdepsPath      string // Path to kdeps directory for agent reader

	// Simple in-memory key-value store: graphID -> canonicalID -> key -> value
	store      map[string]map[string]map[string]string
	storeMutex sync.RWMutex

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

// Read retrieves, sets, or lists PKL records in the key-value store based on the URI.
// Always returns JSON format.
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
		return r.getKeyValue(collectionKey, key)
	case "set":
		collectionKey := query.Get("collection")
		key := query.Get("key")
		value := query.Get("value")
		return r.setKeyValue(collectionKey, key, value)
	case "list":
		collectionKey := query.Get("collection")
		return r.listKeys(collectionKey)
	case "async_resolve":
		// New async resolution operation
		actionID := query.Get("actionID")
		return r.asyncResolve(actionID)
	case "async_status":
		// Check async resolution status
		actionID := query.Get("actionID")
		return r.getAsyncStatus(actionID)
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

// IsInDependencyGraph checks if an actionID exists in the dependency graph
// Returns true if no dependency graph has been set up (backward compatibility)
func (r *PklResourceReader) IsInDependencyGraph(actionID string) bool {
	r.dependencyMutex.RLock()
	defer r.dependencyMutex.RUnlock()

	// If no dependency store exists, allow all operations (backward compatibility)
	if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil {
		return true
	}

	// If dependency store is empty, allow all operations (no graph set up)
	if len(r.dependencyStore[r.GraphID]) == 0 {
		return true
	}

	_, exists := r.dependencyStore[r.GraphID][actionID]
	return exists
}

// getKeyValue retrieves a value from the key-value store and returns it as JSON
func (r *PklResourceReader) getKeyValue(collectionKey, key string) ([]byte, error) {
	if collectionKey == "" || key == "" {
		return nil, errors.New("get operation requires collection and key parameters")
	}

	r.Logger.Debug("getKeyValue: retrieving", "collectionKey", collectionKey, "key", key, "graphID", r.GraphID)

	// Canonicalize the collection key using Agent reader
	canonicalCollectionKey := collectionKey
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
				Path:     "/" + collectionKey,
				RawQuery: query.Encode(),
			}

			resolvedIDBytes, err := agentReader.Read(uri)
			if err == nil {
				canonicalCollectionKey = string(resolvedIDBytes)
				r.Logger.Debug("getKeyValue: resolved ActionID", "original", collectionKey, "canonical", canonicalCollectionKey)
			} else {
				r.Logger.Debug("getKeyValue: failed to resolve ActionID, using original", "collectionKey", collectionKey, "error", err)
			}
		} else {
			r.Logger.Debug("getKeyValue: failed to get agent reader, using original collectionKey", "collectionKey", collectionKey, "error", err)
		}
	}

	// Check if this collection key exists in the dependency graph
	// If not, return null as per the requirement that pklres calls not in the graph should do nothing
	if !r.IsInDependencyGraph(canonicalCollectionKey) {
		r.Logger.Debug("getKeyValue: collection key not in dependency graph, returning null", "collectionKey", canonicalCollectionKey, "key", key)
		return []byte("null"), nil
	}

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
	r.Logger.Debug("getKeyValue: retrieved value", "collectionKey", canonicalCollectionKey, "key", key, "value", value, "exists", exists)

	if !exists {
		r.Logger.Debug("getKeyValue: key not found", "collectionKey", canonicalCollectionKey, "key", key)

		// For backward compatibility: return error if no dependency graph is set up
		if r.dependencyStore == nil || r.dependencyStore[r.GraphID] == nil || len(r.dependencyStore[r.GraphID]) == 0 {
			return nil, fmt.Errorf("key '%s' not found", key)
		}

		// Return null as JSON when key doesn't exist (async mode)
		return []byte("null"), nil
	}

	// Always return JSON format
	return json.Marshal(value)
}

// setKeyValue stores a value in the key-value store and returns it as JSON
func (r *PklResourceReader) setKeyValue(collectionKey, key, value string) ([]byte, error) {
	if collectionKey == "" || key == "" {
		return nil, errors.New("set operation requires collection and key parameters")
	}
	if value == "" {
		return nil, errors.New("set operation requires a value parameter")
	}

	r.Logger.Debug("setKeyValue: storing", "collectionKey", collectionKey, "key", key, "value", value, "graphID", r.GraphID)

	// Canonicalize the collection key using Agent reader
	canonicalCollectionKey := collectionKey
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
				Path:     "/" + collectionKey,
				RawQuery: query.Encode(),
			}

			resolvedIDBytes, err := agentReader.Read(uri)
			if err == nil {
				canonicalCollectionKey = string(resolvedIDBytes)
				r.Logger.Debug("setKeyValue: resolved ActionID", "original", collectionKey, "canonical", canonicalCollectionKey)
			} else {
				r.Logger.Debug("setKeyValue: failed to resolve ActionID, using original", "collectionKey", collectionKey, "error", err)
			}
		} else {
			r.Logger.Debug("setKeyValue: failed to get agent reader, using original collectionKey", "collectionKey", collectionKey, "error", err)
		}
	}

	// Check if this collection key exists in the dependency graph
	// If not, return success but don't store anything as per the requirement
	if !r.IsInDependencyGraph(canonicalCollectionKey) {
		r.Logger.Debug("setKeyValue: collection key not in dependency graph, ignoring set operation", "collectionKey", canonicalCollectionKey, "key", key)
		// Return the value as JSON to indicate success, but don't store it
		return json.Marshal(value)
	}

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

	r.store[r.GraphID][canonicalCollectionKey][key] = value
	r.Logger.Debug("setKeyValue: stored value", "collectionKey", canonicalCollectionKey, "key", key, "value", value)

	// Return the stored value as JSON
	return json.Marshal(value)
}

// listKeys lists all keys in a collection and returns them as JSON array
func (r *PklResourceReader) listKeys(collectionKey string) ([]byte, error) {
	if collectionKey == "" {
		return nil, errors.New("list operation requires collection parameter")
	}

	r.Logger.Debug("listKeys: listing", "collectionKey", collectionKey, "graphID", r.GraphID)

	// Canonicalize the collection key using Agent reader
	canonicalCollectionKey := collectionKey
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
				Path:     "/" + collectionKey,
				RawQuery: query.Encode(),
			}

			resolvedIDBytes, err := agentReader.Read(uri)
			if err == nil {
				canonicalCollectionKey = string(resolvedIDBytes)
				r.Logger.Debug("listKeys: resolved ActionID", "original", collectionKey, "canonical", canonicalCollectionKey)
			} else {
				r.Logger.Debug("listKeys: failed to resolve ActionID, using original", "collectionKey", collectionKey, "error", err)
			}
		} else {
			r.Logger.Debug("listKeys: failed to get agent reader, using original collectionKey", "collectionKey", collectionKey, "error", err)
		}
	}

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

	r.Logger.Debug("listKeys: found keys", "collectionKey", canonicalCollectionKey, "keys", keys)

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

// UpdateGlobalPklresReaderContext updates the context of the global pklres reader
func UpdateGlobalPklresReaderContext(graphID, currentAgent, currentVersion, kdepsPath string) error {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalPklresReader == nil {
		return errors.New("global pklres reader is not initialized")
	}

	globalPklresReader.GraphID = graphID
	globalPklresReader.CurrentAgent = currentAgent
	globalPklresReader.CurrentVersion = currentVersion
	globalPklresReader.KdepsPath = kdepsPath

	return nil
}

// InitializePklResource initializes a new PklResourceReader
func InitializePklResource(graphID, currentAgent, currentVersion, kdepsPath string, fs afero.Fs) (*PklResourceReader, error) {
	reader := &PklResourceReader{
		Fs:                  fs,
		Logger:              logging.GetLogger(),
		GraphID:             graphID,
		CurrentAgent:        currentAgent,
		CurrentVersion:      currentVersion,
		KdepsPath:           kdepsPath,
		store:               make(map[string]map[string]map[string]string),
		dependencyStore:     make(map[string]map[string]*DependencyData),
		dependencyCallbacks: make(map[string][]func(string, *DependencyData)),
	}

	// Set as global reader
	SetGlobalPklresReader(reader)

	return reader, nil
}
