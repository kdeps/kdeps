package utils

import (
	"fmt"
	"strings"
)

// ShouldSkip checks if any condition in the list is true or the string "true".
func ShouldSkip(conditions *[]interface{}) bool {
	for _, condition := range *conditions {
		switch v := condition.(type) {
		case bool:
			if v {
				return true // Skip if any condition is true
			}
		case string:
			if strings.ToLower(v) == "true" {
				return true // Skip if any condition is "true" (case-insensitive)
			}
		}
	}
	return false
}

// AllConditionsMet checks if all conditions in the list are true or the string "true".
func AllConditionsMet(conditions *[]interface{}) bool {
	for _, condition := range *conditions {
		switch v := condition.(type) {
		case bool:
			if !v {
				return false // Return false if any condition is false
			}
		case string:
			if strings.ToLower(v) != "true" {
				return false // Return false if any condition is not "true" (case-insensitive)
			}
		default:
			return false // Return false for unsupported types
		}
	}
	return true // All conditions met
}

// AllConditionsMetWithDetails checks if all conditions are met and returns details about failures.
func AllConditionsMetWithDetails(conditions *[]interface{}) (bool, []string) {
	var failedConditions []string

	for i, condition := range *conditions {
		switch v := condition.(type) {
		case bool:
			if !v {
				failedConditions = append(failedConditions, fmt.Sprintf("condition %d failed: expected true, got false", i+1))
			}
		case string:
			if strings.ToLower(v) != "true" {
				failedConditions = append(failedConditions, fmt.Sprintf("condition %d failed: expected 'true', got '%s'", i+1, v))
			}
		default:
			failedConditions = append(failedConditions, fmt.Sprintf("condition %d failed: unsupported value type %T", i+1, v))
		}
	}

	return len(failedConditions) == 0, failedConditions
}
