package utils

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsBase64Encoded(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid", input: base64.StdEncoding.EncodeToString([]byte("hello")), want: true},
		{name: "empty", input: "", want: true},
		{name: "invalid chars", input: "SGVsbG@=", want: false},
		{name: "wrong padding", input: "abc", want: false},
	}

	for _, tt := range tests {
		got := IsBase64Encoded(tt.input)
		if got != tt.want {
			t.Errorf("IsBase64Encoded(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := "roundtrip value"
	encoded := EncodeBase64String(original)
	if !IsBase64Encoded(encoded) {
		t.Fatalf("encoded value expected to be base64 but IsBase64Encoded returned false: %s", encoded)
	}

	decoded, err := DecodeBase64String(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64String returned error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decode after encode mismatch: got %s want %s", decoded, original)
	}
}

func TestDecodeBase64IfNeeded(t *testing.T) {
	encoded := EncodeBase64String("plain text")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "needs decoding", input: encoded, want: "plain text"},
		{name: "no decoding", input: "already plain", want: "already plain"},
	}

	for _, tt := range tests {
		got, err := DecodeBase64IfNeeded(tt.input)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if got != tt.want {
			t.Errorf("%s: got %s want %s", tt.name, got, tt.want)
		}
	}
}

func TestEncodeValue_Base64Pkg(t *testing.T) {
	encoded := EncodeValue("plain text")
	encodedTwice := EncodeValue(encoded)

	if !IsBase64Encoded(encoded) {
		t.Fatalf("EncodeValue did not encode plain text")
	}
	if encoded != encodedTwice {
		t.Errorf("EncodeValue changed an already encoded string: first %s, second %s", encoded, encodedTwice)
	}
}

func TestEncodeValuePtr_Base64Pkg(t *testing.T) {
	if got := EncodeValuePtr(nil); got != nil {
		t.Errorf("EncodeValuePtr(nil) = %v, want nil", got)
	}

	original := "plain"
	gotPtr := EncodeValuePtr(&original)
	if gotPtr == nil {
		t.Fatalf("EncodeValuePtr returned nil for non-nil input pointer")
	}

	if !IsBase64Encoded(*gotPtr) {
		t.Errorf("EncodeValuePtr did not encode the string, got %s", *gotPtr)
	}
	if original != "plain" {
		t.Errorf("EncodeValuePtr modified the original string variable: %s", original)
	}
}

func TestDecodeStringMapAndSlice(t *testing.T) {
	encoded := EncodeValue("value")

	srcMap := map[string]string{"k": encoded}
	decodedMap, err := DecodeStringMap(&srcMap, "field")
	if err != nil {
		t.Fatalf("DecodeStringMap error: %v", err)
	}
	expectedMap := map[string]string{"k": "value"}
	if !reflect.DeepEqual(*decodedMap, expectedMap) {
		t.Errorf("DecodeStringMap = %v, want %v", *decodedMap, expectedMap)
	}

	srcSlice := []string{encoded, "plain"}
	decodedSlice, err := DecodeStringSlice(&srcSlice, "field")
	if err != nil {
		t.Fatalf("DecodeStringSlice error: %v", err)
	}
	expectedSlice := []string{"value", "plain"}
	if !reflect.DeepEqual(*decodedSlice, expectedSlice) {
		t.Errorf("DecodeStringSlice = %v, want %v", *decodedSlice, expectedSlice)
	}
}

