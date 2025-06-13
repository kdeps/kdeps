package utils_test

import (
	"encoding/base64"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
)

// TestIsBase64EncodedInvalidUTF8 ensures the function returns false when the decoded bytes are not valid UTF-8.
func TestIsBase64EncodedInvalidUTF8(t *testing.T) {
	// 0xff 0xff is invalid UTF-8 sequence
	invalidBytes := []byte{0xff, 0xff}
	enc := base64.StdEncoding.EncodeToString(invalidBytes)

	if utils.IsBase64Encoded(enc) {
		t.Errorf("expected invalid UTF8 payload to return false")
	}
}

// TestDecodeStringMapAndSliceNoDecodeNeeded covers error paths when underlying values are bad base64.
func TestDecodeStringMapAndSliceNoDecodeNeeded(t *testing.T) {
	bad := "!!!notbase64!!!"
	m := map[string]string{"bad": bad}
	decodedMap, err := utils.DecodeStringMap(&m, "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedMap)["bad"] != bad {
		t.Errorf("value changed unexpectedly: %s", (*decodedMap)["bad"])
	}

	s := []string{bad}
	decodedSlice, err := utils.DecodeStringSlice(&s, "field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if (*decodedSlice)[0] != bad {
		t.Errorf("slice value changed unexpectedly: %s", (*decodedSlice)[0])
	}
}
