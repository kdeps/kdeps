package utils

import (
	"encoding/base64"
	"fmt"
	"unicode/utf8"
)

// Helper function to check if a string is already Base64 encoded.
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

// Helper function to decode a Base64-encoded string.
func DecodeBase64String(encodedStr string) (string, error) {
	if !IsBase64Encoded(encodedStr) {
		return encodedStr, nil
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode Base64 string: %w", err)
	}
	return string(decodedBytes), nil
}

// Helper function to Base64 encode a string.
func EncodeBase64String(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}
