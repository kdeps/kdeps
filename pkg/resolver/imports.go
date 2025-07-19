package resolver

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/agent"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

func (dr *DependencyResolver) PrepareImportFiles() error {
	// Map resource types to their canonical actionIDs
	resourceTypes := map[string]string{
		"llm":    "@localproject/llm:1.0.0",
		"client": "@localproject/http:1.0.0",
		"exec":   "@localproject/exec:1.0.0",
		"python": "@localproject/python:1.0.0",
		"data":   "@localproject/data:1.0.0",
	}

	for key, actionID := range resourceTypes {
		// Initialize empty structure for this actionID if it doesn't exist
		// This ensures pklres has the basic structure for imports to work

		// Check if we already have this actionID in pklres
		_, err := dr.PklresHelper.Get(actionID, "initialized")
		if err != nil {
			// If it doesn't exist, create a basic structure
			emptyContent := fmt.Sprintf("// %s resource initialized\n", key)

			// Store the empty structure
			if err := dr.PklresHelper.Set(actionID, "initialized", emptyContent); err != nil {
				return fmt.Errorf("failed to initialize empty %s structure in pklres: %w", key, err)
			}
		}
	}

	return nil
}

// extractActionIDFromContent extracts actionID from PKL file content using a more robust approach
func extractActionIDFromContent(content []byte) (string, error) {
	// First try to find actionID using a more precise regex pattern
	// This pattern looks for actionID = "value" with proper PKL syntax
	actionIDPattern := regexp.MustCompile(`(?m)^\s*actionID\s*=\s*"([^"]+)"\s*$`)
	matches := actionIDPattern.FindSubmatch(content)
	if len(matches) >= 2 {
		return string(matches[1]), nil
	}

	// Fallback to a more flexible pattern if the strict one doesn't match
	fallbackPattern := regexp.MustCompile(`(?i)actionID\s*=\s*"([^"]+)"`)
	matches = fallbackPattern.FindSubmatch(content)
	if len(matches) >= 2 {
		return string(matches[1]), nil
	}

	return "", errors.New("actionID not found in file content")
}

// resolveActionIDCanonically resolves an actionID to its canonical form using the agent reader
func resolveActionIDCanonically(actionID string, wf pklWf.Workflow, agentReader *agent.PklResourceReader) (string, error) {
	// If the actionID is already in canonical form (@agent/action:version), return it
	if strings.HasPrefix(actionID, "@") {
		return actionID, nil
	}

	// Add nil check for Workflow
	if wf == nil {
		return "", errors.New("workflow is nil, cannot resolve action ID")
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", wf.GetAgentID())
	query.Set("version", wf.GetVersion())
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + actionID,
		RawQuery: query.Encode(),
	}

	resovledIDBytes, err := agentReader.Read(uri)
	if err != nil {
		// Fallback to default resolution if agent reader fails
		return "", fmt.Errorf("failed to resolve actionID: %w", err)
	}

	return string(resovledIDBytes), nil
}
