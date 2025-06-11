package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodePklMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    *map[string]string
		expected string
	}{
		{
			name:     "NilMap",
			input:    nil,
			expected: "{}\n",
		},
		{
			name:     "EmptyMap",
			input:    &map[string]string{},
			expected: "{\n    }\n",
		},
		{
			name: "SingleEntry",
			input: &map[string]string{
				"key": "value",
			},
			expected: "{\n      [\"key\"] = \"dmFsdWU=\"\n    }\n",
		},
		{
			name: "SpecialCharacters",
			input: &map[string]string{
				"key with spaces": "value with \"quotes\"",
			},
			expected: "{\n      [\"key with spaces\"] = \"dmFsdWUgd2l0aCAicXVvdGVzIg==\"\n    }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := EncodePklMap(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Additional check for maps with multiple entries where ordering is not deterministic.
	t.Run("MultipleEntries", func(t *testing.T) {
		input := &map[string]string{"key1": "value1", "key2": "value2"}
		result := EncodePklMap(input)
		assert.Contains(t, result, "[\"key1\"] = \"dmFsdWUx\"")
		assert.Contains(t, result, "[\"key2\"] = \"dmFsdWUy\"")
	})
}

func TestEncodePklSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    *[]string
		expected string
	}{
		{
			name:     "NilSlice",
			input:    nil,
			expected: "{}\n",
		},
		{
			name:     "EmptySlice",
			input:    &[]string{},
			expected: "{\n    }\n",
		},
		{
			name:     "SingleEntry",
			input:    &[]string{"value"},
			expected: "{\n      \"dmFsdWU=\"\n    }\n",
		},
		{
			name:     "SpecialCharacters",
			input:    &[]string{"value with \"quotes\""},
			expected: "{\n      \"dmFsdWUgd2l0aCAicXVvdGVzIg==\"\n    }\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := EncodePklSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "EmptyString",
			input:    "",
			expected: "",
		},
		{
			name:     "SimpleString",
			input:    "test",
			expected: "dGVzdA==",
		},
		{
			name:     "AlreadyEncoded",
			input:    "dGVzdA==",
			expected: "dGVzdA==",
		},
		{
			name:     "SpecialCharacters",
			input:    "test with spaces and \"quotes\"",
			expected: "dGVzdCB3aXRoIHNwYWNlcyBhbmQgInF1b3RlcyI=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := EncodeValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
