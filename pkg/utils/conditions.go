package utils

import "strings"

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
