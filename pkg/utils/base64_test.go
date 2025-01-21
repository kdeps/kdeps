package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		assert.NoError(t, err)
		assert.Equal(t, "Some valid string", decoded)
	})

	t.Run("DecodeInvalidBase64String", func(t *testing.T) {
		t.Parallel()
		decoded, err := DecodeBase64String("InvalidString!!!")
		assert.NoError(t, err)
		assert.Equal(t, "InvalidString!!!", decoded) // Should return the original string
	})

	t.Run("DecodeEmptyString", func(t *testing.T) {
		t.Parallel()
		decoded, err := DecodeBase64String("")
		assert.NoError(t, err)
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
	t.Run("EncodeAndDecode", func(t *testing.T) {
		t.Parallel()
		original := "Some valid string"
		encoded := EncodeBase64String(original)
		decoded, err := DecodeBase64String(encoded)

		assert.NoError(t, err)
		assert.Equal(t, original, decoded)
	})
}
