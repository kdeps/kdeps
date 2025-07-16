package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/kdeps/kdeps/pkg/schema"
)

// PklresHelper manages PKL storage operations using the pklres SQLite3 system
type PklresHelper struct {
	resolver *DependencyResolver
}

// NewPklresHelper creates a new PklresHelper instance
func NewPklresHelper(dr *DependencyResolver) *PklresHelper {
	return &PklresHelper{resolver: dr}
}

// ResourceTypeInfo contains information about a resource type for PKL generation
type ResourceTypeInfo struct {
	SchemaFile string // e.g., "LLM.pkl", "Exec.pkl"
	BlockName  string // e.g., "Resources", "Files"
}

// getResourceTypeInfo returns the schema file and block name for a given resource type
func (h *PklresHelper) getResourceTypeInfo(resourceType string) ResourceTypeInfo {
	switch resourceType {
	case "llm":
		return ResourceTypeInfo{SchemaFile: "LLM.pkl", BlockName: "Resources"}
	case "exec":
		return ResourceTypeInfo{SchemaFile: "Exec.pkl", BlockName: "Resources"}
	case "python":
		return ResourceTypeInfo{SchemaFile: "Python.pkl", BlockName: "Resources"}
	case "client", "http":
		return ResourceTypeInfo{SchemaFile: "HTTP.pkl", BlockName: "Resources"}
	case "data":
		return ResourceTypeInfo{SchemaFile: "Data.pkl", BlockName: "Files"}
	default:
		return ResourceTypeInfo{SchemaFile: "Resource.pkl", BlockName: "Resources"}
	}
}

// generatePklHeader generates the PKL header for a given resource type
func (h *PklresHelper) generatePklHeader(resourceType string) string {
	info := h.getResourceTypeInfo(resourceType)
	var schemaVersion string
	if h == nil || h.resolver == nil || h.resolver.Context == nil {
		schemaVersion = "0.3.0" // Default fallback version
	} else {
		schemaVersion = schema.Version(h.resolver.Context)
	}
	return fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/%s\"\n\n", schemaVersion, info.SchemaFile)
}

// StorePklContent stores PKL content in pklres with the appropriate header
func (h *PklresHelper) StorePklContent(resourceType, resourceID, content string) error {
	if h == nil {
		return errors.New("PklresHelper is nil")
	}
	if h.resolver != nil && h.resolver.Logger != nil {
		h.resolver.Logger.Info("storePklContent: called", "resourceType", resourceType, "resourceID", resourceID, "content_length", len(content))
	}
	if h.resolver == nil {
		if h.resolver != nil && h.resolver.Logger != nil {
			h.resolver.Logger.Error("storePklContent: resolver is nil, cannot persist", "resourceType", resourceType, "resourceID", resourceID)
		}
		return errors.New("resolver is nil")
	}
	if h.resolver.PklresReader == nil {
		if h.resolver.Logger != nil {
			h.resolver.Logger.Error("storePklContent: PklresReader is nil, skipping persistence", "resourceType", resourceType, "resourceID", resourceID)
		}
		return nil
	}

	// Use graphID instead of full actionID for storage - tie data only to the operation
	graphID := h.resolver.RequestID // RequestID is the graphID for this operation
	if resourceType == "llm" && h.resolver.Logger != nil {
		h.resolver.Logger.Debug("storePklContent: LLM", "resourceType", resourceType, "resourceID", resourceID, "graphID", graphID, "content", content)
	}
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("storePklContent: storing", "resourceType", resourceType, "resourceID", resourceID, "graphID", graphID, "content", content)
	}

	// Use graphID as the id in the path
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&op=set&value=%s",
		graphID, resourceType, url.QueryEscape(content)))
	if err != nil {
		return fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return fmt.Errorf("failed to store PKL content in pklres: %w", err)
	}

	return nil
}

// RetrievePklContent retrieves PKL content from pklres
func (h *PklresHelper) RetrievePklContent(resourceType, resourceID string) (string, error) {
	if h == nil || h.resolver == nil {
		return "", errors.New("PklresHelper or resolver is nil")
	}

	// Use graphID instead of full actionID for retrieval - data is tied only to the operation
	graphID := h.resolver.RequestID // RequestID is the graphID for this operation
	if resourceType == "llm" && h.resolver.Logger != nil {
		h.resolver.Logger.Debug("retrievePklContent: LLM", "resourceType", resourceType, "resourceID", resourceID, "graphID", graphID)
	}

	// Use pklres to retrieve the content
	// Note: The content is stored with an empty key, so we need to use the graphID as the id and an empty key
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&key=",
		graphID, resourceType))
	if err != nil {
		return "", fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	if h.resolver.PklresReader == nil {
		return "", errors.New("PklresReader is nil")
	}

	// Removed excessive debug logging for pklres reads

	data, err := h.resolver.PklresReader.Read(*uri)
	if err != nil {
		// Only log errors at debug level, not every read attempt
		if h.resolver.Logger != nil {
			h.resolver.Logger.Debug("retrievePklContent: failed to read from pklres", "uri", uri.String(), "error", err)
		}
		return "", fmt.Errorf("failed to retrieve PKL content from pklres: %w", err)
	}

	content := string(data)
	// Log only at trace level to reduce spam
	if h.resolver.Logger != nil && h.resolver.Logger.IsDebugEnabled() {
		// Only log if content is empty or there's an error - success is expected
		if len(content) == 0 {
			h.resolver.Logger.Debug("retrievePklContent: empty content from pklres", "uri", uri.String())
		}
	}

	return content, nil
}

