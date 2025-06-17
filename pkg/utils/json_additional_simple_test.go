package utils

import "testing"

func TestFixJSONAdditional(t *testing.T) {
	// Already valid JSON should remain unchanged
	valid := `{"key":"value"}`
	if got := FixJSON(valid); got != valid {
		t.Fatalf("FixJSON altered valid JSON: %s", got)
	}

	// Quoted JSON string should be unquoted
	quoted := `"{\"a\":1}"`
	expectedUnquoted := `{"a":1}`
	if got := FixJSON(quoted); got != expectedUnquoted {
		t.Fatalf("FixJSON failed to unquote: %s", got)
	}

	// JSON with newline inside string literal should be escaped
	raw := "{\n  \"msg\": \"hello\nworld\"\n}"
	fixed := FixJSON(raw)
	if fixed == raw {
		t.Fatalf("FixJSON did not modify input with raw newline")
	}
	// Ensure result is valid JSON
	if !IsJSON(fixed) {
		t.Fatalf("FixJSON output is not valid JSON: %s", fixed)
	}
}
