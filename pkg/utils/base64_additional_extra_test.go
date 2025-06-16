package utils

import "testing"

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
