package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestIsBase64Encoded(t *testing.T) {
	// Test valid base64 strings
	assert.True(t, utils.IsBase64Encoded("SGVsbG8gV29ybGQ="))
	assert.True(t, utils.IsBase64Encoded("YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo="))
	assert.True(t, utils.IsBase64Encoded("MTIzNDU2Nzg5MA=="))

	// Test invalid base64 strings
	assert.False(t, utils.IsBase64Encoded("Hello World"))
	assert.False(t, utils.IsBase64Encoded("abc123!@#"))
	assert.False(t, utils.IsBase64Encoded(""))
}

func TestEncodeBase64String(t *testing.T) {
	// Test encoding
	encoded := utils.EncodeBase64String("Hello World")
	assert.True(t, utils.IsBase64Encoded(encoded))

	// Test decoding
	decoded, err := utils.DecodeBase64String(encoded)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", decoded)
}

func TestDecodeBase64String(t *testing.T) {
	// Test valid base64 decoding
	decoded, err := utils.DecodeBase64String("SGVsbG8gV29ybGQ=")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", decoded)

	// Test invalid base64 decoding
	_, err = utils.DecodeBase64String("invalid-base64")
	assert.Error(t, err)
}

func TestDecodeBase64IfNeeded(t *testing.T) {
	// Test base64 encoded string
	decoded, err := utils.DecodeBase64IfNeeded("SGVsbG8gV29ybGQ=")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", decoded)

	// Test non-base64 string (should return as-is)
	decoded, err = utils.DecodeBase64IfNeeded("Hello World")
	require.NoError(t, err)
	assert.Equal(t, "Hello World", decoded)
}

func TestEncodeValueBase64(t *testing.T) {
	// Test encoding string value
	encoded := utils.EncodeValue("Hello World")
	assert.True(t, utils.IsBase64Encoded(encoded))

	// Test encoding empty string
	encoded = utils.EncodeValue("")
	assert.True(t, utils.IsBase64Encoded(encoded))
}

func TestEncodeValuePtr(t *testing.T) {
	// Test encoding string pointer
	str := "Hello World"
	encoded := utils.EncodeValuePtr(&str)
	assert.True(t, utils.IsBase64Encoded(*encoded))

	// Test encoding nil pointer
	encoded = utils.EncodeValuePtr(nil)
	assert.Nil(t, encoded)
}
