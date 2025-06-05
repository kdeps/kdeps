package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

// GetResourceFilePath returns the file path for a given resourceType.
func (dr *DependencyResolver) GetResourceFilePath(resourceType ResourceType) (string, error) {
	files := map[ResourceType]string{
		LLMResource:    filepath.Join(dr.ActionDir, "llm", dr.RequestID+"__llm_output.pkl"),
		HTTPResource:   filepath.Join(dr.ActionDir, "client", dr.RequestID+"__client_output.pkl"),
		ExecResource:   filepath.Join(dr.ActionDir, "exec", dr.RequestID+"__exec_output.pkl"),
		PythonResource: filepath.Join(dr.ActionDir, "python", dr.RequestID+"__python_output.pkl"),
	}

	pklPath, exists := files[resourceType]
	if !exists {
		return "", fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}
	return pklPath, nil
}

// LoadPKLFile loads and returns the PKL file based on resourceType.
func (dr *DependencyResolver) LoadPKLFile(resourceType ResourceType, pklPath string) (interface{}, error) {
	// Use mock/test loader if set
	if dr.LoadResourceFunc != nil {
		return dr.LoadResourceFunc(dr.Context, pklPath, resourceType)
	}

	switch resourceType {
	case ExecResource:
		return pklExec.LoadFromPath(dr.Context, pklPath)
	case PythonResource:
		return pklPython.LoadFromPath(dr.Context, pklPath)
	case LLMResource:
		return pklLLM.LoadFromPath(dr.Context, pklPath)
	case HTTPResource:
		return pklHTTP.LoadFromPath(dr.Context, pklPath)
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
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID string, resourceType ResourceType) (pkl.Duration, error) {
	pklPath, err := dr.GetResourceFilePath(resourceType)
	if err != nil {
		return pkl.Duration{}, err
	}

	var pklRes interface{}
	if dr.LoadResourceFunc != nil {
		pklRes, err = dr.LoadResourceFunc(dr.Context, pklPath, resourceType)
	} else {
		pklRes, err = dr.LoadPKLFile(resourceType, pklPath)
	}
	if err != nil {
		return pkl.Duration{}, fmt.Errorf("failed to load %s PKL file: %w", resourceTypeGoName(resourceType), err)
	}

	timestamp, err := getResourceTimestamp(resourceID, pklRes)
	if err != nil {
		return pkl.Duration{}, err
	}

	return *timestamp, nil
}

// resourceTypeGoName returns the Go constant name for a ResourceType.
func resourceTypeGoName(resourceType ResourceType) string {
	switch resourceType {
	case ExecResource:
		return "ExecResource"
	case PythonResource:
		return "PythonResource"
	case LLMResource:
		return "LLMResource"
	case HTTPResource:
		return "HTTPResource"
	default:
		return string(resourceType)
	}
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
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, previousTimestamp pkl.Duration, timeout time.Duration, resourceType ResourceType) error {
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
