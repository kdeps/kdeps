package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

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
		return pklExec.LoadFromPath(dr.Context, pklPath)
	case "python":
		return pklPython.LoadFromPath(dr.Context, pklPath)
	case "llm":
		return pklLLM.LoadFromPath(dr.Context, pklPath)
	case "client":
		return pklHTTP.LoadFromPath(dr.Context, pklPath)
	default:
		return nil, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}
}

// getResourceTimestamp retrieves the timestamp for a specific resource from the given PKL result.
func getResourceTimestamp(resourceID string, pklRes interface{}) (*uint32, error) {
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
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID, resourceType string) (uint32, error) {
	pklPath, err := dr.getResourceFilePath(resourceType)
	if err != nil {
		return 0, err
	}

	pklRes, err := dr.loadPKLFile(resourceType, pklPath)
	if err != nil {
		return 0, fmt.Errorf("failed to load %s PKL file: %w", resourceType, err)
	}

	timestamp, err := getResourceTimestamp(resourceID, pklRes)
	if err != nil {
		return 0, err
	}

	return *timestamp, nil
}

// WaitForTimestampChange waits until the timestamp for the specified resourceID changes from the provided previous timestamp.
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, previousTimestamp uint32, timeout time.Duration, resourceType string) error {
	pklPath, err := dr.getResourceFilePath(resourceType)
	if err != nil {
		return err
	}

	startTime := time.Now()

	for {
		// Check if timeout has been exceeded
		if timeout > 0 && time.Since(startTime) > timeout {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceID)
		}

		// Reload the PKL file and check the timestamp
		pklRes, err := dr.loadPKLFile(resourceType, pklPath)
		if err != nil {
			return fmt.Errorf("failed to reload %s PKL file: %w", resourceType, err)
		}

		timestamp, err := getResourceTimestamp(resourceID, pklRes)
		if err != nil {
			return err
		}

		// If the timestamp has changed, return successfully
		if *timestamp != previousTimestamp {
			return nil
		}

		// Sleep before rechecking
		time.Sleep(100 * time.Millisecond)
	}
}
