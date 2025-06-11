package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateString_EdgeCases(t *testing.T) {
	require.Equal(t, "short", TruncateString("short", 10))
	require.Equal(t, "...", TruncateString("longstring", 2))
	require.Equal(t, "longer", TruncateString("longer", 6))
}

func TestAllConditionsMet_Various(t *testing.T) {
	t.Run("AllTrueBool", func(t *testing.T) {
		conds := []interface{}{true, true}
		require.True(t, AllConditionsMet(&conds))
	})

	t.Run("AllTrueString", func(t *testing.T) {
		conds := []interface{}{"true", "TRUE"}
		require.True(t, AllConditionsMet(&conds))
	})

	t.Run("MixedFalse", func(t *testing.T) {
		conds := []interface{}{true, "false"}
		require.False(t, AllConditionsMet(&conds))
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		conds := []interface{}{errors.New("oops")}
		require.False(t, AllConditionsMet(&conds))
	})
}

func TestIsBase64Encoded_DecodeFunctions(t *testing.T) {
	original := "hello world"
	encoded := EncodeBase64String(original)

	// Positive path
	require.True(t, IsBase64Encoded(encoded))
	decoded, err := DecodeBase64String(encoded)
	require.NoError(t, err)
	require.Equal(t, original, decoded)

	// Negative path: not base64
	invalid := "not@@base64!"
	require.False(t, IsBase64Encoded(invalid))
	same, err := DecodeBase64String(invalid)
	require.NoError(t, err)
	require.Equal(t, invalid, same)
}

func TestDecodeStringHelpers_ErrorPaths(t *testing.T) {
	// Map with one bad base64 value
	badVal := "###" // definitely invalid
	m := map[string]string{"good": EncodeBase64String("ok"), "bad": badVal}
	decodedMap, err := DecodeStringMap(&m, "field")
	require.NoError(t, err)
	require.NotNil(t, decodedMap)

	// Slice with bad value
	s := []string{EncodeBase64String("x"), badVal}
	decodedSlice, err := DecodeStringSlice(&s, "slice")
	require.NoError(t, err)
	require.NotNil(t, decodedSlice)

	// Map/slice with nil pointer should return nil, no error
	mh, err := DecodeStringMap(nil, "field")
	require.NoError(t, err)
	require.Nil(t, mh)

	sh, err := DecodeStringSlice(nil, "slice")
	require.NoError(t, err)
	require.Nil(t, sh)
}
