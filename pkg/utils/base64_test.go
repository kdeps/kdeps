package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBase64Encoded(t *testing.T) {
	t.Parallel()
	t.Run("ValidBase64String", func(t *testing.T) {
		t.Parallel()
		assert.True(t, IsBase64Encoded("U29tZSB2YWxpZCBzdHJpbmc=")) // "Some valid string"
	})

	t.Run("InvalidBase64String", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsBase64Encoded("InvalidString!!!"))
	})

	t.Run("EmptyString", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsBase64Encoded(""))
	})

	t.Run("NonBase64Characters", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsBase64Encoded("Hello@World"))
	})

	t.Run("ValidBase64ButInvalidUTF8", func(t *testing.T) {
		t.Parallel()
		assert.False(t, IsBase64Encoded("////")) // Decodes to invalid UTF-8
	})
}

func TestDecodeBase64String(t *testing.T) {
	t.Parallel()
	t.Run("DecodeValidBase64String", func(t *testing.T) {
		t.Parallel()
		decoded, err := DecodeBase64String("U29tZSB2YWxpZCBzdHJpbmc=") // "Some valid string"
		require.NoError(t, err)
		assert.Equal(t, "Some valid string", decoded)
	})

	t.Run("DecodeInvalidBase64String", func(t *testing.T) {
		t.Parallel()
		decoded, err := DecodeBase64String("InvalidString!!!")
		require.NoError(t, err)
		assert.Equal(t, "InvalidString!!!", decoded) // Should return the original string
	})

	t.Run("DecodeEmptyString", func(t *testing.T) {
		t.Parallel()
		decoded, err := DecodeBase64String("")
		require.NoError(t, err)
		assert.Equal(t, "", decoded)
	})
}

func TestEncodeBase64String(t *testing.T) {
	t.Parallel()
	t.Run("EncodeString", func(t *testing.T) {
		t.Parallel()
		encoded := EncodeBase64String("Some valid string")
		assert.Equal(t, "U29tZSB2YWxpZCBzdHJpbmc=", encoded)
	})

	t.Run("EncodeEmptyString", func(t *testing.T) {
		t.Parallel()
		encoded := EncodeBase64String("")
		assert.Equal(t, "", encoded)
	})
}

func TestRoundTripBase64Encoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
	}{
		{"EncodeAndDecode", "Hello, World!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			encoded := EncodeBase64String(tt.input)
			decoded, err := DecodeBase64String(encoded)
			assert.NoError(t, err)
			assert.Equal(t, tt.input, decoded)
		})
	}
}

func TestDecodeBase64IfNeeded(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"AlreadyDecoded", "hello world", "hello world", false},
		{"Base64Encoded", "aGVsbG8gd29ybGQ=", "hello world", false},
		{"EmptyString", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := DecodeBase64IfNeeded(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEncodeValuePtr(t *testing.T) {
	t.Parallel()

	// Test with nil pointer
	result := EncodeValuePtr(nil)
	assert.Nil(t, result)

	// Test with valid string pointer
	input := "hello world"
	result = EncodeValuePtr(&input)
	assert.NotNil(t, result)
	assert.Equal(t, "aGVsbG8gd29ybGQ=", *result)

	// Test with already encoded string pointer
	encoded := "aGVsbG8gd29ybGQ="
	result = EncodeValuePtr(&encoded)
	assert.NotNil(t, result)
	assert.Equal(t, encoded, *result)
}

func TestDecodeStringMap(t *testing.T) {
	t.Parallel()

	// Test with nil map
	result, err := DecodeStringMap(nil, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test with empty map
	emptyMap := make(map[string]string)
	result, err = DecodeStringMap(&emptyMap, "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, *result)

	// Test with map containing encoded values
	encodedMap := map[string]string{
		"key1": "aGVsbG8=",   // "hello" encoded
		"key2": "plain text", // not encoded
		"key3": "d29ybGQ=",   // "world" encoded
	}
	result, err = DecodeStringMap(&encodedMap, "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "hello", (*result)["key1"])
	assert.Equal(t, "plain text", (*result)["key2"])
	assert.Equal(t, "world", (*result)["key3"])
}

func TestDecodeStringSlice(t *testing.T) {
	t.Parallel()

	// Test with nil slice
	result, err := DecodeStringSlice(nil, "test")
	assert.NoError(t, err)
	assert.Nil(t, result)

	// Test with empty slice
	emptySlice := []string{}
	result, err = DecodeStringSlice(&emptySlice, "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, *result)

	// Test with slice containing encoded values
	encodedSlice := []string{
		"aGVsbG8=",   // "hello" encoded
		"plain text", // not encoded
		"d29ybGQ=",   // "world" encoded
	}
	result, err = DecodeStringSlice(&encodedSlice, "test")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []string{"hello", "plain text", "world"}, *result)
}
