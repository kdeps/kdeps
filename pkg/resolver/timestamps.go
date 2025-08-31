package resolver

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

// getResourceFilePath returns the file path for a given resourceType.
func (dr *DependencyResolver) getResourceFilePath(resourceType string) (string, error) {
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "llm", dr.RequestID+"__llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "client", dr.RequestID+"__client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "exec", dr.RequestID+"__exec_output.pkl"),
		"python": filepath.Join(dr.ActionDir, "python", dr.RequestID+"__python_output.pkl"),
	}

	pklPath, exists := files[resourceType]
	if !exists {
		return "", fmt.Errorf("invalid resourceType %s provided", resourceType)
	}
	return pklPath, nil
}

// loadPKLFile loads and returns the PKL file based on resourceType.
func (dr *DependencyResolver) loadPKLFile(resourceType, pklPath string) (interface{}, error) {
	switch resourceType {
	case "exec":
		result, err := pklExec.LoadFromPath(dr.Context, pklPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load exec PKL file %s: %w", pklPath, err)
		}
		return result, nil
	case "python":
		result, err := pklPython.LoadFromPath(dr.Context, pklPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load python PKL file %s: %w", pklPath, err)
		}
		return result, nil
	case "llm":
		result, err := pklLLM.LoadFromPath(dr.Context, pklPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load llm PKL file %s: %w", pklPath, err)
		}
		return result, nil
	case "client":
		result, err := pklHTTP.LoadFromPath(dr.Context, pklPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load http PKL file %s: %w", pklPath, err)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}
}

// getResourceTimestamp retrieves the timestamp for a specific resource from the given PKL result.
func getResourceTimestamp(resourceID string, pklRes interface{}) (*pkl.Duration, error) {
	if pklRes == nil {
		return nil, fmt.Errorf("PKL result is nil for resource ID %s", resourceID)
	}

	switch res := pklRes.(type) {
	case *pklExec.ExecImpl:
		// ExecImpl resources are of type *ResourceExec
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for exec resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case pklExec.ExecImpl:
		// Handle value type as well
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for exec resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklPython.PythonImpl:
		// PythonImpl resources are of type *ResourcePython
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for python resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case pklPython.PythonImpl:
		// Handle value type as well
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for python resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklLLM.LLMImpl:
		// LLMImpl resources are of type *ResourceChat
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for llm resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case pklLLM.LLMImpl:
		// Handle value type as well
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for llm resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case *pklHTTP.HTTPImpl:
		// HTTPImpl resources are of type *ResourceHTTPClient
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for http resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	case pklHTTP.HTTPImpl:
		// Handle value type as well
		if resource, exists := (*res.GetResources())[resourceID]; exists {
			if resource.Timestamp == nil {
				return nil, fmt.Errorf("timestamp for http resource ID %s is nil", resourceID)
			}
			return resource.Timestamp, nil
		}
	default:
		return nil, fmt.Errorf("unknown PKL result type %T for resource ID %s", pklRes, resourceID)
	}

	// If the resource does not exist, return an error
	return nil, fmt.Errorf("resource ID %s does not exist in the PKL file", resourceID)
}

// GetCurrentTimestamp retrieves the current timestamp for the given resourceID and resourceType.
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID, resourceType string) (pkl.Duration, error) {
	pklPath, err := dr.getResourceFilePath(resourceType)
	if err != nil {
		// If we can't get the file path, return a default timestamp for workflow resources
		dr.Logger.Debug("could not get file path for timestamp, returning default", "resourceID", resourceID, "resourceType", resourceType, "error", err)
		return pkl.Duration{}, nil
	}

	// Check if the file exists before trying to load it
	if _, err := dr.Fs.Stat(pklPath); err != nil {
		// If the timestamp file doesn't exist, return a default timestamp
		// This can happen for workflow resources that don't have timestamp tracking
		dr.Logger.Debug("timestamp file does not exist, returning default", "path", pklPath, "resourceID", resourceID, "resourceType", resourceType)
		return pkl.Duration{}, nil
	}

	pklRes, err := dr.loadPKLFile(resourceType, pklPath)
	if err != nil {
		// If we can't load the PKL file, return a default timestamp
		dr.Logger.Debug("failed to load PKL file for timestamp, returning default", "path", pklPath, "resourceID", resourceID, "resourceType", resourceType, "error", err)
		return pkl.Duration{}, nil
	}

	timestamp, err := getResourceTimestamp(resourceID, pklRes)
	if err != nil {
		// If we can't get the timestamp from the loaded file, return a default timestamp
		dr.Logger.Debug("failed to get timestamp from PKL file, returning default", "resourceID", resourceID, "resourceType", resourceType, "error", err)
		return pkl.Duration{}, nil
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

	for {
		elapsed := time.Since(startTime)

		// Check if elapsed time meets or exceeds the timeout first
		if timeout > 0 && elapsed >= timeout {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceID)
		}

		// Calculate remaining time correctly for logging (only if not timed out)
		remaining := timeout - elapsed
		if timeout > 0 {
			formattedRemaining := formatDuration(remaining)
			dr.Logger.Infof("action '%s' will timeout in '%s'", resourceID, formattedRemaining)
		} else {
			dr.Logger.Infof("action '%s' waiting for completion (no timeout)", resourceID)
		}

		currentTimestamp, err := dr.GetCurrentTimestamp(resourceID, resourceType)
		if err != nil {
			return fmt.Errorf("failed to get current timestamp for resource %s: %w", resourceID, err)
		}

		// If the timestamp has changed from the initial previous timestamp, the resource has completed
		if currentTimestamp != previousTimestamp {
			elapsedTime := time.Since(startTime)
			dr.Logger.Infof("resource '%s' (type: %s) completed in %s", resourceID, resourceType, formatDuration(elapsedTime))
			return nil
		}

		// Sleep for a shorter interval, but not longer than the remaining timeout
		sleepDuration := 100 * time.Millisecond
		if timeout > 0 {
			remaining := timeout - elapsed
			if remaining > 0 && remaining < sleepDuration {
				sleepDuration = remaining
			}
		}
		time.Sleep(sleepDuration)
	}
}
