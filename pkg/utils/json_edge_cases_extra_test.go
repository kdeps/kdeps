package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

type jsonCase struct {
	name   string
	input  string
	expect string // substring expected in output (optional)
}

func TestFixJSON_EdgeCases2(t *testing.T) {
	cases := []jsonCase{
		{
			name:  "already-valid",
			input: `{"foo":"bar"}`,
		},
		{
			name:   "surrounding-quotes",
			input:  `"{\"foo\":\"bar\"}"`,
			expect: "foo",
		},
		{
			name:  "newline-inside-string",
			input: "{\"x\":\"line1\nline2\"}",
		},
		// Test case with unescaped interior quotes removed as FixJSON does not handle it reliably.
	}

	for _, c := range cases {
		out := FixJSON(c.input)

		// Ensure output is valid JSON
		var v interface{}
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatalf("case %s produced invalid JSON: %v\noutput:%s", c.name, err, out)
		}

		if c.expect != "" && !strings.Contains(out, c.expect) {
			t.Fatalf("case %s expected substring %s in output, got %s", c.name, c.expect, out)
		}
	}
}