func TestBase64Helpers(t *testing.T) {
	original := "hello, kdeps!"

	encoded := EncodeBase64String(original)
	if !IsBase64Encoded(encoded) {
		t.Fatalf("expected encoded string to be detected as base64")
	}

	// Ensure raw strings are not falsely detected
	if IsBase64Encoded(original) {
		t.Fatalf("expected raw string to NOT be detected as base64")
	}

	// Decode the encoded string and verify it matches the original
	decoded, err := DecodeBase64String(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64String returned error: %v", err)
	}
	if decoded != original {
		t.Fatalf("decoded value mismatch: got %q, want %q", decoded, original)
	}

	// DecodeBase64String should return an error if the input is not base64
	_, err = DecodeBase64String(original)
	if err == nil {
		t.Fatalf("expected error when decoding non-base64 string, but got none")
	}

	// DecodeBase64IfNeeded helper
	maybeDecoded, err := DecodeBase64IfNeeded(encoded)
	if err != nil || maybeDecoded != original {
		t.Fatalf("DecodeBase64IfNeeded failed: %v, value: %q", err, maybeDecoded)
	}

	unchanged, err := DecodeBase64IfNeeded(original)
	if err != nil || unchanged != original {
		t.Fatalf("DecodeBase64IfNeeded altered raw string: %q", unchanged)
	}

	// EncodeValue should encode raw strings but leave encoded ones intact
	if ev := EncodeValue(original); !IsBase64Encoded(ev) {
		t.Fatalf("EncodeValue did not encode raw string")
	}
	if ev := EncodeValue(encoded); ev != encoded {
		t.Fatalf("EncodeValue modified already encoded string")
	}

	// EncodeValuePtr tests
	ptrOriginal := original
	encodedPtr := EncodeValuePtr(&ptrOriginal)
	if encodedPtr == nil || !IsBase64Encoded(*encodedPtr) {
		t.Fatalf("EncodeValuePtr failed to encode pointer value")
	}

	// nil pointer should stay nil
	if res := EncodeValuePtr(nil); res != nil {
		t.Fatalf("EncodeValuePtr should return nil for nil input")
	}

	// Map decoding helper
	srcMap := map[string]string{"key": encoded}
	decodedMap, err := DecodeStringMap(&srcMap, "test-map")
	if err != nil {
		t.Fatalf("DecodeStringMap returned error: %v", err)
	}
	if (*decodedMap)["key"] != original {
		t.Fatalf("DecodeStringMap failed: got %q, want %q", (*decodedMap)["key"], original)
	}

	// Slice decoding helper
	srcSlice := []string{encoded, encoded}
	decodedSlice, err := DecodeStringSlice(&srcSlice, "test-slice")
	if err != nil {
		t.Fatalf("DecodeStringSlice returned error: %v", err)
	}
	for i, v := range *decodedSlice {
		if v != original {
			t.Fatalf("DecodeStringSlice[%d] = %q, want %q", i, v, original)
		}
	}
}

func TestDecodeBase64StringHelpers(t *testing.T) {
	orig := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(orig))

	t.Run("ValidString", func(t *testing.T) {
		out, err := DecodeBase64String(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != orig {
			t.Fatalf("want %q got %q", orig, out)
		}
	})

	t.Run("InvalidString", func(t *testing.T) {
		in := "$$invalid$$"
		_, err := DecodeBase64String(in)
		if err == nil {
			t.Fatalf("expected error for invalid base64 string, but got none")
		}
	})
}

