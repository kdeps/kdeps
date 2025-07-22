package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kdeps/kdeps/pkg/pklres"
)

// PklresHelper provides a simple interface to the generic pklres key-value store
type PklresHelper struct {
	resolver *DependencyResolver
	// operationCounter is a map to track the sequence of operations per graph
	operationCounter map[string]int
}

// NewPklresHelper creates a new PklresHelper instance
func NewPklresHelper(dr *DependencyResolver) *PklresHelper {
	return &PklresHelper{resolver: dr}
}

// Get retrieves a value from the generic key-value store
func (h *PklresHelper) Get(collectionKey, key string) (string, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return "", errors.New("PklresHelper not properly initialized")
	}

	// Get caller information for context
	caller := "unknown"
	if pc, _, line, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fmt.Sprintf("%s:%d", filepath.Base(fn.Name()), line)
		}
	}

	// Canonicalize the collection key
	actionID := h.resolveActionID(collectionKey)
	if !strings.HasPrefix(actionID, "@") {
		return "", fmt.Errorf("pklres Get: actionID '%s' is not canonical (must start with @)", actionID)
	}

	graphID := "<nil>"
	if h.resolver.PklresReader != nil {
		graphID = h.resolver.PklresReader.GraphID
	}

	// Get operation number for this graph
	opNumber := h.getNextOperationNumber(graphID)

	h.resolver.Logger.Debug("PklresHelper.Get",
		"actionID", actionID,
		"graphID", graphID,
		"key", key,
		"caller", caller,
		"op", fmt.Sprintf("GET [%d]", opNumber),
		"processID", h.resolver.PklresReader.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", h.resolver.PklresReader.ProcessID))

	// Create URI for pklres get operation
	query := url.Values{}
	query.Set("op", "get")
	query.Set("collection", actionID)
	query.Set("key", key)
	uri := url.URL{
		Scheme:   "pklres",
		RawQuery: query.Encode(),
	}

	// Execute the get operation
	result, err := h.resolver.PklresReader.Read(uri)
	if err != nil {
		h.resolver.Logger.Error("PklresHelper.Get FAILED",
			"actionID", actionID,
			"graphID", graphID,
			"key", key,
			"error", err,
			"caller", caller,
			"op", fmt.Sprintf("GET [%d]", opNumber),
			"processID", h.resolver.PklresReader.ProcessID,
			"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", h.resolver.PklresReader.ProcessID))
		return "", fmt.Errorf("pklres Get failed for collection=%s key=%s: %w", actionID, key, err)
	}

	// Parse the result
	var value string
	if err := json.Unmarshal(result, &value); err != nil {
		h.resolver.Logger.Error("PklresHelper.Get: failed to unmarshal result",
			"actionID", actionID,
			"graphID", graphID,
			"key", key,
			"result", string(result),
			"error", err,
			"caller", caller,
			"op", fmt.Sprintf("GET [%d]", opNumber),
			"processID", h.resolver.PklresReader.ProcessID,
			"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", h.resolver.PklresReader.ProcessID))
		return "", fmt.Errorf("failed to unmarshal pklres result: %w", err)
	}

	h.resolver.Logger.Debug("PklresHelper.Get result",
		"actionID", actionID,
		"graphID", graphID,
		"key", key,
		"result", value,
		"caller", caller,
		"op", fmt.Sprintf("GET [%d]", opNumber),
		"processID", h.resolver.PklresReader.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD GET %s", h.resolver.PklresReader.ProcessID))
	return value, nil
}

