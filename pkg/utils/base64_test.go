package utils_test

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
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
		got := utils.IsBase64Encoded(tt.input)
		if got != tt.want {
			t.Errorf("IsBase64Encoded(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := "roundtrip value"
	encoded := utils.EncodeBase64String(original)
	if !utils.IsBase64Encoded(encoded) {
		t.Fatalf("encoded value expected to be base64 but IsBase64Encoded returned false: %s", encoded)
	}

	decoded, err := utils.DecodeBase64String(encoded)
	if err != nil {
		t.Fatalf("DecodeBase64String returned error: %v", err)
	}
	if decoded != original {
		t.Errorf("Decode after encode mismatch: got %s want %s", decoded, original)
	}
}

func TestDecodeBase64IfNeeded(t *testing.T) {
	encoded := utils.EncodeBase64String("plain text")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "needs decoding", input: encoded, want: "plain text"},
		{name: "no decoding", input: "already plain", want: "already plain"},
	}

	for _, tt := range tests {
		got, err := utils.DecodeBase64IfNeeded(tt.input)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.name, err)
		}
		if got != tt.want {
			t.Errorf("%s: got %s want %s", tt.name, got, tt.want)
		}
	}
}

func TestEncodeValue(t *testing.T) {
	encoded := utils.EncodeValue("plain text")
	encodedTwice := utils.EncodeValue(encoded)

	if !utils.IsBase64Encoded(encoded) {
		t.Fatalf("EncodeValue did not encode plain text")
	}
	if encoded != encodedTwice {
		t.Errorf("EncodeValue changed an already encoded string: first %s, second %s", encoded, encodedTwice)
	}
}

func TestEncodeValuePtr(t *testing.T) {
	if got := utils.EncodeValuePtr(nil); got != nil {
		t.Errorf("EncodeValuePtr(nil) = %v, want nil", got)
	}

	original := "plain"
	gotPtr := utils.EncodeValuePtr(&original)
	if gotPtr == nil {
		t.Fatalf("EncodeValuePtr returned nil for non-nil input pointer")
	}

	if !utils.IsBase64Encoded(*gotPtr) {
		t.Errorf("EncodeValuePtr did not encode the string, got %s", *gotPtr)
	}
	if original != "plain" {
		t.Errorf("EncodeValuePtr modified the original string variable: %s", original)
	}
}

func TestDecodeStringMapAndSlice(t *testing.T) {
	encoded := utils.EncodeValue("value")

	srcMap := map[string]string{"k": encoded}
	decodedMap, err := utils.DecodeStringMap(&srcMap, "field")
	if err != nil {
		t.Fatalf("DecodeStringMap error: %v", err)
	}
	expectedMap := map[string]string{"k": "value"}
	if !reflect.DeepEqual(*decodedMap, expectedMap) {
		t.Errorf("DecodeStringMap = %v, want %v", *decodedMap, expectedMap)
	}

	srcSlice := []string{encoded, "plain"}
	decodedSlice, err := utils.DecodeStringSlice(&srcSlice, "field")
	if err != nil {
		t.Fatalf("DecodeStringSlice error: %v", err)
	}
	expectedSlice := []string{"value", "plain"}
	if !reflect.DeepEqual(*decodedSlice, expectedSlice) {
		t.Errorf("DecodeStringSlice = %v, want %v", *decodedSlice, expectedSlice)
	}
}