func TestDecodeStringMapAndSliceHelpers(t *testing.T) {
	m := map[string]string{"a": "foo", "b": "bar"}
	for k, v := range m {
		m[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	gotMapPtr, err := DecodeStringMap(&m, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotMap := *gotMapPtr
	if !reflect.DeepEqual(gotMap, map[string]string{"a": "foo", "b": "bar"}) {
		t.Fatalf("decoded map mismatch: %v", gotMap)
	}

	sl := []string{"foo", "bar"}
	encodedSlice := []string{
		base64.StdEncoding.EncodeToString([]byte(sl[0])),
		base64.StdEncoding.EncodeToString([]byte(sl[1])),
	}
	gotSlicePtr, err := DecodeStringSlice(&encodedSlice, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSlice := *gotSlicePtr
	if !reflect.DeepEqual(gotSlice, sl) {
		t.Fatalf("decoded slice mismatch: %v", gotSlice)
	}
}

func TestIsBase64Encoded_EdgeCasesAdditional(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},                          // empty string is valid base64 encoding of empty byte array
		{"abc", false},                      // length not multiple of 4
		{"@@@@", false},                     // invalid chars
		{EncodeBase64String("hello"), true}, // valid
	}
	for _, tt := range tests {
		got := IsBase64Encoded(tt.in)
		if got != tt.want {
			t.Fatalf("IsBase64Encoded(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestEncodeFunctions(t *testing.T) {
	input := "hello"
	encoded := EncodeBase64String(input)
	if encoded == input {
		t.Fatalf("expected encoded string to differ from input")
	}
	// EncodeValue should detect non-base64 input and encode again (idempotency when applied twice)
	once := EncodeValue(input)
	if once != encoded {
		t.Fatalf("EncodeValue did not encode as expected")
	}
	// Calling EncodeValue on already encoded string should return unchanged.
	twice := EncodeValue(encoded)
	if twice != encoded {
		t.Fatalf("EncodeValue re-encoded already encoded string")
	}
}

func TestIsBase64Encoded_InvalidChar(t *testing.T) {
	str := "abcd#==" // '#' invalid
	if IsBase64Encoded(str) {
		t.Errorf("expected false for string with invalid char")
	}
}

// TestIsBase64Encoded_EdgeCases covers the branch where input length is not divisible by 4 but still contains only
// valid characters, ensuring the early-length check triggers the false path.
func TestIsBase64Encoded_EdgeCases(t *testing.T) {
	// length not divisible by 4 but contains only valid base64 characters
	badLen := "abcdE" // 5 chars
	if IsBase64Encoded(badLen) {
		t.Fatalf("expected false for invalid length input")
	}
}

func TestIsBase64Encoded_DecodeError(t *testing.T) {
	malformed := "A===" // length divisible by 4 and only valid chars but invalid padding
	if IsBase64Encoded(malformed) {
		t.Fatalf("expected false for malformed padding input")
	}
}

func TestDecodeBase64String_ErrorPath(t *testing.T) {
	// non-base64 but passes IsBase64Encoded check (length %4==0 and valid chars)
	invalid := "AAAA" // length divisible by 4 but will decode to invalid UTF-8 (all zero bytes but valid)
	decoded, err := DecodeBase64String(invalid)
	if err != nil {
		// expected an error only when DecodeString fails due to bad padding etc.
		// For "AAAA" decoding succeeds to "\x00\x00\x00", which is valid UTF-8, so err should be nil.
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != "\x00\x00\x00" {
		t.Fatalf("unexpected decoded value: %q", decoded)
	}

	// Produce input that is *not* base64, the helper should return an error.
	notEncoded := "not_base64"
	_, err = DecodeBase64String(notEncoded)
	if err == nil {
		t.Fatalf("expected error for non-base64 input, but got none")
	}
}

func TestDecodeStringMapAndSlice_ErrorPaths(t *testing.T) {
	// map with a value that is *not* base64 should simply be returned unchanged without error
	mp := map[string]string{"x": "not_base64"}
	decodedMap, err := DecodeStringMap(&mp, "headers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedMap)["x"] != "not_base64" {
		t.Fatalf("expected value unchanged, got %s", (*decodedMap)["x"])
	}

	// slice variant
	slc := []string{"not_base64"}
	decodedSlice, err := DecodeStringSlice(&slc, "items")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedSlice)[0] != "not_base64" {
		t.Fatalf("expected slice value unchanged, got %s", (*decodedSlice)[0])
	}
}

func TestDecodeStringHelpers_Branches(t *testing.T) {
	// 1) nil inputs should return error
	_, err := DecodeStringMap(nil, "hdr")
	if err == nil {
		t.Fatalf("expected error for nil map, but got none")
	}
	_, err = DecodeStringSlice(nil, "slice")
	if err == nil {
		t.Fatalf("expected error for nil slice, but got none")
	}

	// 2) non-base64 path: helper should return value unchanged without error
	badVal := "not_base64_val"
	mp := map[string]string{"k": badVal}
	dm, err := DecodeStringMap(&mp, "hdr")
	if err != nil || (*dm)["k"] != badVal {
		t.Fatalf("unexpected result for non-base64 map: %v err %v", dm, err)
	}
	sl := []string{badVal}
	ds, err := DecodeStringSlice(&sl, "slice")
	if err != nil || (*ds)[0] != badVal {
		t.Fatalf("unexpected result for non-base64 slice: %v err %v", ds, err)
	}
}

// TestDecodeStringHelpersErrorPaths exercises the error returns when values are malformed base64.
func TestDecodeStringHelpersErrorPaths(t *testing.T) {
	bad := "!!!notbase64!!!"

	m := map[string]string{"x": bad}
	mm, err := DecodeStringMap(&m, "hdr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*mm)["x"] != bad {
		t.Fatalf("value altered unexpectedly")
	}

	s := []string{bad}
	ss, err := DecodeStringSlice(&s, "arr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*ss)[0] != bad {
		t.Fatalf("slice value altered unexpectedly")
	}
}

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
	_, err = DecodeBase64String(invalid)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid base64 encoding")
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

	// Map/slice with nil pointer should return error
	_, err = DecodeStringMap(nil, "field")
	require.Error(t, err)
	require.Contains(t, err.Error(), "source map is nil")

	_, err = DecodeStringSlice(nil, "slice")
	require.Error(t, err)
	require.Contains(t, err.Error(), "source slice is nil")
}