// retrieveAllResourcesForType retrieves all resources of a given type
func (h *PklresHelper) retrieveAllResourcesForType(resourceType string) (map[string]string, error) {
	// List all resource IDs for this type - use the request ID to find resources in the current request context
	listUri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&op=list", h.resolver.RequestID, resourceType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse list URI: %w", err)
	}

	data, err := h.resolver.PklresReader.Read(*listUri)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	// Parse the JSON list of IDs (assuming it's a simple JSON array)
	listStr := string(data)
	if listStr == "" || listStr == "[]" {
		return make(map[string]string), nil
	}

	// Simple parsing - extract resource IDs from JSON array
	// This assumes the list is in format ["id1", "id2", ...]
	ids := strings.Trim(listStr, "[]")
	if ids == "" {
		return make(map[string]string), nil
	}

	resources := make(map[string]string)

	// Split by comma and clean up quotes
	idParts := strings.Split(ids, ",")
	for _, idPart := range idParts {
		id := strings.Trim(strings.TrimSpace(idPart), "\"")
		if id != "" {
			content, err := h.RetrievePklContent(resourceType, id)
			if err != nil {
				h.resolver.Logger.Warn("failed to retrieve content for resource", "type", resourceType, "id", id, "error", err)
				continue
			}
			resources[id] = content
		}
	}

	h.resolver.Logger.Debug("retrieveAllResourcesForType: found keys", "resourceType", resourceType, "keys", resources)
	return resources, nil
}

// storeCompleteResourceMap stores a complete PKL document for a resource type
func (h *PklresHelper) storeCompleteResourceMap(resourceType string, resourcesMap map[string]interface{}) error {
	info := h.getResourceTypeInfo(resourceType)
	header := h.generatePklHeader(resourceType)

	var content strings.Builder
	content.WriteString(header)
	content.WriteString(fmt.Sprintf("%s {\n", info.BlockName))

	for resourceID, resourceData := range resourcesMap {
		// Convert resourceData to string representation
		// This is a simplified approach - in practice, you might want more sophisticated serialization
		resourceStr := fmt.Sprintf("%v", resourceData)
		content.WriteString(fmt.Sprintf("  [\"%s\"] = %s\n", resourceID, resourceStr))
	}

	content.WriteString("}\n")

	// Store the complete document
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&op=set&value=%s",
		h.resolver.RequestID, resourceType, url.QueryEscape(content.String())))
	if err != nil {
		return fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return fmt.Errorf("failed to store complete resource map: %w", err)
	}

	return nil
}

// clearResourceType clears all resources of a given type
func (h *PklresHelper) clearResourceType(resourceType string) error {
	uri, err := url.Parse(fmt.Sprintf("pklres:///_?type=%s&op=clear", resourceType))
	if err != nil {
		return fmt.Errorf("failed to parse clear URI: %w", err)
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return fmt.Errorf("failed to clear resources of type %s: %w", resourceType, err)
	}

	return nil
}

// GetResourcePath returns the pklres path for a resource type (for compatibility with existing code)
func (h *PklresHelper) GetResourcePath(resourceType string) string {
	// Handle nil resolver case (for tests or incomplete initialization)
	if h == nil || h.resolver == nil {
		return fmt.Sprintf("pklres:///test-request?type=%s", resourceType)
	}
	// Return a virtual path that indicates this is using pklres
	return fmt.Sprintf("pklres:///%s?type=%s", h.resolver.RequestID, resourceType)
}

// resolveActionID automatically resolves actionIDs using the agent package
func (h *PklresHelper) resolveActionID(actionID string) string {
	if h == nil || h.resolver == nil {
		if h != nil && h.resolver != nil && h.resolver.Logger != nil {
			h.resolver.Logger.Debug("resolveActionID: helper or resolver is nil", "actionID", actionID)
		}
		return actionID // Return as-is if we can't resolve
	}

	// Check if this looks like an actionID that needs resolution
	if strings.HasPrefix(actionID, "@") {
		if h.resolver.Logger != nil {
			h.resolver.Logger.Debug("resolveActionID: already canonical", "actionID", actionID)
		}
		return actionID
	}

	// If it doesn't start with @, it might be a local actionID that needs resolution
	// We need to use the agent package to resolve it
	if h.resolver.AgentReader == nil {
		if h.resolver.Logger != nil {
			h.resolver.Logger.Warn("resolveActionID: AgentReader is nil", "actionID", actionID)
		}
		return actionID
	}
	if h.resolver.AgentReader != nil {
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

			if h.resolver.Logger != nil {
				h.resolver.Logger.Debug("resolveActionID: attempting resolution", "actionID", actionID, "uri", uri.String(), "currentAgent", h.resolver.AgentReader.CurrentAgent, "currentVersion", h.resolver.AgentReader.CurrentVersion)
			}

			// Try to resolve using the agent reader
			data, err := h.resolver.AgentReader.Read(*uri)
			if err == nil && len(data) > 0 {
				// Successfully resolved, return the resolved ID
				resolvedID := string(data)
				if h.resolver.Logger != nil {
					h.resolver.Logger.Debug("resolveActionID: successfully resolved", "actionID", actionID, "resolvedID", resolvedID)
				}
				return resolvedID
			} else {
				if h.resolver.Logger != nil {
					h.resolver.Logger.Warn("resolveActionID: resolution failed", "actionID", actionID, "error", err, "dataLength", len(data))
				}
			}
		} else {
			if h.resolver.Logger != nil {
				h.resolver.Logger.Warn("resolveActionID: failed to parse URI", "actionID", actionID, "error", err)
			}
		}
	}

	// If resolution fails, return the original actionID
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("resolveActionID: returning original", "actionID", actionID)
	}
	return actionID
}
