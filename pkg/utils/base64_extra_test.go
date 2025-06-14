package utils

import (
	"encoding/base64"
	"reflect"
	"testing"
)

// TestIsBase64EncodedInvalidUTF8 ensures the function returns false when the decoded bytes are not valid UTF-8.
func TestIsBase64EncodedInvalidUTF8(t *testing.T) {
	// 0xff 0xff is invalid UTF-8 sequence
	invalidBytes := []byte{0xff, 0xff}
	enc := base64.StdEncoding.EncodeToString(invalidBytes)

	if IsBase64Encoded(enc) {
		t.Errorf("expected invalid UTF8 payload to return false")
	}
}

// TestDecodeStringMapAndSliceNoDecodeNeeded covers error paths when underlying values are bad base64.
func TestDecodeStringMapAndSliceNoDecodeNeeded(t *testing.T) {
	bad := "!!!notbase64!!!"
	m := map[string]string{"bad": bad}
	decodedMap, err := DecodeStringMap(&m, "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedMap)["bad"] != bad {
		t.Errorf("value changed unexpectedly: %s", (*decodedMap)["bad"])
	}

	s := []string{bad}
	decodedSlice, err := DecodeStringSlice(&s, "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedSlice)[0] != bad {
		t.Errorf("slice value changed unexpectedly: %s", (*decodedSlice)[0])
	}
}

func TestIsBase64EncodedExtra(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello, world"))
	if !IsBase64Encoded(encoded) {
		t.Fatalf("expected %q to be detected as base64", encoded)
	}

	notEncoded := "definitely-not-base64!"
	if IsBase64Encoded(notEncoded) {
		t.Fatalf("expected %q to NOT be detected as base64", notEncoded)
	}

	// empty string should return false
	if IsBase64Encoded("") {
		t.Fatalf("expected empty string to NOT be detected as base64")
	}
}

func TestDecodeBase64StringExtra(t *testing.T) {
	original := "sample-text"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))

	// should decode correctly
	decoded, err := DecodeBase64String(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != original {
		t.Fatalf("expected %q, got %q", original, decoded)
	}

	// non-base64 strings should be returned unchanged
	unchanged, err := DecodeBase64String(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unchanged != original {
		t.Fatalf("expected %q to remain unchanged, got %q", original, unchanged)
	}
}

func TestEncodeAndDecodeHelpersExtra(t *testing.T) {
	raw := "plain-value"
	encoded := EncodeValue(raw)
	if encoded == raw {
		t.Fatalf("EncodeValue did not encode the value")
	}

	// re-encoding an already encoded value should return it unchanged
	encodedAgain := EncodeValue(encoded)
	if encodedAgain != encoded {
		t.Fatalf("EncodeValue re-encoded an already encoded value")
	}

	// pointer variant
	if ptr := EncodeValuePtr(nil); ptr != nil {
		t.Fatalf("EncodeValuePtr(nil) expected nil, got %v", *ptr)
	}
	ptrVal := raw
	encodedPtr := EncodeValuePtr(&ptrVal)
	if encodedPtr == nil || *encodedPtr == raw {
		t.Fatalf("EncodeValuePtr did not encode pointed value")
	}

	// decode helper should round-trip
	decoded, err := DecodeBase64IfNeeded(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != raw {
		t.Fatalf("expected decoded value %q, got %q", raw, decoded)
	}
}

func TestDecodeStringMapAndSliceExtra(t *testing.T) {
	originalMap := map[string]string{
		"a": "value-a",
		"b": EncodeBase64String("value-b"),
	}
	decodedMap, err := DecodeStringMap(&originalMap, "headers")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedMap := map[string]string{
		"a": "value-a",
		"b": "value-b",
	}
	if !reflect.DeepEqual(*decodedMap, expectedMap) {
		t.Fatalf("expected map %v, got %v", expectedMap, *decodedMap)
	}

	// nil map should return nil
	if m, err := DecodeStringMap(nil, "headers"); err != nil || m != nil {
		t.Fatalf("expected nil result for nil input map, got %v, err=%v", m, err)
	}

	originalSlice := []string{"value-a", EncodeBase64String("value-b")}
	decodedSlice, err := DecodeStringSlice(&originalSlice, "args")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedSlice := []string{"value-a", "value-b"}
	if !reflect.DeepEqual(*decodedSlice, expectedSlice) {
		t.Fatalf("expected slice %v, got %v", expectedSlice, *decodedSlice)
	}

	// nil slice should return nil
	if slc, err := DecodeStringSlice(nil, "args"); err != nil || slc != nil {
		t.Fatalf("expected nil result for nil input slice, got %v, err=%v", slc, err)
	}
}
