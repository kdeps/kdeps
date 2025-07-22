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

// createCountdownIndicator creates a visual countdown indicator showing time remaining.
func (dr *DependencyResolver) createCountdownIndicator(elapsed, total time.Duration) string {
	if total <= 0 {
		return "⏳ unlimited"
	}

	// Calculate percentage of time elapsed
	percentage := float64(elapsed) / float64(total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Create a 20-character wide progress bar
	barWidth := 20
	filled := int(percentage * float64(barWidth))

	// Different visual representations based on time remaining
	var bar strings.Builder
	remaining := total - elapsed

	// Choose color/symbol based on remaining time
	var symbol string
	if remaining <= 10*time.Second {
		symbol = "🔴" // Red for critical (≤10s)
	} else if remaining <= 30*time.Second {
		symbol = "🟡" // Yellow for warning (≤30s)
	} else {
		symbol = "🟢" // Green for normal (>30s)
	}

	bar.WriteString(symbol)
	bar.WriteString(" [")

	// Fill the progress bar
	for i := 0; i < barWidth; i++ {
		if i < filled {
			if remaining <= 10*time.Second {
				bar.WriteString("█") // Solid block for critical
			} else if remaining <= 30*time.Second {
				bar.WriteString("▓") // Medium shade for warning
			} else {
				bar.WriteString("▒") // Light shade for normal
			}
		} else {
			bar.WriteString("░") // Very light for unfilled
		}
	}

	bar.WriteString("] ")

	// Add percentage and time remaining
	remainingSeconds := int(remaining.Seconds())
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	bar.WriteString(fmt.Sprintf("%d%% (%ds left)", int((1.0-percentage)*100), remainingSeconds))

	return bar.String()
}

// WaitForTimestampChange waits for the timestamp to change for the given resourceID and resourceType.
func (dr *DependencyResolver) WaitForTimestampChange(resourceID string, initialTimestamp pkl.Duration, timeout time.Duration, resourceType string) error {
	startTime := time.Now()
	timeoutDuration := timeout
	lastLogTime := startTime
	logInterval := 1 * time.Second // Update countdown every second for visual indicator

	dr.Logger.Debug("WaitForTimestampChange: starting", "resourceID", resourceID, "resourceType", resourceType, "timeout", timeoutDuration)

	for {
		elapsed := time.Since(startTime)

		// Check if we've exceeded the timeout
		if timeout > 0 && elapsed > timeoutDuration {
			dr.Logger.Error("WaitForTimestampChange: timeout exceeded", "resourceID", resourceID, "resourceType", resourceType, "elapsed", formatDuration(elapsed), "timeout", formatDuration(timeoutDuration))
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
					"elapsed", formatDuration(elapsed))
			}
			return nil
		}

		// Show graphical countdown indicator every second
		if elapsed > lastLogTime.Sub(startTime)+logInterval {
			// Only show countdown for operations with timeout > 5 seconds to avoid spam
			if timeoutDuration > 5*time.Second {
				remaining := timeoutDuration - elapsed
				if remaining < 0 {
					remaining = 0
				}

				// Create graphical countdown indicator
				countdownBar := dr.createCountdownIndicator(elapsed, timeoutDuration)

				dr.Logger.Info("resource timeout countdown",
					"resourceID", resourceID,
					"step", resourceType,
					"countdown", countdownBar,
					"remaining", formatDuration(remaining))
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
