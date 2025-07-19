package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/kdeps/kdeps/pkg/schema"
)

// PklresHelper manages PKL storage operations using the pklres key-value store system
type PklresHelper struct {
	resolver       *DependencyResolver
	isLoadingPhase bool // Flag to indicate if we're in the resource loading phase
}

// NewPklresHelper creates a new PklresHelper instance
func NewPklresHelper(dr *DependencyResolver) *PklresHelper {
	return &PklresHelper{resolver: dr, isLoadingPhase: false}
}

// SetLoadingPhase sets the loading phase flag
func (h *PklresHelper) SetLoadingPhase(loading bool) {
	if h != nil {
		h.isLoadingPhase = loading
	}
}

// IsLoadingPhase returns true if we're in the resource loading phase
func (h *PklresHelper) IsLoadingPhase() bool {
	if h == nil {
		return false
	}
	return h.isLoadingPhase
}

// getRecordSafely safely retrieves a record from pklres without blocking
func (h *PklresHelper) getRecordSafely(resourceID, resourceType string) map[string]interface{} {
	// During resource loading phase, return nil to avoid circular dependencies
	if h.isLoadingPhase {
		return nil
	}

	// For now, return nil during resource loading phase to avoid circular dependencies
	// Once the resource dependency ordering is fixed, this can be implemented properly
	// to return actual data from processed resources
	return nil
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
	if h == nil || h.resolver == nil || h.resolver.Context == nil {
		// Default fallback when no context available
		return fmt.Sprintf("extends \"package://schema.kdeps.com/core@0.3.0#/%s\"\n\n", info.SchemaFile)
	}
	return fmt.Sprintf("extends \"%s\"\n\n", schema.ImportPath(h.resolver.Context, info.SchemaFile))
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

	// Use graphID as the id in the path with a proper key
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&key=content&op=set&value=%s",
		graphID, resourceID, url.QueryEscape(content)))
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
	// Note: The content is stored with the "content" key
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&key=content",
		graphID, resourceID))
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
	listURI, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&op=list", h.resolver.RequestID, resourceType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse list URI: %w", err)
	}

	data, err := h.resolver.PklresReader.Read(*listURI)
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

