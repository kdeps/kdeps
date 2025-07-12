package resolver

import (
	"errors"
	"fmt"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

// getResourceFilePath returns the pklres path for a given resourceType.
func (dr *DependencyResolver) getResourceFilePath(resourceType string) (string, error) {
	// Map resource types to ensure they're valid
	validTypes := map[string]bool{
		"llm":    true,
		"client": true,
		"exec":   true,
		"python": true,
	}

	if !validTypes[resourceType] {
		return "", fmt.Errorf("invalid resourceType %s provided", resourceType)
	}

	// Return pklres path instead of file path
	return dr.PklresHelper.getResourcePath(resourceType), nil
}

// loadPKLFile loads and returns the PKL data from pklres based on resourceType.
func (dr *DependencyResolver) loadPKLFile(resourceType, pklPath string) (interface{}, error) {
	// Since we're using pklres now, we need to retrieve the content and parse it
	// For now, we'll create a temporary file approach or use the existing LoadResource method
	// This is a compatibility layer - ideally we'd refactor to work directly with content

	// Extract the resource ID from the pklres path
	// pklPath format: "pklres:///requestID?type=resourceType"
	// We need to get the content for the specific resource type
	_, err := dr.PklresHelper.retrievePklContent(resourceType, "")
	if err != nil {
		// If no content exists, return an empty structure
		switch resourceType {
		case "exec":
			return &pklExec.ExecImpl{Resources: make(map[string]*pklExec.ResourceExec)}, nil
		case "python":
			return &pklPython.PythonImpl{Resources: make(map[string]*pklPython.ResourcePython)}, nil
		case "llm":
			return &pklLLM.LLMImpl{Resources: make(map[string]*pklLLM.ResourceChat)}, nil
		case "client":
			return &pklHTTP.HTTPImpl{Resources: make(map[string]*pklHTTP.ResourceHTTPClient)}, nil
		default:
			return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
		}
	}

	// For now, we'll return empty structures since we're storing individual resources
	// In a more sophisticated implementation, we'd parse the content and return the proper structure
	switch resourceType {
	case "exec":
		return &pklExec.ExecImpl{Resources: make(map[string]*pklExec.ResourceExec)}, nil
	case "python":
		return &pklPython.PythonImpl{Resources: make(map[string]*pklPython.ResourcePython)}, nil
	case "llm":
		return &pklLLM.LLMImpl{Resources: make(map[string]*pklLLM.ResourceChat)}, nil
	case "client":
		return &pklHTTP.HTTPImpl{Resources: make(map[string]*pklHTTP.ResourceHTTPClient)}, nil
	default:
		return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}
}

// getResourceTimestamp retrieves the timestamp for a specific resource from the given PKL result.
func getResourceTimestamp(resourceID string, pklRes interface{}) (*pkl.Duration, error) {
	switch res := pklRes.(type) {
	case *pklExec.ExecImpl:
		// ExecImpl resources are of type *ResourceExec
		if resource, exists := res.GetResources()[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklPython.PythonImpl:
		// PythonImpl resources are of type *ResourcePython
		if resource, exists := res.GetResources()[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklLLM.LLMImpl:
		// LLMImpl resources are of type *ResourceChat
		if resource, exists := res.GetResources()[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklHTTP.HTTPImpl:
		// HTTPImpl resources are of type *ResourceHTTPClient
		if resource, exists := res.GetResources()[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	default:
		return nil, errors.New("unknown PKL result type")
	}

	// If the resource does not exist, return an error
	return nil, fmt.Errorf("resource ID %s does not exist in the file", resourceID)
}

// GetCurrentTimestamp retrieves the current timestamp for the given resourceID and resourceType.
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID, resourceType string) (pkl.Duration, error) {
	pklPath, err := dr.getResourceFilePath(resourceType)
	if err != nil {
		return pkl.Duration{}, err
	}

	pklRes, err := dr.loadPKLFile(resourceType, pklPath)
	if err != nil {
		return pkl.Duration{}, fmt.Errorf("failed to load %s PKL file: %w", resourceType, err)
	}

	timestamp, err := getResourceTimestamp(resourceID, pklRes)
	if err != nil {
		// If the resource doesn't exist yet, return a default timestamp (current time)
		// This happens when a resource is being executed for the first time
		return pkl.Duration{
			Value: float64(time.Now().Unix()),
			Unit:  pkl.Nanosecond,
		}, nil
	}

	return *timestamp, nil
}

// formatDuration converts a time.Duration into a human-friendly string.
// It prints hours, minutes, and seconds when appropriate.
func formatDuration(d time.Duration) string {
	secondsTotal := int(d.Seconds())
	hours := secondsTotal / 3600
	minutes := (secondsTotal % 3600) / 60
	seconds := secondsTotal % 60

	switch {
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

// WaitForTimestampChange waits for the timestamp to change for the given resourceID and resourceType.
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, initialTimestamp pkl.Duration, timeout time.Duration, resourceType string) error {
	startTime := time.Now()
	timeoutDuration := timeout

	for {
		// Check if we've exceeded the timeout
		if timeout > 0 && time.Since(startTime) > timeoutDuration {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceID)
		}

		// Get current timestamp
		currentTimestamp, err := dr.GetCurrentTimestamp(resourceID, resourceType)
		if err != nil {
			// Log the error but continue waiting
			dr.Logger.Debug("Error getting current timestamp, continuing to wait", "error", err, "resourceID", resourceID, "resourceType", resourceType)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Check if timestamp has changed
		// Use a more robust comparison that accounts for potential precision issues
		timestampDiff := currentTimestamp.Value - initialTimestamp.Value
		if timestampDiff > 0 { // Allow for any positive difference
			dr.Logger.Debug("Timestamp change detected",
				"resourceID", resourceID,
				"resourceType", resourceType,
				"initialTimestamp", initialTimestamp.Value,
				"currentTimestamp", currentTimestamp.Value,
				"difference", timestampDiff)
			return nil
		}

		// Wait a bit before checking again
		time.Sleep(100 * time.Millisecond)
	}
}
