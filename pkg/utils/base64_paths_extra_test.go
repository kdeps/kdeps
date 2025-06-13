package utils

import "testing"

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