// GetResourcePath returns the pklres path for a resource type (for compatibility with existing code)
func (h *PklresHelper) GetResourcePath(resourceType string) string {
	// Validate resource type
	validTypes := map[string]bool{
		"exec":     true,
		"python":   true,
		"llm":      true,
		"http":     true,
		"data":     true,
		"response": true,
		"resource": true,
	}

	if !validTypes[resourceType] {
		return ""
	}

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

// StoreResourceRecord stores a resource record using the set operation
func (h *PklresHelper) StoreResourceRecord(resourceType, resourceID, key, value string) error {
	if h == nil {
		return errors.New("PklresHelper is nil")
	}

	// Use graphID as the primary identifier for storage - ensures single record collection per graphID
	graphID := h.resolver.RequestID
	if h.resolver.Logger != nil {
		h.resolver.Logger.Debug("StoreResourceRecord: using graphID for storage", "resourceType", resourceType, "resourceID", resourceID, "key", key, "graphID", graphID)
	}

	// Use set operation - this will create or update the record
	uri, err := url.Parse(fmt.Sprintf("pklres:///%s?collection=%s&key=%s&op=set&value=%s",
		graphID, resourceID, url.QueryEscape(key), url.QueryEscape(value)))
	if err != nil {
		return fmt.Errorf("failed to parse pklres URI: %w", err)
	}

	_, err = h.resolver.PklresReader.Read(*uri)
	if err != nil {
		return fmt.Errorf("failed to store resource record: %w", err)
	}

	return nil
}

// StoreResourceObject stores a resource object by converting it to proper PKL content
func (h *PklresHelper) StoreResourceObject(resourceType, resourceID string, resourceObject interface{}) error {
	if h == nil {
		return errors.New("PklresHelper is nil")
	}

	// Generate proper PKL content for the resource object
	pklContent, err := h.generateResourcePklContent(resourceType, resourceID, resourceObject)
	if err != nil {
		return fmt.Errorf("failed to generate PKL content: %w", err)
	}

	// Store the PKL content
	return h.StorePklContent(resourceType, resourceID, pklContent)
}

// generateResourcePklContent generates proper PKL content for a resource object
func (h *PklresHelper) generateResourcePklContent(resourceType, resourceID string, resourceObject interface{}) (string, error) {
	// Generate header
	header := h.generatePklHeader(resourceType)

	// Generate the resource content based on type
	var resourceContent string
	var err error

	switch resourceType {
	case "llm":
		resourceContent, err = h.generateLLMContent(resourceID, resourceObject)
	case "python":
		resourceContent, err = h.generatePythonContent(resourceID, resourceObject)
	case "exec":
		resourceContent, err = h.generateExecContent(resourceID, resourceObject)
	case "http":
		resourceContent, err = h.generateHTTPContent(resourceID, resourceObject)
	case "data":
		resourceContent, err = h.generateDataContent(resourceID, resourceObject)
	default:
		return "", fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if err != nil {
		return "", err
	}

	// Combine header and content
	return header + resourceContent, nil
}

// generateLLMContent generates PKL content for LLM resources
func (h *PklresHelper) generateLLMContent(resourceID string, resourceObject interface{}) (string, error) {
	var content strings.Builder
	content.WriteString("// LLM Resource Content - Key-Value Store Format\n")
	content.WriteString("// Individual attributes are stored as key-value pairs in pklres\n")
	content.WriteString("// Collection key: CanonicalID from Agent reader\n")
	content.WriteString("// Keys: template attributes (model, role, prompt, response, etc.)\n")
	content.WriteString("// Values: obtained directly from pklres as key-value pairs\n\n")

	// For now, just add basic fields - this can be expanded based on the actual LLM object structure
	content.WriteString("// Example of how to access LLM attributes:\n")
	content.WriteString("// model = core.getPklValue(\"" + resourceID + "\", \"llm\", \"model\")\n")
	content.WriteString("// role = core.getPklValue(\"" + resourceID + "\", \"llm\", \"role\")\n")
	content.WriteString("// prompt = core.getPklValue(\"" + resourceID + "\", \"llm\", \"prompt\")\n")
	content.WriteString("// response = core.getPklValue(\"" + resourceID + "\", \"llm\", \"response\")\n")
	content.WriteString("// file = core.getPklValue(\"" + resourceID + "\", \"llm\", \"file\")\n")
	content.WriteString("// jsonResponse = core.getPklValue(\"" + resourceID + "\", \"llm\", \"jsonResponse\")\n")
	content.WriteString("// timeoutDuration = core.getPklValue(\"" + resourceID + "\", \"llm\", \"timeoutDuration\")\n")
	content.WriteString("// timestamp = core.getPklValue(\"" + resourceID + "\", \"llm\", \"timestamp\")\n")

	return content.String(), nil
}

// generatePythonContent generates PKL content for Python resources
func (h *PklresHelper) generatePythonContent(resourceID string, resourceObject interface{}) (string, error) {
	var content strings.Builder
	content.WriteString("// Python Resource Content - Key-Value Store Format\n")
	content.WriteString("// Individual attributes are stored as key-value pairs in pklres\n")
	content.WriteString("// Collection key: CanonicalID from Agent reader\n")
	content.WriteString("// Keys: template attributes (script, stdout, stderr, exitCode, etc.)\n")
	content.WriteString("// Values: obtained directly from pklres as key-value pairs\n\n")

	// For now, just add basic fields - this can be expanded based on the actual Python object structure
	content.WriteString("// Example of how to access Python attributes:\n")
	content.WriteString("// script = core.getPklValue(\"" + resourceID + "\", \"python\", \"script\")\n")
	content.WriteString("// stdout = core.getPklValue(\"" + resourceID + "\", \"python\", \"stdout\")\n")
	content.WriteString("// stderr = core.getPklValue(\"" + resourceID + "\", \"python\", \"stderr\")\n")
	content.WriteString("// exitCode = core.getPklValue(\"" + resourceID + "\", \"python\", \"exitCode\")\n")
	content.WriteString("// file = core.getPklValue(\"" + resourceID + "\", \"python\", \"file\")\n")
	content.WriteString("// timestamp = core.getPklValue(\"" + resourceID + "\", \"python\", \"timestamp\")\n")
	content.WriteString("// env = core.getPklValue(\"" + resourceID + "\", \"python\", \"env\")\n")

	return content.String(), nil
}

// generateExecContent generates PKL content for Exec resources
func (h *PklresHelper) generateExecContent(resourceID string, resourceObject interface{}) (string, error) {
	var content strings.Builder
	content.WriteString("// Exec Resource Content - Key-Value Store Format\n")
	content.WriteString("// Individual attributes are stored as key-value pairs in pklres\n")
	content.WriteString("// Collection key: CanonicalID from Agent reader\n")
	content.WriteString("// Keys: template attributes (command, stdout, stderr, exitCode, etc.)\n")
	content.WriteString("// Values: obtained directly from pklres as key-value pairs\n\n")

	// For now, just add basic fields - this can be expanded based on the actual Exec object structure
	content.WriteString("// Example of how to access Exec attributes:\n")
	content.WriteString("// command = core.getPklValue(\"" + resourceID + "\", \"exec\", \"command\")\n")
	content.WriteString("// stdout = core.getPklValue(\"" + resourceID + "\", \"exec\", \"stdout\")\n")
	content.WriteString("// stderr = core.getPklValue(\"" + resourceID + "\", \"exec\", \"stderr\")\n")
	content.WriteString("// exitCode = core.getPklValue(\"" + resourceID + "\", \"exec\", \"exitCode\")\n")
	content.WriteString("// file = core.getPklValue(\"" + resourceID + "\", \"exec\", \"file\")\n")
	content.WriteString("// timestamp = core.getPklValue(\"" + resourceID + "\", \"exec\", \"timestamp\")\n")
	content.WriteString("// env = core.getPklValue(\"" + resourceID + "\", \"exec\", \"env\")\n")

	return content.String(), nil
}

// generateHTTPContent generates PKL content for HTTP resources
func (h *PklresHelper) generateHTTPContent(resourceID string, resourceObject interface{}) (string, error) {
	var content strings.Builder
	content.WriteString("// HTTP Resource Content - Key-Value Store Format\n")
	content.WriteString("// Individual attributes are stored as key-value pairs in pklres\n")
	content.WriteString("// Collection key: CanonicalID from Agent reader\n")
	content.WriteString("// Keys: template attributes (url, method, headers, body, response, etc.)\n")
	content.WriteString("// Values: obtained directly from pklres as key-value pairs\n\n")

	// For now, just add basic fields - this can be expanded based on the actual HTTP object structure
	content.WriteString("// Example of how to access HTTP attributes:\n")
	content.WriteString("// url = core.getPklValue(\"" + resourceID + "\", \"http\", \"url\")\n")
	content.WriteString("// method = core.getPklValue(\"" + resourceID + "\", \"http\", \"method\")\n")
	content.WriteString("// headers = core.getPklValue(\"" + resourceID + "\", \"http\", \"headers\")\n")
	content.WriteString("// body = core.getPklValue(\"" + resourceID + "\", \"http\", \"body\")\n")
	content.WriteString("// response = core.getPklValue(\"" + resourceID + "\", \"http\", \"response\")\n")
	content.WriteString("// statusCode = core.getPklValue(\"" + resourceID + "\", \"http\", \"statusCode\")\n")
	content.WriteString("// file = core.getPklValue(\"" + resourceID + "\", \"http\", \"file\")\n")
	content.WriteString("// timestamp = core.getPklValue(\"" + resourceID + "\", \"http\", \"timestamp\")\n")

	return content.String(), nil
}

// generateDataContent generates PKL content for Data resources
func (h *PklresHelper) generateDataContent(resourceID string, resourceObject interface{}) (string, error) {
	var content strings.Builder
	content.WriteString("// Data Resource Content - Key-Value Store Format\n")
	content.WriteString("// Individual attributes are stored as key-value pairs in pklres\n")
	content.WriteString("// Collection key: CanonicalID from Agent reader\n")
	content.WriteString("// Keys: template attributes (files, content, etc.)\n")
	content.WriteString("// Values: obtained directly from pklres as key-value pairs\n\n")

	// For now, just add basic fields - this can be expanded based on the actual Data object structure
	content.WriteString("// Example of how to access Data attributes:\n")
	content.WriteString("// files = core.getPklValue(\"" + resourceID + "\", \"data\", \"files\")\n")
	content.WriteString("// content = core.getPklValue(\"" + resourceID + "\", \"data\", \"content\")\n")
	content.WriteString("// filepath = core.getPklValue(\"" + resourceID + "\", \"data\", \"filepath\")\n")

	return content.String(), nil
}
