package pklres

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

var (
	globalPklresReader *PklResourceReader
	globalMutex        sync.RWMutex
)

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
	default:
		return nil, fmt.Errorf("unsupported operation: %s", op)
	}
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
		r.Logger.Debug("getKeyValue: key not found, returning null", "collectionKey", canonicalCollectionKey, "key", key)
		// Return null as JSON when key doesn't exist
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
		Fs:             fs,
		Logger:         logging.GetLogger(),
		GraphID:        graphID,
		CurrentAgent:   currentAgent,
		CurrentVersion: currentVersion,
		KdepsPath:      kdepsPath,
		store:          make(map[string]map[string]map[string]string),
	}

	// Set as global reader
	SetGlobalPklresReader(reader)

	return reader, nil
}
