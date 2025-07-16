package utils

import (
	"encoding/base64"
	"errors"
	"fmt"
	"unicode/utf8"
)

// IsBase64Encoded checks if a string is already Base64 encoded.
func IsBase64Encoded(s string) bool {
	// Return false for empty strings
	if s == "" {
		return false
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
	if !IsBase64Encoded(s) {
		return s, nil
	}
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

func DecodeBase64IfNeeded(value string) (string, error) {
	if IsBase64Encoded(value) {
		return DecodeBase64String(value)
	}
	return value, nil
}

func EncodeValue(value string) string {
	if !IsBase64Encoded(value) {
		return EncodeBase64String(value)
	}
	return value
}

// EncodeValuePtr handles optional string pointers (like Stderr/Stdout).
func EncodeValuePtr(s *string) *string {
	if s == nil {
		return nil
	}
	encoded := EncodeValue(*s)
	return &encoded
}

func DecodeStringMap(src *map[string]string, fieldType string) (*map[string]string, error) {
	if src == nil {
		return nil, errors.New("source map is nil")
	}
	decoded := make(map[string]string)
	for k, v := range *src {
		decodedVal, err := DecodeBase64IfNeeded(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s %s: %w", fieldType, k, err)
		}
		decoded[k] = decodedVal
	}
	return &decoded, nil
}

func DecodeStringSlice(src *[]string, fieldType string) (*[]string, error) {
	if src == nil {
		return nil, errors.New("source slice is nil")
	}
	decoded := make([]string, len(*src))
	for i, v := range *src {
		decodedVal, err := DecodeBase64IfNeeded(v)
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s index %d: %w", fieldType, i, err)
		}
		decoded[i] = decodedVal
	}
	return &decoded, nil
}
