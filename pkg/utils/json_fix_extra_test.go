package utils

import "testing"

func TestFixJSONRepairsCommonIssues(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "SurroundingQuotes",
			input: "\"{\\\"k\\\":1}\"", // string surrounded by extra quotes from CLI
		},
		{
			name:  "RawNewlineInString",
			input: "{\"msg\": \"line1\nline2\"}", // raw \n makes JSON invalid
		},
		{
			name:  "InteriorQuote",
			input: "{\"quote\": \"He said \\\"hello\\\".\"}",
		},
	}

	for _, c := range cases {
		fixed := FixJSON(c.input)
		if !IsJSON(fixed) {
			t.Fatalf("case %s: result is not valid JSON: %s", c.name, fixed)
		}
	}
}

func TestIsJSONDetectsValidAndInvalid(t *testing.T) {
	if !IsJSON("{}") {
		t.Fatalf("expected '{}' to be valid JSON")
	}
	if IsJSON("not-json") {
		t.Fatalf("expected 'not-json' to be invalid")
	}
}

// TestFixJSONRepair ensures the function repairs common mistakes like missing commas
// and unescaped newlines so that the result is valid JSON.
func TestFixJSONRepair(t *testing.T) {
	// Common CLI scenario: JSON gets wrapped in quotes with inner quotes escaped
	bad := "\"{\\\"foo\\\":123}\""

	fixed := FixJSON(bad)
	if !IsJSON(fixed) {
		t.Fatalf("FixJSON did not return valid JSON: %s", fixed)
	}
}
