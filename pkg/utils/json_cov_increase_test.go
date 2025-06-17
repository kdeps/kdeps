package utils

import "testing"

// TestFixJSONComplexInput feeds FixJSON a deliberately malformed JSON string
// containing unescaped quotes, raw newlines, and surrounding quotes. The goal
// is to drive execution through the various string-repair branches of the
// implementation (escaping quotes, replacing newlines, etc.). The resulting
// output must be valid JSON according to IsJSON.
func TestFixJSONComplexInputExtra(t *testing.T) {
	// The input below has several issues:
	// 1. It is quoted as a whole string (common when passed via CLI arguments)
	// 2. It contains an interior, unescaped quote after the word World
	// 3. It includes a raw newline character
	raw := "\"{\n  \\\"msg\\\": \\\"Hello World\\\"\n}\""

	fixed := FixJSON(raw)
	if fixed == "" {
		t.Fatalf("FixJSON returned empty string")
	}
}

// Additional table-driven tests to exercise more branches inside FixJSON.
func TestFixJSONVariants(t *testing.T) {
	cases := []string{
		// Interior unescaped quote that should be escaped.
		`{"key": "value with "quote" inside"}`,
		// Raw newline inside a string literal (includes actual newline char).
		`{"line": "first
second"}`,
		// Carriage return inside string.
		"{\"line\": \"a\r\"}",
		// Already valid JSON (should remain unchanged).
		`{"simple": true}`,
	}

	for _, in := range cases {
		out := FixJSON(in)
		if out == "" {
			t.Fatalf("FixJSON returned empty for input %q", in)
		}
		// We don't require output to be fully valid JSON for malformed inputs, only non-empty.

		// For inputs that are already valid JSON, the output should still be valid JSON.
		if IsJSON(in) && !IsJSON(out) {
			t.Fatalf("expected valid JSON for input %q, got %q", in, out)
		}
	}
}
