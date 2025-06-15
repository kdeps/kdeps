package utils

import (
	"strings"
	"testing"
)

func TestFixJSON_UnquoteAndCleanup(t *testing.T) {
	input := "\"{\\\"msg\\\": \"Hello\\nWorld\"}\"" // wrapped & escaped
	fixed := FixJSON(input)

	// Should be unwrapped and contain escaped newline, not raw
	if !strings.Contains(fixed, "Hello\\nWorld") {
		t.Fatalf("FixJSON did not escape newline correctly: %s", fixed)
	}
}

func TestFixJSON_NoChangeNeeded(t *testing.T) {
	// Already well-formed JSON should come back unchanged
	input := "{\n\"foo\": \"bar\"\n}"
	if FixJSON(input) != input {
		t.Fatalf("FixJSON modified an already valid JSON string")
	}
}

func TestFixJSON_UnquoteErrorPath(t *testing.T) {
	// malformed quoted string that will fail strconv.Unquote
	input := "\"\\x\"" // contains invalid escape sequence
	out := FixJSON(input)
	// Function should return a non-empty string and not panic
	if out == "" {
		t.Fatalf("FixJSON returned empty string for malformed input")
	}
}