// Set stores a value in the generic key-value store
func (h *PklresHelper) Set(collectionKey, key, value string) error {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return errors.New("PklresHelper not properly initialized")
	}

	// Get caller information for context
	caller := "unknown"
	if pc, _, line, ok := runtime.Caller(1); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			caller = fmt.Sprintf("%s:%d", filepath.Base(fn.Name()), line)
		}
	}

	// Canonicalize the collection key
	actionID := h.resolveActionID(collectionKey)
	if !strings.HasPrefix(actionID, "@") {
		return fmt.Errorf("pklres Set: actionID '%s' is not canonical (must start with @)", actionID)
	}

	graphID := "<nil>"
	if h.resolver.PklresReader != nil {
		graphID = h.resolver.PklresReader.GraphID
	}

	// Get operation number for this graph
	opNumber := h.getNextOperationNumber(graphID)

	h.resolver.Logger.Debug("PklresHelper.Set",
		"actionID", actionID,
		"graphID", graphID,
		"key", key,
		"value", value,
		"caller", caller,
		"op", fmt.Sprintf("SET [%d]", opNumber),
		"processID", h.resolver.PklresReader.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD SET %s", h.resolver.PklresReader.ProcessID))

	// Create URI for pklres set operation
	query := url.Values{}
	query.Set("op", "set")
	query.Set("collection", actionID)
	query.Set("key", key)
	query.Set("value", value)
	uri := url.URL{
		Scheme:   "pklres",
		RawQuery: query.Encode(),
	}

	// Execute the set operation
	_, err := h.resolver.PklresReader.Read(uri)
	if err != nil {
		h.resolver.Logger.Error("PklresHelper.Set FAILED",
			"actionID", actionID,
			"graphID", graphID,
			"key", key,
			"value", value,
			"error", err,
			"caller", caller,
			"op", fmt.Sprintf("SET [%d]", opNumber),
			"processID", h.resolver.PklresReader.ProcessID,
			"prefix", fmt.Sprintf("kdeps: 🚀 RECORD SET %s", h.resolver.PklresReader.ProcessID))
		return fmt.Errorf("pklres Set failed for collection=%s key=%s: %w", actionID, key, err)
	}

	h.resolver.Logger.Debug("PklresHelper.Set SUCCESS",
		"actionID", actionID,
		"graphID", graphID,
		"key", key,
		"value", value,
		"caller", caller,
		"op", fmt.Sprintf("SET [%d]", opNumber),
		"processID", h.resolver.PklresReader.ProcessID,
		"prefix", fmt.Sprintf("kdeps: 🚀 RECORD SET %s", h.resolver.PklresReader.ProcessID))
	return nil
}

// List returns all keys in a collection
func (h *PklresHelper) List(collectionKey string) ([]string, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	// Canonicalize the collection key
	actionID := h.resolveActionID(collectionKey)
	if !strings.HasPrefix(actionID, "@") {
		return nil, fmt.Errorf("pklres List: actionID '%s' is not canonical (must start with @)", actionID)
	}

	// Use pklres to list keys
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&op=list",
		h.resolver.RequestID, actionID))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	data, err := h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys from pklres: %w", err)
	}

	// Parse the JSON array response
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse keys from pklres: %w", err)
	}

	return keys, nil
}

