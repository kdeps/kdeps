package utils

import "testing"

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
