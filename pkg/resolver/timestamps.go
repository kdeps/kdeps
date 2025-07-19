package resolver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
)

const (
	waitTimestampSleep = 100 * time.Millisecond
)

// GetCurrentTimestamp retrieves the current timestamp for the given resourceID and resourceType.
// If the resource doesn't exist in pklres yet (during initial processing), returns a default timestamp.
func (dr *DependencyResolver) GetCurrentTimestamp(resourceID, resourceType string) (pkl.Duration, error) {
	// Use the new generic key-value store approach
	if dr.PklresHelper == nil {
		dr.Logger.Error("GetCurrentTimestamp: PklresHelper not initialized", "resourceID", resourceID, "resourceType", resourceType)
		return pkl.Duration{}, errors.New("PklresHelper not initialized")
	}

	// Try to get the timestamp from the generic store
	timestampStr, err := dr.PklresHelper.Get(resourceID, "timestamp")
	if err != nil || timestampStr == "" {
		// During initial resource processing, the resource may not exist in pklres yet
		// Return a default timestamp instead of failing
		dr.Logger.Debug("GetCurrentTimestamp: resource not in pklres yet, returning default timestamp", "resourceID", resourceID, "resourceType", resourceType, "error", err)
		defaultTimestamp := pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
		return defaultTimestamp, nil
	}

	// Parse the timestamp string
	timestampValue, err := strconv.ParseFloat(timestampStr, 64)
	if err != nil {
		dr.Logger.Error("GetCurrentTimestamp: failed to parse timestamp", "resourceID", resourceID, "resourceType", resourceType, "timestampStr", timestampStr, "error", err)
		return pkl.Duration{}, err
	}

	return pkl.Duration{
		Value: timestampValue,
		Unit:  pkl.Nanosecond,
	}, nil
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
		time.Sleep(waitTimestampSleep)
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
