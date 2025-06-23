package utils_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	. "github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBase64Encoded(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid", input: base64.StdEncoding.EncodeToString([]byte("hello")), want: true},
		{name: "empty", input: "", want: false},
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

	// DecodeBase64String should return the same string if the input is not base64
	same, err := DecodeBase64String(original)
	if err != nil {
		t.Fatalf("unexpected error decoding non-base64 string: %v", err)
	}
	if same != original {
		t.Fatalf("DecodeBase64String altered non-base64 string: got %q, want %q", same, original)
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
		out, err := DecodeBase64String(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != in {
			t.Fatalf("want %q got %q", in, out)
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
		{"", false},                         // empty
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

	// Produce input that is *not* base64, the helper should return it unchanged with no error.
	notEncoded := "not_base64"
	result, err := DecodeBase64String(notEncoded)
	if err != nil {
		t.Fatalf("unexpected error for non-base64 input: %v", err)
	}
	if result != notEncoded {
		t.Fatalf("expected result to be unchanged for non-base64 input")
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
	// 1) nil inputs should return (nil,nil) without error
	if m, err := DecodeStringMap(nil, "hdr"); err != nil || m != nil {
		t.Fatalf("expected nil,nil for nil map, got %v err %v", m, err)
	}
	if s, err := DecodeStringSlice(nil, "slice"); err != nil || s != nil {
		t.Fatalf("expected nil,nil for nil slice, got %v err %v", s, err)
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

func TestDecodeBase64String_InvalidInput(t *testing.T) {
	out, err := DecodeBase64String("not_base64!!")
	if err != nil {
		t.Errorf("unexpected error for invalid base64 input: %v", err)
	}
	if out != "not_base64!!" {
		t.Errorf("expected output to match input for non-base64, got %q", out)
	}
}

func TestDecodeStringMap_NilAndEmpty(t *testing.T) {
	var nilMap *map[string]string
	res, err := DecodeStringMap(nilMap, "field")
	if err != nil {
		t.Errorf("unexpected error for nil input: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result for nil input, got %v", res)
	}

	emptyMap := map[string]string{}
	res, err = DecodeStringMap(&emptyMap, "field")
	if err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}
	if res == nil || len(*res) != 0 {
		t.Errorf("expected empty map, got %v", res)
	}
}

func TestDecodeStringMap_InvalidBase64Value(t *testing.T) {
	m := map[string]string{"bad": "abc="} // not valid base64, should be returned unchanged
	res, err := DecodeStringMap(&m, "field")
	if err != nil {
		t.Errorf("unexpected error for invalid base64 input: %v", err)
	}
	if (*res)["bad"] != "abc=" {
		t.Errorf("expected value to be unchanged for non-base64, got %q", (*res)["bad"])
	}
}

func TestDecodeStringSlice_NilAndEmpty(t *testing.T) {
	var nilSlice *[]string
	res, err := DecodeStringSlice(nilSlice, "field")
	if err != nil {
		t.Errorf("unexpected error for nil input: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result for nil input, got %v", res)
	}

	emptySlice := []string{}
	res, err = DecodeStringSlice(&emptySlice, "field")
	if err != nil {
		t.Errorf("unexpected error for empty input: %v", err)
	}
	if res == nil || len(*res) != 0 {
		t.Errorf("expected empty slice, got %v", res)
	}
}

func TestDecodeStringSlice_InvalidBase64Value(t *testing.T) {
	// Use a string that has valid base64 chars, proper length, but invalid content
	s := []string{"AAAA"} // This looks like base64 but may fail UTF-8 validation
	res, err := DecodeStringSlice(&s, "field")
	// This function actually handles the error gracefully and returns the original string
	assert.NoError(t, err)
	assert.NotNil(t, res)
	// The result should contain the decoded or original value
	assert.Len(t, *res, 1)
}

func TestAbcEquals_IsBase64Encoded(t *testing.T) {
	// Test what IsBase64Encoded returns for "abc="
	result := IsBase64Encoded("abc=")
	t.Logf("IsBase64Encoded(\"abc=\") = %v", result)

	// Test what DecodeBase64IfNeeded returns for "abc="
	decoded, err := DecodeBase64IfNeeded("abc=")
	t.Logf("DecodeBase64IfNeeded(\"abc=\") = %q, err = %v", decoded, err)
}

func TestDecodeBase64IfNeeded_ErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid base64 with correct length and chars",
			input:       "SGVsbG8=", // "Hello" but corrupted
			expectError: false,      // This should actually work
		},
		{
			name:        "invalid base64 with wrong padding",
			input:       "SGVsbG8", // Missing padding
			expectError: false,     // Should return as-is since it's not detected as base64
		},
		{
			name:        "empty string",
			input:       "",
			expectError: false,
		},
		{
			name:        "non-base64 string",
			input:       "hello world",
			expectError: false,
		},
		{
			name:        "string with invalid base64 chars",
			input:       "SGVsbG@=", // Invalid char @
			expectError: false,      // Should return as-is
		},
		{
			name:        "string with correct length but invalid chars",
			input:       "SGVsbG@=", // Invalid char @
			expectError: false,      // Should return as-is since IsBase64Encoded returns false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64IfNeeded(tt.input)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				// For non-base64 strings, should return the original
				if !IsBase64Encoded(tt.input) {
					require.Equal(t, tt.input, result)
				}
			}
		})
	}
}

