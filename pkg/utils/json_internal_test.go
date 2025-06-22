package utils

import (
	"testing"
)

func TestIsJSON_Internal(t *testing.T) {
	if !IsJSON("{}") {
		t.Error("IsJSON failed for valid JSON object")
	}
	if !IsJSON("[]") {
		t.Error("IsJSON failed for valid JSON array")
	}
	if IsJSON("not json") {
		t.Error("IsJSON returned true for invalid JSON")
	}
}

func TestFixJSON_Internal(t *testing.T) {
	// Already valid
	in := `{"foo": "bar"}`
	if out := FixJSON(in); out != in {
		t.Errorf("FixJSON changed valid JSON: %q", out)
	}
	// Quoted
	quoted := `"{\"foo\": 1}"`
	if out := FixJSON(quoted); out != "{\"foo\": 1}" {
		t.Errorf("FixJSON did not unquote: %q", out)
	}
}
