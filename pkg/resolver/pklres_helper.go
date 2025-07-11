package resolver

import (
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
		schemaVersion = schema.SchemaVersion(h.resolver.Context)
	}
	return fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/%s\"\n\n", schemaVersion, info.SchemaFile)
}

// storePklContent stores PKL content in pklres with the appropriate header
func (h *PklresHelper) storePklContent(resourceType, resourceID, content string) error {
	if h == nil || h.resolver == nil {
		return fmt.Errorf("PklresHelper or resolver is nil")
	}

	// Store just the content without the header - the PKL schema will handle the parsing
	// ID = RequestID, Type = resourceType, Key = resourceID, Value = content
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&key=%s&op=set&value=%s",
		h.resolver.RequestID, resourceType, resourceID, url.QueryEscape(content)))
	if err != nil {
		return fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	if h.resolver.PklresReader == nil {
		// In testing contexts we may skip actual persistence
		h.resolver.Logger.Info("PklresReader is nil - skipping storePklContent (likely test context)")
		return nil
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return fmt.Errorf("failed to store PKL content in pklres: %w", err)
	}

	return nil
}

// retrievePklContent retrieves PKL content from pklres
func (h *PklresHelper) retrievePklContent(resourceType, resourceID string) (string, error) {
	if h == nil || h.resolver == nil {
		return "", fmt.Errorf("PklresHelper or resolver is nil")
	}

	// Use pklres to retrieve the content
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?type=%s&key=%s",
		h.resolver.RequestID, resourceType, resourceID))
	if err != nil {
		return "", fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	if h.resolver.PklresReader == nil {
		return "", fmt.Errorf("PklresReader is nil")
	}

	data, err := h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve PKL content from pklres: %w", err)
	}

	return string(data), nil
}

// retrieveAllResourcesForType retrieves all resources of a given type
func (h *PklresHelper) retrieveAllResourcesForType(resourceType string) (map[string]string, error) {
	// First, list all resource IDs for this type
	listUri, err := url.Parse(fmt.Sprintf("pklres:///_?type=%s&op=list", resourceType))
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
			content, err := h.retrievePklContent(resourceType, id)
			if err != nil {
				h.resolver.Logger.Warn("failed to retrieve content for resource", "type", resourceType, "id", id, "error", err)
				continue
			}
			resources[id] = content
		}
	}

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

// getResourcePath returns the pklres path for a resource type (for compatibility with existing code)
func (h *PklresHelper) getResourcePath(resourceType string) string {
	// Handle nil resolver case (for tests or incomplete initialization)
	if h == nil || h.resolver == nil {
		return fmt.Sprintf("pklres:///test-request?type=%s", resourceType)
	}
	// Return a virtual path that indicates this is using pklres
	return fmt.Sprintf("pklres:///%s?type=%s", h.resolver.RequestID, resourceType)
}
