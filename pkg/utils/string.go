package utils

import "strings"

func StringPtr(s string) *string {
	return &s
}

// ContainsString checks if a string exists in a slice (case-sensitive).
func ContainsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// ContainsStringInsensitive checks if a string exists in a slice (case-insensitive).
func ContainsStringInsensitive(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
