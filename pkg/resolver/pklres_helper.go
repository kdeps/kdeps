package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// PklresHelper provides a simple interface to the generic pklres key-value store
type PklresHelper struct {
	resolver *DependencyResolver
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

	// Canonicalize the collection key
	actionID := h.resolveActionID(collectionKey)
	if !strings.HasPrefix(actionID, "@") {
		return "", fmt.Errorf("pklres Get: actionID '%s' is not canonical (must start with @)", actionID)
	}

	graphID := "<nil>"
	if h.resolver.PklresReader != nil {
		graphID = h.resolver.PklresReader.GraphID
	}
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("PklresHelper.Get", "actionID", actionID, "graphID", graphID, "key", key)
	}

	// Use pklres to retrieve the value
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&key=%s&op=get",
		h.resolver.RequestID, actionID, key))
	if err != nil {
		return "", fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	data, err := h.resolver.PklresReader.Read(*uri)
	if err != nil {
		if h.resolver.Logger != nil {
			h.resolver.Logger.Debug("PklresHelper.Get FAILED", "actionID", actionID, "graphID", graphID, "key", key, "err", err)
		}
		return "", fmt.Errorf("failed to get value from pklres: %w", err)
	}

	// Parse the JSON response
	var result string
	if err := json.Unmarshal(data, &result); err != nil {
		// If it's not a string, return the raw JSON
		if h.resolver.Logger != nil {
			h.resolver.Logger.Debug("PklresHelper.Get result (raw)", "actionID", actionID, "graphID", graphID, "key", key, "result", string(data))
		}
		return string(data), nil
	}
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("PklresHelper.Get result", "actionID", actionID, "graphID", graphID, "key", key, "result", result)
	}
	return result, nil
}

// Set stores a value in the generic key-value store
func (h *PklresHelper) Set(collectionKey, key, value string) error {
	if h == nil || h.resolver == nil || h.resolver.PklresReader == nil {
		return errors.New("PklresHelper not properly initialized")
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
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("PklresHelper.Set", "actionID", actionID, "graphID", graphID, "key", key, "value", value)
	}

	// Use pklres to store the value
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&key=%s&op=set&value=%s",
		h.resolver.RequestID, actionID, key, url.QueryEscape(value)))
	if err != nil {
		return fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		if h.resolver.Logger != nil {
			h.resolver.Logger.Debug("PklresHelper.Set FAILED", "actionID", actionID, "graphID", graphID, "key", key, "value", value, "err", err)
		}
		return fmt.Errorf("failed to set value in pklres: %w", err)
	}
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("PklresHelper.Set SUCCESS", "actionID", actionID, "graphID", graphID, "key", key, "value", value)
	}
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
