package resolver

import (
	"fmt"
	"path/filepath"
	"time"

	pklExec "github.com/kdeps/schema/gen/exec"
	pklHttp "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
)

func (dr *DependencyResolver) GetCurrentTimestamp(resourceId string, resourceType string) (uint32, error) {
	// Define file paths based on resource types
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "llm/"+dr.RequestId+"__llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "client/"+dr.RequestId+"__client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "exec/"+dr.RequestId+"__exec_output.pkl"),
	}

	// Check if the resource type is valid and get the corresponding path
	pklPath, exists := files[resourceType]
	if !exists {
		return 0, fmt.Errorf("invalid resourceType %s provided", resourceType)
	}

	// Load the appropriate PKL file based on the resourceType
	switch resourceType {
	case "exec":
		pklRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
		if err != nil {
			return 0, fmt.Errorf("failed to load exec PKL file: %w", err)
		}
		// Dereference the resource map for exec and handle ResourceExec
		existingResources := *pklRes.GetResources()
		if resource, exists := existingResources[resourceId]; exists {
			if resource.Timestamp == nil {
				return 0, fmt.Errorf("timestamp for resource ID %s is nil", resourceId)
			}
			return *resource.Timestamp, nil
		}
	case "llm":
		pklRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
		if err != nil {
			return 0, fmt.Errorf("failed to load llm PKL file: %w", err)
		}
		// Dereference the resource map for llm and handle ResourceChat
		existingResources := *pklRes.GetResources()
		if resource, exists := existingResources[resourceId]; exists {
			if resource.Timestamp == nil {
				return 0, fmt.Errorf("timestamp for resource ID %s is nil", resourceId)
			}
			return *resource.Timestamp, nil
		}
	case "client":
		pklRes, err := pklHttp.LoadFromPath(*dr.Context, pklPath)
		if err != nil {
			return 0, fmt.Errorf("failed to load client PKL file: %w", err)
		}
		// Dereference the resource map for exec and handle ResourceExec
		existingResources := *pklRes.GetResources()
		if resource, exists := existingResources[resourceId]; exists {
			if resource.Timestamp == nil {
				return 0, fmt.Errorf("timestamp for resource ID %s is nil", resourceId)
			}
			return *resource.Timestamp, nil
		}
	default:
		return 0, fmt.Errorf("unsupported resourceType %s provided", resourceType)
	}

	return 0, fmt.Errorf("resource ID %s does not exist in the file", resourceId)
}

// WaitForTimestampChange waits until the timestamp for the specified resource ID changes from the provided previous timestamp.
func (dr *DependencyResolver) WaitForTimestampChange(resourceId string, previousTimestamp uint32, timeout time.Duration,
	resourceType string) error {

	// Map containing the paths for different resource types
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "llm/"+dr.RequestId+"__llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "client/"+dr.RequestId+"__client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "exec/"+dr.RequestId+"__exec_output.pkl"),
	}

	// Retrieve the correct path based on resourceType
	pklPath, exists := files[resourceType]
	if !exists {
		return fmt.Errorf("invalid resourceType %s provided", resourceType)
	}

	// Start the waiting loop
	startTime := time.Now()
	for {
		// If timeout is not zero, check if the timeout has been exceeded
		if timeout > 0 && time.Since(startTime) > timeout {
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceId)
		}

		// Reload the current state of the PKL file and handle based on the resourceType
		switch resourceType {
		case "exec":
			// Load exec type PKL file
			updatedRes, err := pklExec.LoadFromPath(*dr.Context, pklPath)
			if err != nil {
				return fmt.Errorf("failed to reload exec PKL file: %w", err)
			}

			// Get the resource map and check for timestamp changes
			updatedResources := *updatedRes.GetResources() // Dereference to get the map
			if updatedResource, exists := updatedResources[resourceId]; exists {
				// Compare the current timestamp with the previous timestamp
				if updatedResource.Timestamp != nil && *updatedResource.Timestamp != previousTimestamp {
					// Timestamp has changed
					return nil
				}
			} else {
				return fmt.Errorf("resource ID %s does not exist in the exec file", resourceId)
			}

		case "llm":
			// Load llm type PKL file
			updatedRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
			if err != nil {
				return fmt.Errorf("failed to reload llm PKL file: %w", err)
			}

			// Get the resource map and check for timestamp changes
			updatedResources := *updatedRes.GetResources() // Dereference to get the map
			if updatedResource, exists := updatedResources[resourceId]; exists {
				// Compare the current timestamp with the previous timestamp
				if updatedResource.Timestamp != nil && *updatedResource.Timestamp != previousTimestamp {
					// Timestamp has changed
					return nil
				}
			} else {
				return fmt.Errorf("resource ID %s does not exist in the llm file", resourceId)
			}
		case "client":
			// Load client type PKL file
			updatedRes, err := pklHttp.LoadFromPath(*dr.Context, pklPath)
			if err != nil {
				return fmt.Errorf("failed to reload http PKL file: %w", err)
			}

			// Get the resource map and check for timestamp changes
			updatedResources := *updatedRes.GetResources() // Dereference to get the map
			if updatedResource, exists := updatedResources[resourceId]; exists {
				// Compare the current timestamp with the previous timestamp
				if updatedResource.Timestamp != nil && *updatedResource.Timestamp != previousTimestamp {
					// Timestamp has changed
					return nil
				}
			} else {
				return fmt.Errorf("resource ID %s does not exist in the client file", resourceId)
			}
		default:
			return fmt.Errorf("unsupported resourceType %s provided", resourceType)
		}

		// Sleep for a short duration before checking again
		time.Sleep(100 * time.Millisecond)
	}
}
