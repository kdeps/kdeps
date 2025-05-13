package utils

import "strings"

// StringPtr returns a pointer to the provided string.
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to the provided bool.
func BoolPtr(b bool) *bool {
	return &b
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

// SafeDerefString safely dereferences a string pointer, returning an empty string if nil.
func SafeDerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// SafeDerefBool safely dereferences a bool pointer, returning false if nil.
func SafeDerefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// SafeDerefSlice safely dereferences a slice pointer, returning an empty slice if nil.
func SafeDerefSlice[T any](s *[]T) []T {
	if s == nil {
		return []T{}
	}
	return *s
}

// SafeDerefMap safely dereferences a map pointer, returning an empty map if nil.
func SafeDerefMap[K comparable, V any](m *map[K]V) map[K]V {
	if m == nil {
		return make(map[K]V)
	}
	return *m
}
