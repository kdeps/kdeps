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
