package utils

// This file contains basic base64 detection utilities
// All encoding/decoding functions have been removed as they are no longer needed
// with the new SQLite-based resource storage system

import (
	"encoding/base64"
	"unicode/utf8"
)

// Helper function to check if a string is already Base64 encoded.
// Kept for backward compatibility with any remaining code that might use it.
func IsBase64Encoded(str string) bool {
	// Return false for empty strings
	if str == "" {
		return false
	}

	// Check length of the string
	if len(str)%4 != 0 {
		return false
	}

	// Check if the string contains only Base64 valid characters
	for _, char := range str {
		if !(('A' <= char && char <= 'Z') || ('a' <= char && char <= 'z') ||
			('0' <= char && char <= '9') || char == '+' || char == '/' || char == '=') {
			return false
		}
	}

	// Try decoding the string
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return false
	}

	// Check if the decoded string is valid UTF-8
	return utf8.Valid(decoded)
}
