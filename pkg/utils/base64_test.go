package utils

import (
	"encoding/base64"
	"reflect"
	"testing"
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
