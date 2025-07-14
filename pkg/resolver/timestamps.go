package resolver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

// getResourcePath returns the pklres path for a given resourceType.
func (dr *DependencyResolver) getResourcePath(resourceType string) (string, error) {
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

	// Check if PklresHelper is initialized
	if dr.PklresHelper == nil {
		return "", fmt.Errorf("PklresHelper is not initialized")
	}

	// Return pklres path
	return dr.PklresHelper.getResourcePath(resourceType), nil
}

// loadPKLData loads and returns the PKL data from pklres based on resourceType.
func (dr *DependencyResolver) loadPKLData(resourceType, pklPath string) (interface{}, error) {
	// Retrieve all resources of this type from pklres
	// and build the complete structure that the timestamp lookup expects

	// Retrieve all resources of this type from pklres
	resources, err := dr.PklresHelper.retrieveAllResourcesForType(resourceType)
	if err != nil {
		// If no content exists, return an error instead of an empty structure
		return nil, fmt.Errorf("Cannot find module: %w", err)
	}

	// Build the appropriate structure based on resource type
	dr.Logger.Debug("loadPKLData: resource keys", "resourceType", resourceType, "keys", resources)
	switch resourceType {
	case "exec":
		execResources := make(map[string]*pklExec.ResourceExec)
		for resourceID := range resources {
			// Parse the PKL content to extract timestamp
			// For now, we'll create a basic structure with current timestamp
			// In a full implementation, we'd parse the actual PKL content
			execResources[resourceID] = &pklExec.ResourceExec{
				Timestamp: &pkl.Duration{Value: float64(time.Now().UnixNano())},
			}
		}
		return &pklExec.ExecImpl{Resources: execResources}, nil
	case "python":
		pythonResources := make(map[string]*pklPython.ResourcePython)
		for resourceID := range resources {
			pythonResources[resourceID] = &pklPython.ResourcePython{
				Timestamp: &pkl.Duration{Value: float64(time.Now().UnixNano())},
			}
		}
		return &pklPython.PythonImpl{Resources: pythonResources}, nil
	case "llm":
		llmResources := make(map[string]*pklLLM.ResourceChat)
		for resourceID := range resources {
			llmResources[resourceID] = &pklLLM.ResourceChat{
				Timestamp: &pkl.Duration{Value: float64(time.Now().UnixNano())},
			}
		}
		return &pklLLM.LLMImpl{Resources: llmResources}, nil
	case "client":
		clientResources := make(map[string]*pklHTTP.ResourceHTTPClient)
		for resourceID, content := range resources {
			// Debug: print the PKL content we're trying to parse
			dr.Logger.Debug("loadPKLData: parsing PKL content", "resourceID", resourceID, "contentLength", len(content), "content", content)

			// Parse the PKL content to extract the actual timestamp
			// The content should contain a Timestamp field like: Timestamp = 1.7523852347917573e+18.ns
			timestamp := parseTimestampFromPklContent(content)
			if timestamp == nil {
				// Fallback to current timestamp if parsing fails
				timestamp = &pkl.Duration{Value: float64(time.Now().UnixNano())}
				dr.Logger.Debug("loadPKLData: failed to parse timestamp, using fallback", "resourceID", resourceID, "fallbackTimestamp", timestamp.Value)
			} else {
				dr.Logger.Debug("loadPKLData: successfully parsed timestamp", "resourceID", resourceID, "timestamp", timestamp.Value)
			}
			clientResources[resourceID] = &pklHTTP.ResourceHTTPClient{
				Timestamp: timestamp,
			}
		}
		// Debug: print the resources we found
		dr.Logger.Debug("loadPKLData: found client resources", "count", len(clientResources), "resources", clientResources)
		return &pklHTTP.HTTPImpl{Resources: clientResources}, nil
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
	return nil, fmt.Errorf("resource ID %s does not exist in pklres", resourceID)
}

// GetCurrentTimestamp retrieves the current timestamp for the given resourceID and resourceType.
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID, resourceType string) (pkl.Duration, error) {
	pklPath, err := dr.getResourcePath(resourceType)
	if err != nil {
		dr.Logger.Error("GetCurrentTimestamp: failed to get resource path", "resourceID", resourceID, "resourceType", resourceType, "error", err)
		return pkl.Duration{}, err
	}

	pklRes, err := dr.loadPKLData(resourceType, pklPath)
	if err != nil {
		dr.Logger.Error("GetCurrentTimestamp: failed to load PKL data", "resourceID", resourceID, "resourceType", resourceType, "error", err)
		return pkl.Duration{}, fmt.Errorf("failed to load %s PKL data from pklres: %w", resourceType, err)
	}

	timestamp, err := getResourceTimestamp(resourceID, pklRes)
	if err != nil {
		dr.Logger.Error("GetCurrentTimestamp: failed to get resource timestamp", "resourceID", resourceID, "resourceType", resourceType, "error", err)
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

// WaitForTimestampChange waits for the timestamp to change for the given resourceID and resourceType.
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, initialTimestamp pkl.Duration, timeout time.Duration, resourceType string) error {
	startTime := time.Now()
	timeoutDuration := timeout
	lastLogTime := startTime
	logInterval := 5 * time.Second // Log only every 5 seconds to reduce spam

	dr.Logger.Debug("WaitForTimestampChange: starting", "resourceID", resourceID, "resourceType", resourceType, "timeout", timeoutDuration)

	for {
		// Check if we've exceeded the timeout
		if timeout > 0 && time.Since(startTime) > timeoutDuration {
			dr.Logger.Error("WaitForTimestampChange: timeout exceeded", "resourceID", resourceID, "resourceType", resourceType, "elapsed", formatDuration(time.Since(startTime)), "timeout", formatDuration(timeoutDuration))
			return fmt.Errorf("timeout exceeded while waiting for timestamp change for resource ID %s", resourceID)
		}

		// Get current timestamp
		currentTimestamp, err := dr.GetCurrentTimestamp(resourceID, resourceType)
		if err != nil {
			// Check the full error chain for 'Cannot find module'
			unwrapped := err
			for unwrapped != nil {
				if strings.Contains(unwrapped.Error(), "Cannot find module") {
					dr.Logger.Error("WaitForTimestampChange: module not found error", "resourceID", resourceID, "resourceType", resourceType, "error", unwrapped)
					return unwrapped
				}
				if wrapped, ok := unwrapped.(interface{ Unwrap() error }); ok {
					unwrapped = wrapped.Unwrap()
				} else {
					break
				}
			}
			// For any error, return immediately (fail fast)
			dr.Logger.Error("WaitForTimestampChange: failed to get current timestamp", "resourceID", resourceID, "resourceType", resourceType, "error", err)
			return err
		}

		// Check if timestamp has changed
		timestampDiff := currentTimestamp.Value - initialTimestamp.Value
		if timestampDiff > 0 { // Allow for any positive difference
			// Only log timestamp changes for debugging, not for normal operations
			if dr.Logger.IsDebugEnabled() {
				dr.Logger.Debug("WaitForTimestampChange: timestamp change detected",
					"resourceID", resourceID,
					"resourceType", resourceType,
					"difference", timestampDiff,
					"elapsed", formatDuration(time.Since(startTime)))
			}
			return nil
		}

		// Log progress only every 5 seconds to reduce spam
		elapsed := time.Since(startTime)
		if elapsed > lastLogTime.Sub(startTime)+logInterval {
			// Only log if timeout is more than 10 seconds to avoid spam for short operations
			if timeoutDuration > 10*time.Second {
				dr.Logger.Debug("WaitForTimestampChange: still waiting",
					"resourceID", resourceID,
					"resourceType", resourceType,
					"elapsed", formatDuration(elapsed),
					"remaining", formatDuration(timeoutDuration-elapsed))
			}
			lastLogTime = time.Now()
		}

		// Wait a bit before checking again
		time.Sleep(100 * time.Millisecond)
	}
}

// parseTimestampFromPklContent extracts the timestamp from PKL content
func parseTimestampFromPklContent(content string) *pkl.Duration {
	// Look for Timestamp = <value>.<unit> pattern
	// Example: Timestamp = 1.7523852347917573e+18.ns
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Timestamp = ") {
			// Extract the timestamp value
			timestampStr := strings.TrimPrefix(line, "Timestamp = ")

			// Parse the value and unit
			if strings.Contains(timestampStr, ".") {
				lastDot := strings.LastIndex(timestampStr, ".")
				if lastDot > 0 && lastDot < len(timestampStr)-1 {
					valueStr := timestampStr[:lastDot]
					unitStr := timestampStr[lastDot+1:]

					// Parse the value
					if value, err := parseFloat(valueStr); err == nil {
						// Determine the unit
						var unit pkl.DurationUnit
						switch {
						case strings.Contains(unitStr, "ns"):
							unit = pkl.Nanosecond
						case strings.Contains(unitStr, "us"):
							unit = pkl.Microsecond
						case strings.Contains(unitStr, "ms"):
							unit = pkl.Millisecond
						case strings.Contains(unitStr, "s"):
							unit = pkl.Second
						case strings.Contains(unitStr, "m"):
							unit = pkl.Minute
						case strings.Contains(unitStr, "h"):
							unit = pkl.Hour
						default:
							unit = pkl.Nanosecond // default
						}

						return &pkl.Duration{
							Value: value,
							Unit:  unit,
						}
					}
				}
			}
		}
	}
	return nil
}

// parseFloat parses a float value, handling scientific notation
func parseFloat(s string) (float64, error) {
	// Remove any non-numeric characters except for . and e/E
	clean := strings.TrimSpace(s)
	return strconv.ParseFloat(clean, 64)
}