// GetWithTimeout retrieves a value from the key-value store, waiting up to the specified timeout for it to appear.
func (h *PklresHelper) GetWithTimeout(collectionKey, key string, timeout time.Duration) (string, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return "", errors.New("PklresHelper not properly initialized")
	}

	actionID := h.resolveActionID(collectionKey)
	if !strings.HasPrefix(actionID, "@") {
		return "", fmt.Errorf("pklres GetWithTimeout: actionID '%s' is not canonical (must start with @)", actionID)
	}

	graphID := "<nil>"
	if h.resolver.PklresReader != nil {
		graphID = h.resolver.PklresReader.GraphID
	}
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("PklresHelper.GetWithTimeout", "actionID", actionID, "graphID", graphID, "key", key, "timeout", timeout)
	}

	deadline := time.Now().Add(timeout)
	for {
		value, err := h.Get(collectionKey, key)
		if err == nil && value != "" && value != "null" {
			return value, nil
		}
		if time.Now().After(deadline) {
			if h.resolver.Logger != nil {
				h.resolver.Logger.Warn("PklresHelper.GetWithTimeout: timeout reached", "actionID", actionID, "graphID", graphID, "key", key)
			}
			return "", fmt.Errorf("timeout waiting for key '%s' in collection '%s' (canonical: %s)", key, collectionKey, actionID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// resolveActionID automatically resolves actionIDs using the agent package
func (h *PklresHelper) resolveActionID(actionID string) string {
	if h == nil || h.resolver == nil {
		return actionID // Return as-is if we can't resolve
	}

	// Check if this looks like an actionID that needs resolution
	if strings.HasPrefix(actionID, "@") {
		return actionID
	}

	// If it doesn't start with @, it might be a local actionID that needs resolution
	if h.resolver.AgentReader == nil {
		return actionID
	}

	// Create a URI for agent resolution
	uri, err := url.Parse(fmt.Sprintf("agent:///%s", actionID))
	if err == nil {
		// Add current context as query parameters
		query := uri.Query()
		if h.resolver.AgentReader.CurrentAgent != "" {
			query.Set("agent", h.resolver.AgentReader.CurrentAgent)
		}
		if h.resolver.AgentReader.CurrentVersion != "" {
			query.Set("version", h.resolver.AgentReader.CurrentVersion)
		}
		uri.RawQuery = query.Encode()

		// Try to resolve using the agent reader
		data, err := h.resolver.AgentReader.Read(*uri)
		if err == nil && len(data) > 0 {
			// Successfully resolved, return the resolved ID
			return string(data)
		}
	}

	// If resolution fails, return the original actionID
	return actionID
}

// getNextOperationNumber returns the next operation number for the given graph
func (h *PklresHelper) getNextOperationNumber(graphID string) int {
	if h.operationCounter == nil {
		h.operationCounter = make(map[string]int)
	}
	h.operationCounter[graphID]++
	return h.operationCounter[graphID]
}

// Relational Algebra Methods

// Select performs a selection operation (filtering) on a collection
func (h *PklresHelper) Select(collectionKey string, conditions []pklres.SelectionCondition) (*pklres.RelationalResult, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	h.resolver.Logger.Debug("PklresHelper.Select",
		"collection", collectionKey,
		"conditions", conditions,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)

	return h.resolver.PklresReader.Select(collectionKey, conditions)
}

// Project performs a projection operation (column selection) on a collection
func (h *PklresHelper) Project(collectionKey string, condition pklres.ProjectionCondition) (*pklres.RelationalResult, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	h.resolver.Logger.Debug("PklresHelper.Project",
		"collection", collectionKey,
		"condition", condition,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)

	return h.resolver.PklresReader.Project(collectionKey, condition)
}

// Join performs a join operation between two collections
func (h *PklresHelper) Join(condition pklres.JoinCondition) (*pklres.RelationalResult, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	h.resolver.Logger.Debug("PklresHelper.Join",
		"leftCollection", condition.LeftCollection,
		"rightCollection", condition.RightCollection,
		"joinType", condition.JoinType,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)

	return h.resolver.PklresReader.Join(condition)
}

// ClearCache clears the query cache for the current graph
func (h *PklresHelper) ClearCache() error {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return errors.New("PklresHelper not properly initialized")
	}

	h.resolver.PklresReader.ClearCache()
	h.resolver.Logger.Debug("PklresHelper.ClearCache",
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)
	return nil
}

// SetCacheTTL sets the default TTL for cached queries
func (h *PklresHelper) SetCacheTTL(ttl time.Duration) error {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return errors.New("PklresHelper not properly initialized")
	}

	h.resolver.PklresReader.SetCacheTTL(ttl)
	h.resolver.Logger.Debug("PklresHelper.SetCacheTTL",
		"ttl", ttl,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)
	return nil
}

// GetCacheStats returns statistics about the query cache
func (h *PklresHelper) GetCacheStats() (map[string]interface{}, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	stats := h.resolver.PklresReader.GetCacheStats()
	h.resolver.Logger.Debug("PklresHelper.GetCacheStats",
		"stats", stats,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)
	return stats, nil
}

// QueryWithCache performs a query with automatic caching to avoid repeated operations
func (h *PklresHelper) QueryWithCache(queryType string, params map[string]interface{}) (*pklres.RelationalResult, error) {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return nil, errors.New("PklresHelper not properly initialized")
	}

	h.resolver.Logger.Debug("PklresHelper.QueryWithCache",
		"queryType", queryType,
		"params", params,
		"graphID", h.resolver.PklresReader.GraphID,
		"processID", h.resolver.PklresReader.ProcessID)

	switch queryType {
	case "select":
		collection, ok := params["collection"].(string)
		if !ok {
			return nil, errors.New("select query requires 'collection' parameter")
		}
		conditions, ok := params["conditions"].([]pklres.SelectionCondition)
		if !ok {
			return nil, errors.New("select query requires 'conditions' parameter")
		}
		return h.Select(collection, conditions)

	case "project":
		collection, ok := params["collection"].(string)
		if !ok {
			return nil, errors.New("project query requires 'collection' parameter")
		}
		condition, ok := params["condition"].(pklres.ProjectionCondition)
		if !ok {
			return nil, errors.New("project query requires 'condition' parameter")
		}
		return h.Project(collection, condition)

	case "join":
		condition, ok := params["condition"].(pklres.JoinCondition)
		if !ok {
			return nil, errors.New("join query requires 'condition' parameter")
		}
		return h.Join(condition)

	default:
		return nil, fmt.Errorf("unknown query type: %s", queryType)
	}
}
