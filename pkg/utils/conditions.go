package utils

func ShouldSkip(conditions *[]bool) bool {
	for _, condition := range *conditions {
		if condition {
			return true // Skip if any condition is true
		}
	}
	return false
}

// Function to check if all conditions in a preflight check are met
func AllConditionsMet(conditions *[]bool) bool {
	for _, condition := range *conditions {
		if !condition {
			return false // Return false if any condition is not met
		}
	}
	return true // All conditions met
}
