package utils

import "testing"

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
