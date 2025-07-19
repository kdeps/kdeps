package utils

import (
	"encoding/base64"
	"errors"
	"unicode/utf8"
)

// IsBase64Encoded checks if a string is already Base64 encoded.
func IsBase64Encoded(s string) bool {
	// Return true for empty strings (valid base64 encoding of empty byte array)
	if s == "" {
		return true
	}

	// Check length of the string
	if len(s)%4 != 0 {
		return false
	}

	// Check if the string contains only Base64 valid characters
	for _, char := range s {
		if !(('A' <= char && char <= 'Z') || ('a' <= char && char <= 'z') ||
			('0' <= char && char <= '9') || char == '+' || char == '/' || char == '=') {
			return false
		}
	}

	// Try decoding the string
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return false
	}

	// Check if the decoded string is valid UTF-8
	return utf8.Valid(decoded)
}

// DecodeBase64String decodes a Base64-encoded string.
func DecodeBase64String(s string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", errors.New("invalid base64 encoding")
	}
	return string(decodedBytes), nil
}

// EncodeBase64String Base64 encodes a string.
func EncodeBase64String(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// DecodeBase64IfNeeded now returns the string directly without base64 decoding
func DecodeBase64IfNeeded(value string) (string, error) {
	return value, nil
}

// EncodeValue now returns the string directly without base64 encoding
func EncodeValue(value string) string {
	return value
}

// EncodeValuePtr now returns the string pointer directly without base64 encoding
func EncodeValuePtr(s *string) *string {
	return s
}

// DecodeStringMap now returns the map directly without base64 decoding
func DecodeStringMap(src *map[string]string, fieldType string) (*map[string]string, error) {
	if src == nil {
		return nil, errors.New("source map is nil")
	}
	return src, nil
}

// DecodeStringSlice now returns the slice directly without base64 decoding
func DecodeStringSlice(src *[]string, fieldType string) (*[]string, error) {
	if src == nil {
		return nil, errors.New("source slice is nil")
	}
	return src, nil
}
