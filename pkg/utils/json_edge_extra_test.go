package utils

import (
	"strings"
	"testing"
)

// TestFixJSON_EdgeCases exercises branches related to surrounding quotes and newline escaping.
func TestFixJSON_EdgeCases(t *testing.T) {
	checks := []string{
		`"msg"`,
	}

	in := `"{\n\"msg\":\"hi\"\n}"`
	out := FixJSON(in)
	for _, sub := range checks {
		if !strings.Contains(out, sub) {
			t.Fatalf("FixJSON output missing %s: %s", sub, out)
		}
	}

	// newline escaping case
	newlineIn := "\"line1\nline2\""
	newlineOut := FixJSON(newlineIn)
	if !strings.Contains(newlineOut, "\\n") && !strings.Contains(newlineOut, "\n") {
		t.Fatalf("expected newline preserved or escaped, got %s", newlineOut)
	}
}
