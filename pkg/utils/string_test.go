package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringPtr(t *testing.T) {

	t.Run("ValidString", func(t *testing.T) {
	
		input := "test string"
		result := StringPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})

	t.Run("EmptyString", func(t *testing.T) {
	
		input := ""
		result := StringPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})
}

func TestBoolPtr(t *testing.T) {

	t.Run("True", func(t *testing.T) {
	
		result := BoolPtr(true)
		assert.NotNil(t, result)
		assert.True(t, *result)
	})

	t.Run("False", func(t *testing.T) {
	
		result := BoolPtr(false)
		assert.NotNil(t, result)
		assert.False(t, *result)
	})
}

func TestContainsString(t *testing.T) {
	slice := []string{"one", "Two", "three"}
	assert.True(t, ContainsString(slice, "Two"))
	assert.False(t, ContainsString(slice, "two"))
	assert.True(t, ContainsStringInsensitive(slice, "two"))
	assert.False(t, ContainsStringInsensitive(slice, "four"))
}

func TestContainsStringInsensitive(t *testing.T) {

	tests := []struct {
		name     string
		slice    []string
		target   string
		expected bool
	}{
		{
			name:     "StringFoundCaseInsensitive",
			slice:    []string{"Apple", "Banana", "Cherry"},
			target:   "apple",
			expected: true,
		},
		{
			name:     "StringNotFound",
			slice:    []string{"Apple", "Banana", "Cherry"},
			target:   "orange",
			expected: false,
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			target:   "apple",
			expected: false,
		},
		{
			name:     "MixedCase",
			slice:    []string{"ApPlE", "BaNaNa", "ChErRy"},
			target:   "apple",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		
			result := ContainsStringInsensitive(tt.slice, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeDerefString(t *testing.T) {
	var ptr *string
	assert.Equal(t, "", SafeDerefString(ptr))
	val := "value"
	ptr = &val
	assert.Equal(t, "value", SafeDerefString(ptr))
}

func TestSafeDerefBool(t *testing.T) {

	t.Run("True", func(t *testing.T) {
	
		input := true
		result := SafeDerefBool(&input)
		assert.True(t, result)
	})

	t.Run("False", func(t *testing.T) {
	
		input := false
		result := SafeDerefBool(&input)
		assert.False(t, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
	
		var input *bool
		result := SafeDerefBool(input)
		assert.False(t, result)
	})
}

func TestSafeDerefSlice(t *testing.T) {

	t.Run("ValidSlice", func(t *testing.T) {
	
		input := []string{"a", "b", "c"}
		result := SafeDerefSlice(&input)
		assert.Equal(t, input, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
	
		var input *[]string
		result := SafeDerefSlice(input)
		assert.Empty(t, result)
	})
}

func TestSafeDerefMap(t *testing.T) {

	t.Run("ValidMap", func(t *testing.T) {
	
		input := map[string]int{"a": 1, "b": 2}
		result := SafeDerefMap(&input)
		assert.Equal(t, input, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
	
		var input *map[string]int
		result := SafeDerefMap(input)
		assert.Empty(t, result)
	})
}

func TestTruncateString(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz"
	assert.Equal(t, s, TruncateString(s, len(s)))
	assert.Equal(t, "abc...", TruncateString(s, 6))
}