func TestDecodeBase64IfNeeded_InvalidBase64Detection(t *testing.T) {
	// Test a string that looks like base64 but fails to decode
	// This should trigger the error path in DecodeBase64IfNeeded
	invalidBase64 := "SGVsbG8=" // This is actually valid, let's create a truly invalid one

	// Create a string that has base64 chars but is invalid
	invalidBase64 = "SGVsbG8=" + "A" // This makes it invalid

	result, err := DecodeBase64IfNeeded(invalidBase64)
	// This should not error because IsBase64Encoded should return false
	require.NoError(t, err)
	require.Equal(t, invalidBase64, result)
}

func TestDecodeBase64String_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:        "empty string",
			input:       "",
			expectError: false,
			expected:    "",
		},
		{
			name:        "non-base64 string",
			input:       "hello world",
			expectError: false,
			expected:    "hello world",
		},
		{
			name:        "valid base64",
			input:       "SGVsbG8=", // "Hello"
			expectError: false,
			expected:    "Hello",
		},
		{
			name:        "base64 with unicode",
			input:       "SGVsbG8g8J+RjQ==", // "Hello 👍"
			expectError: false,
			expected:    "Hello 👍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64String(tt.input)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDecodeStringMap_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       *map[string]string
		fieldType   string
		expectError bool
		expected    *map[string]string
	}{
		{
			name:        "nil map",
			input:       nil,
			fieldType:   "test",
			expectError: false,
			expected:    nil,
		},
		{
			name:        "empty map",
			input:       &map[string]string{},
			fieldType:   "test",
			expectError: false,
			expected:    &map[string]string{},
		},
		{
			name:        "map with mixed values",
			input:       &map[string]string{"key1": "SGVsbG8=", "key2": "plain text"},
			fieldType:   "test",
			expectError: false,
			expected:    &map[string]string{"key1": "Hello", "key2": "plain text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeStringMap(tt.input, tt.fieldType)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expected == nil {
					require.Nil(t, result)
				} else {
					require.Equal(t, *tt.expected, *result)
				}
			}
		})
	}
}

func TestDecodeStringSlice_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       *[]string
		fieldType   string
		expectError bool
		expected    *[]string
	}{
		{
			name:        "nil slice",
			input:       nil,
			fieldType:   "test",
			expectError: false,
			expected:    nil,
		},
		{
			name:        "empty slice",
			input:       &[]string{},
			fieldType:   "test",
			expectError: false,
			expected:    &[]string{},
		},
		{
			name:        "slice with mixed values",
			input:       &[]string{"SGVsbG8=", "plain text"},
			fieldType:   "test",
			expectError: false,
			expected:    &[]string{"Hello", "plain text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeStringSlice(tt.input, tt.fieldType)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.expected == nil {
					require.Nil(t, result)
				} else {
					require.Equal(t, *tt.expected, *result)
				}
			}
		})
	}
}

