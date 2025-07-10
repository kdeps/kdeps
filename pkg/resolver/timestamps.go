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
	
	switch resourceType {
	case "exec":
		// Try to load from pklres first, fallback to LoadResource method
		return dr.LoadResource(dr.Context, pklPath, ExecResource)
	case "python":
		return dr.LoadResource(dr.Context, pklPath, PythonResource)
	case "llm":
		return dr.LoadResource(dr.Context, pklPath, LLMResource)
	case "client":
		return dr.LoadResource(dr.Context, pklPath, HTTPResource)
	default:
		return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}
}

// getResourceTimestamp retrieves the timestamp for a specific resource from the given PKL result.
func getResourceTimestamp(resourceID string, pklRes interface{}) (*pkl.Duration, error) {
	switch res := pklRes.(type) {
	case *pklExec.ExecImpl:
		// ExecImpl resources are of type *ResourceExec
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklPython.PythonImpl:
		// PythonImpl resources are of type *ResourcePython
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklLLM.LLMImpl:
		// LLMImpl resources are of type *ResourceChat
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklHTTP.HTTPImpl:
		// HTTPImpl resources are of type *ResourceHTTPClient
		if resource, exists := (*res.GetResources())[resourceID]; exists {
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
		return pkl.Duration{}, err
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

// WaitForTimestampChange waits until the timestamp for the specified resourceID changes from the provided previous timestamp.
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, previousTimestamp pkl.Duration, timeout time.Duration, resourceType string) error {
	startTime := time.Now()
	lastSeenTimestamp := previousTimestamp

	for {
		elapsed := time.Since(startTime)
		// Calculate remaining time correctly for logging
		remaining := timeout - elapsed
		formattedRemaining := formatDuration(remaining)
		dr.Logger.Infof("action '%s' will timeout in '%s'", resourceID, formattedRemaining)

		// Check if elapsed time meets or exceeds the timeout
		if timeout > 0 && remaining < 0 {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceID)
		}

		currentTimestamp, err := dr.GetCurrentTimestamp(resourceID, resourceType)
		if err != nil {
			return fmt.Errorf("failed to get current timestamp for resource %s: %w", resourceID, err)
		}

		if currentTimestamp != previousTimestamp && currentTimestamp == lastSeenTimestamp {
			elapsedTime := time.Since(startTime)
			dr.Logger.Infof("resource '%s' (type: %s) completed in %s", resourceID, resourceType, formatDuration(elapsedTime))
			return nil
		}
		lastSeenTimestamp = currentTimestamp

		time.Sleep(1000 * time.Millisecond)
	}
}
