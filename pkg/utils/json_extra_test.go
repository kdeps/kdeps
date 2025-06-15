package utils

import (
	"testing"
)

func TestFixJSON_EscapesAndWhitespace(t *testing.T) {
	// Contains newline inside quoted value and stray unescaped quote.
	input := "{\n  \"msg\": \"Hello\nWorld\",\n  \"quote\": \"She said \"Hi\"\"\n}"
	expected := "{\n\"msg\": \"Hello\\nWorld\",\n\"quote\": \"She said \\\"Hi\\\"\"\n}"

	if got := FixJSON(input); got != expected {
		t.Errorf("FixJSON mismatch\nwant: %s\n got: %s", expected, got)
	}
}

func TestFixJSONComprehensive(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		assert func(string) bool
	}{
		{
			name:   "AlreadyValid",
			input:  `{"a":1}`,
			assert: func(out string) bool { return out == `{"a":1}` },
		},
		{
			name:   "WrappedQuotes",
			input:  `"{\"b\":2}"`, // double-quoted JSON string (common from CLI)
			assert: func(out string) bool { return out == `{"b":2}` },
		},
	}

	for _, tc := range cases {
		out := FixJSON(tc.input)
		if !tc.assert(out) {
			t.Fatalf("%s: FixJSON output %q did not meet assertion", tc.name, out)
		}
	}
}
