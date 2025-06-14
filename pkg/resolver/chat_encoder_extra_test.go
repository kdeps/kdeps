package resolver

import (
	"encoding/base64"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestEncodeJSONResponseKeys(t *testing.T) {
	keys := []string{"one", "two"}
	encoded := encodeJSONResponseKeys(&keys)
	if encoded == nil || len(*encoded) != 2 {
		t.Fatalf("expected 2 encoded keys")
	}
	for i, k := range keys {
		want := utils.EncodeValue(k)
		if (*encoded)[i] != want {
			t.Errorf("key %d mismatch: got %s want %s", i, (*encoded)[i], want)
		}
	}
}

func TestDecodeField_Base64(t *testing.T) {
	original := "hello world"
	b64 := base64.StdEncoding.EncodeToString([]byte(original))
	ptr := &b64
	if err := decodeField(&ptr, "testField", utils.SafeDerefString, ""); err != nil {
		t.Fatalf("decodeField returned error: %v", err)
	}
	if utils.SafeDerefString(ptr) != original {
		t.Errorf("decodeField did not decode correctly: got %s", utils.SafeDerefString(ptr))
	}
}

func TestDecodeField_NonBase64(t *testing.T) {
	val := "plain value"
	ptr := &val
	if err := decodeField(&ptr, "testField", utils.SafeDerefString, "default"); err != nil {
		t.Fatalf("decodeField returned error: %v", err)
	}
	if utils.SafeDerefString(ptr) != val {
		t.Errorf("expected field to remain unchanged, got %s", utils.SafeDerefString(ptr))
	}
}
