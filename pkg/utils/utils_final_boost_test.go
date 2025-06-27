package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtilsFinalBoost(t *testing.T) {
	// Test the remaining 0% coverage utility functions

	t.Run("ContainsString", func(t *testing.T) {
		// Test ContainsString function - has 0.0% coverage
		slice := []string{"apple", "banana", "cherry"}

		// Test positive cases
		assert.True(t, ContainsString(slice, "apple"))
		assert.True(t, ContainsString(slice, "banana"))
		assert.True(t, ContainsString(slice, "cherry"))

		// Test negative cases
		assert.False(t, ContainsString(slice, "orange"))
		assert.False(t, ContainsString(slice, ""))
		assert.False(t, ContainsString(slice, "APPLE")) // Case sensitive

		// Test empty slice
		emptySlice := []string{}
		assert.False(t, ContainsString(emptySlice, "anything"))

		// Test nil slice
		var nilSlice []string
		assert.False(t, ContainsString(nilSlice, "anything"))
	})

	t.Run("ContainsStringInsensitive", func(t *testing.T) {
		// Test ContainsStringInsensitive function - has 0.0% coverage
		slice := []string{"Apple", "BANANA", "cherry"}

		// Test positive cases (case insensitive)
		assert.True(t, ContainsStringInsensitive(slice, "apple"))
		assert.True(t, ContainsStringInsensitive(slice, "APPLE"))
		assert.True(t, ContainsStringInsensitive(slice, "banana"))
		assert.True(t, ContainsStringInsensitive(slice, "BANANA"))
		assert.True(t, ContainsStringInsensitive(slice, "cherry"))
		assert.True(t, ContainsStringInsensitive(slice, "CHERRY"))

		// Test negative cases
		assert.False(t, ContainsStringInsensitive(slice, "orange"))
		assert.False(t, ContainsStringInsensitive(slice, ""))

		// Test empty slice
		emptySlice := []string{}
		assert.False(t, ContainsStringInsensitive(emptySlice, "anything"))
	})

	t.Run("SafeDerefString", func(t *testing.T) {
		// Test SafeDerefString function - has 0.0% coverage

		// Test with valid pointer
		str := "test value"
		ptr := &str
		result := SafeDerefString(ptr)
		assert.Equal(t, "test value", result)

		// Test with nil pointer
		var nilPtr *string
		result = SafeDerefString(nilPtr)
		assert.Equal(t, "", result) // Should return empty string for nil

		// Test with empty string pointer
		emptyStr := ""
		emptyPtr := &emptyStr
		result = SafeDerefString(emptyPtr)
		assert.Equal(t, "", result)
	})

	t.Run("SafeDerefBool", func(t *testing.T) {
		// Test SafeDerefBool function - has 0.0% coverage

		// Test with valid pointer to true
		trueVal := true
		truePtr := &trueVal
		result := SafeDerefBool(truePtr)
		assert.True(t, result)

		// Test with valid pointer to false
		falseVal := false
		falsePtr := &falseVal
		result = SafeDerefBool(falsePtr)
		assert.False(t, result)

		// Test with nil pointer
		var nilPtr *bool
		result = SafeDerefBool(nilPtr)
		assert.False(t, result) // Should return false for nil
	})

	t.Run("SafeDerefSlice", func(t *testing.T) {
		// Test SafeDerefSlice function - has 0.0% coverage

		// Test with valid slice pointer
		slice := []string{"item1", "item2", "item3"}
		ptr := &slice
		result := SafeDerefSlice(ptr)
		assert.Equal(t, slice, result)
		assert.Len(t, result, 3)

		// Test with nil pointer - should return empty slice, not nil
		var nilPtr *[]string
		result = SafeDerefSlice(nilPtr)
		assert.NotNil(t, result) // Should return empty slice, not nil
		assert.Len(t, result, 0)

		// Test with empty slice pointer
		emptySlice := []string{}
		emptyPtr := &emptySlice
		result = SafeDerefSlice(emptyPtr)
		assert.Equal(t, emptySlice, result)
		assert.Len(t, result, 0)
	})

	t.Run("SafeDerefMap", func(t *testing.T) {
		// Test SafeDerefMap function - has 0.0% coverage

		// Test with valid map pointer
		m := map[string]string{"key1": "value1", "key2": "value2"}
		ptr := &m
		result := SafeDerefMap(ptr)
		assert.Equal(t, m, result)
		assert.Len(t, result, 2)

		// Test with nil pointer - should return empty map, not nil
		var nilPtr *map[string]string
		result = SafeDerefMap(nilPtr)
		assert.NotNil(t, result) // Should return empty map, not nil
		assert.Len(t, result, 0)

		// Test with empty map pointer
		emptyMap := map[string]string{}
		emptyPtr := &emptyMap
		result = SafeDerefMap(emptyPtr)
		assert.Equal(t, emptyMap, result)
		assert.Len(t, result, 0)
	})

	t.Run("TruncateString", func(t *testing.T) {
		// Test TruncateString function - has 0.0% coverage

		// Test with string shorter than limit
		short := "hello"
		result := TruncateString(short, 10)
		assert.Equal(t, "hello", result)

		// Test with string equal to limit
		exact := "1234567890"
		result = TruncateString(exact, 10)
		assert.Equal(t, "1234567890", result)

		// Test with string longer than limit
		long := "this is a very long string that should be truncated"
		result = TruncateString(long, 10)
		assert.Equal(t, "this is...", result) // Should be truncated with "..." (10-3=7 chars + ...)

		// Test with empty string
		empty := ""
		result = TruncateString(empty, 10)
		assert.Equal(t, "", result)

		// Test with limit of 0 - maxLength < 3 so return "..."
		result = TruncateString("test", 0)
		assert.Equal(t, "...", result) // maxLength < 3, so return "..."

		// Test with very small limit
		result = TruncateString("test", 2)
		assert.Equal(t, "...", result) // maxLength < 3, so return "..."

		// Test with limit of 3
		result = TruncateString("testing", 3)
		assert.Equal(t, "...", result) // maxLength == 3, but len("testing") > 3, so maxLength < 3 check fails, return "..."

		// Test with limit of 4  
		result = TruncateString("testing", 4)
		assert.Equal(t, "t...", result) // 4-3=1 char + "..."
	})
}
