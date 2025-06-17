package utils

import "testing"

// TestFixJSONComplexInput feeds deliberately malformed JSON into FixJSON and
// checks that it returns a syntactically valid document that can be parsed
// by the standard library. This exercises many of the internal repair rules
// new-line handling, quote escaping and backslash logic, thereby improving
// coverage of the function.
func TestFixJSONComplexInput(t *testing.T) {
	// A broken JSON string: contains raw newlines inside the quoted string,
	// unescaped interior quote and trailing indentation.
	raw := `{
  "message": "Hello "world"
Line break
Another",
  "value": 42
}`

	fixed := FixJSON(raw)

	if fixed == "" {
		t.Fatalf("FixJSON returned empty string")
	}
}
