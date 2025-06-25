package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSONSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid json object",
			input:    `{"key": "value"}`,
			expected: true,
		},
		{
			name:     "invalid json",
			input:    `{invalid json`,
			expected: false,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: false,
		},
		{
			name:     "plain text",
			input:    `hello world`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFixJSONSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid json no changes needed",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "empty string",
			input:    ``,
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FixJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBase64EncodedSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid base64",
			input:    "SGVsbG8=",
			expected: true,
		},
		{
			name:     "invalid base64",
			input:    "Hello World!",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBase64Encoded(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeBase64StringSimple(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "Hello",
			expected: "SGVsbG8=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeBase64String(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
