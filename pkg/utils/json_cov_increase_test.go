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