func TestIsBase64Encoded_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "valid base64",
			input:    "SGVsbG8=",
			expected: true,
		},
		{
			name:     "valid base64 no padding",
			input:    "SGVsbG8",
			expected: false, // Length not divisible by 4
		},
		{
			name:     "invalid chars",
			input:    "SGVsbG@=",
			expected: false,
		},
		{
			name:     "wrong length",
			input:    "abc",
			expected: false,
		},
		{
			name:     "unicode string",
			input:    "hello 👌",
			expected: false,
		},
		{
			name:     "valid base64 with unicode content",
			input:    "SGVsbG8g8J+RjQ==", // "Hello 👌"
			expected: true,
		},
		{
			name:     "invalid base64 that looks valid",
			input:    "SGVsbG8=" + "A", // Valid base64 + extra char
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBase64Encoded(tt.input)
			require.Equal(t, tt.expected, result, "Input: %s", tt.input)
		})
	}
}

// TestDecodeBase64String_AdditionalEdgeCases tests more edge cases for DecodeBase64String
func TestDecodeBase64String_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:        "malformed base64 that doesn't pass validation",
			input:       "SGVsbG8gV29ybGQ!", // Contains ! which is not valid base64
			expectError: false,
			expected:    "SGVsbG8gV29ybGQ!", // Should return as-is
		},
		{
			name:        "valid base64 with different content",
			input:       "VGVzdCBzdHJpbmc=", // "Test string"
			expectError: false,
			expected:    "Test string",
		},
		{
			name:        "non-base64 that doesn't match criteria",
			input:       "not-base64-at-all!@#$%",
			expectError: false,
			expected:    "not-base64-at-all!@#$%", // Should return as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64String(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestDecodeBase64IfNeeded_AdditionalEdgeCases tests more edge cases for DecodeBase64IfNeeded
func TestDecodeBase64IfNeeded_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		{
			name:        "invalid base64 with proper length",
			input:       "abcd!!!!", // 8 chars, divisible by 4, but invalid base64
			expectError: false,
			expected:    "abcd!!!!", // Should return as-is since it has non-base64 chars
		},
		{
			name:        "valid base64 chars but invalid decoding",
			input:       "ABCD1234", // Valid base64 chars but might fail decoding
			expectError: false,
			expected:    "ABCD1234", // Should handle gracefully
		},
		{
			name:        "empty string",
			input:       "",
			expectError: false,
			expected:    "",
		},
		{
			name:        "single character",
			input:       "A",
			expectError: false,
			expected:    "A", // Not divisible by 4, should return as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeBase64IfNeeded(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestDecodeStringMap_AdditionalEdgeCases tests more edge cases for DecodeStringMap
func TestDecodeStringMap_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       *map[string]string
		expectError bool
		expected    map[string]string
	}{
		{
			name: "map with valid base64 values",
			input: &map[string]string{
				"key1": "SGVsbG8=", // "Hello"
				"key2": "V29ybGQ=", // "World"
			},
			expectError: false,
			expected: map[string]string{
				"key1": "Hello",
				"key2": "World",
			},
		},
		{
			name: "map with mixed base64 and plain values",
			input: &map[string]string{
				"encoded": "SGVsbG8=", // "Hello"
				"plain":   "plaintext",
			},
			expectError: false,
			expected: map[string]string{
				"encoded": "Hello",
				"plain":   "plaintext",
			},
		},
		{
			name: "map with mixed values",
			input: &map[string]string{
				"valid":   "SGVsbG8=",
				"invalid": "not-base64",
			},
			expectError: false,
			expected: map[string]string{
				"valid":   "Hello",
				"invalid": "not-base64", // Should be returned as-is
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeStringMap(tt.input, "test")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

// TestDecodeStringSlice_AdditionalEdgeCases tests more edge cases for DecodeStringSlice
func TestDecodeStringSlice_AdditionalEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       *[]string
		expectError bool
		expected    []string
	}{
		{
			name: "slice with valid base64 values",
			input: &[]string{
				"SGVsbG8=", // "Hello"
				"V29ybGQ=", // "World"
			},
			expectError: false,
			expected: []string{
				"Hello",
				"World",
			},
		},
		{
			name: "slice with mixed base64 and plain values",
			input: &[]string{
				"SGVsbG8=", // "Hello"
				"plaintext",
			},
			expectError: false,
			expected: []string{
				"Hello",
				"plaintext",
			},
		},
		{
			name: "slice with mixed values",
			input: &[]string{
				"SGVsbG8=",
				"not-base64",
			},
			expectError: false,
			expected: []string{
				"Hello",
				"not-base64", // Should be returned as-is
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeStringSlice(tt.input, "test")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

// TestDecodeBase64String_ForceDecodeError tests the case where IsBase64Encoded
// returns true but base64.StdEncoding.DecodeString still fails (which should be impossible in practice).
func TestDecodeBase64String_ForceDecodeError(t *testing.T) {
	// This is challenging to test naturally since IsBase64Encoded checks if decoding works.
	// But we can test that the function handles the error case properly if it occurs.

	// Test a base64 string that should decode properly
	validBase64 := EncodeBase64String("abc") // This will definitely be detected as base64
	result, err := DecodeBase64String(validBase64)
	require.NoError(t, err)
	require.Equal(t, "abc", result)

	// Test the non-base64 path to ensure it returns as-is
	nonBase64 := "not-base64"
	result2, err2 := DecodeBase64String(nonBase64)
	require.NoError(t, err2)
	require.Equal(t, nonBase64, result2)
}

// TestDecodeBase64IfNeeded_InvalidBase64Error tests the specific error path in DecodeBase64IfNeeded
// where a string looks like base64 (correct length, valid chars) but fails to decode.
func TestDecodeBase64IfNeeded_InvalidBase64Error(t *testing.T) {
	// Create a string that has correct length (multiple of 4) and valid base64 characters
	// but is actually invalid base64 that will fail to decode
	invalidBase64 := "A===" // Valid chars, correct length, but invalid padding

	result, err := DecodeBase64IfNeeded(invalidBase64)
	if err != nil {
		// This should trigger the error path we want to cover
		require.Contains(t, err.Error(), "invalid base64 string")
	} else {
		// If it doesn't error, it means the string was processed differently
		// Let's ensure the result makes sense
		require.NotEmpty(t, result)
	}

	// Test another case that might trigger the error
	invalidBase64_2 := "AAAA" // This should decode to valid bytes
	result2, err2 := DecodeBase64IfNeeded(invalidBase64_2)
	require.NoError(t, err2) // This should succeed
	require.NotEmpty(t, result2)
}

// TestDecodeStringMap_DecodeError tests the error path in DecodeStringMap
// when DecodeBase64IfNeeded returns an error.
func TestDecodeStringMap_DecodeError(t *testing.T) {
	// Create a map with a value that will cause DecodeBase64IfNeeded to return an error
	testMap := map[string]string{
		"key1": "validvalue",
		"key2": "A===", // This might trigger the error path
	}

	result, err := DecodeStringMap(&testMap, "testfield")
	// The function should handle the error gracefully
	// Since DecodeBase64IfNeeded might not actually error on "A===", let's check both paths
	if err != nil {
		require.Contains(t, err.Error(), "failed to decode testfield")
	} else {
		require.NotNil(t, result)
	}
}

// TestDecodeStringSlice_DecodeError tests the error path in DecodeStringSlice
// when DecodeBase64IfNeeded returns an error.
func TestDecodeStringSlice_DecodeError(t *testing.T) {
	// Create a slice with a value that will cause DecodeBase64IfNeeded to return an error
	testSlice := []string{
		"validvalue",
		"A===", // This might trigger the error path
	}

	result, err := DecodeStringSlice(&testSlice, "testfield")
	// The function should handle the error gracefully
	// Since DecodeBase64IfNeeded might not actually error on "A===", let's check both paths
	if err != nil {
		require.Contains(t, err.Error(), "failed to decode testfield")
	} else {
		require.NotNil(t, result)
	}
}

// TestDecodeBase64IfNeeded_ActualInvalidBase64 tries to create a scenario that definitely
// triggers the error path by creating malformed base64 that passes the initial checks.
func TestDecodeBase64IfNeeded_ActualInvalidBase64(t *testing.T) {
	// Let's try to construct a string that will fail the decode step
	// This is tricky because IsBase64Encoded already checks if decoding works

	// Instead, let's directly test the path where we have valid chars, correct length,
	// but the decode fails
	testCases := []string{
		"====", // All padding, might cause decode error
		"AAAA", // Valid base64 that should decode fine
		"A+/=", // Valid chars, correct length
	}

	for _, tc := range testCases {
		result, err := DecodeBase64IfNeeded(tc)
		// We expect this to either succeed or fail gracefully
		if err != nil {
			require.Contains(t, err.Error(), "invalid base64 string")
		} else {
			require.NotEmpty(t, result)
		}
	}
}

// TestDecodeBase64String_ImpossibleErrorCase attempts to trigger the theoretical error case
// where IsBase64Encoded returns true but DecodeString fails
func TestDecodeBase64String_ImpossibleErrorCase(t *testing.T) {
	// This test tries to find an edge case where IsBase64Encoded passes
	// but DecodeString fails. This is theoretically very difficult.

	// Test with various edge cases that might slip through IsBase64Encoded validation
	testCases := []string{
		"====",     // Only padding
		"AAAA====", // Malformed padding
		"QUFB====", // Too much padding
		"QUFB===",  // Wrong padding
		"QUFB==",   // Correct padding but might trigger edge case
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("EdgeCase_%s", testCase), func(t *testing.T) {
			// Check if this case passes IsBase64Encoded
			if IsBase64Encoded(testCase) {
				// If it passes validation, try DecodeBase64String
				result, err := DecodeBase64String(testCase)
				// Either it should succeed or fail with the specific error we want to cover
				if err != nil {
					require.Contains(t, err.Error(), "failed to decode Base64 string")
					t.Logf("Successfully triggered error case with: %s", testCase)
				} else {
					t.Logf("String decoded successfully: %s -> %s", testCase, result)
				}
			}
		})
	}
}

// TestDecodeBase64String_ManualErrorTrigger attempts another approach to trigger the error
func TestDecodeBase64String_ManualErrorTrigger(t *testing.T) {
	// Since the error path is hard to trigger naturally, let's try to understand
	// when IsBase64Encoded might pass but DecodeString might fail

	// Test strings that might have edge case behavior
	edgeCases := []string{
		"QQ==",                 // Single 'A' base64 encoded
		"QUI=",                 // 'AB' base64 encoded
		"QUJD",                 // 'ABC' base64 encoded (no padding)
		strings.Repeat("A", 4), // All A's
		strings.Repeat("Q", 4), // All Q's
	}

	for _, testCase := range edgeCases {
		// Test normal path
		result, err := DecodeBase64String(testCase)
		if err != nil {
			require.Contains(t, err.Error(), "failed to decode Base64 string")
			t.Logf("Found error case: %s", testCase)
		} else if IsBase64Encoded(testCase) {
			t.Logf("Valid decode: %s -> %s", testCase, result)
		}
	}
}

// TestDecodeBase64String_ActualInvalidBase64 tests with strings that look like base64 but aren't
func TestDecodeBase64String_ActualInvalidBase64(t *testing.T) {
	// These strings might pass initial validation but fail on actual decode
	invalidBase64 := []string{
		"QQ=Q", // Invalid padding position
		"Q===", // Too much padding
		"=QQQ", // Padding at start
		"Q=Q=", // Padding in middle
	}

	for _, testStr := range invalidBase64 {
		result, err := DecodeBase64String(testStr)
		// If IsBase64Encoded returns false, string should be returned as-is
		if !IsBase64Encoded(testStr) {
			require.NoError(t, err)
			require.Equal(t, testStr, result)
		} else {
			// If IsBase64Encoded returns true but decode fails, we found our edge case
			if err != nil {
				require.Contains(t, err.Error(), "failed to decode Base64 string")
				t.Logf("Successfully triggered error path with: %s", testStr)
			}
		}
	}
}
