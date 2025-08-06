package archiver

import (
	"strings"
	"testing"
)

func TestProcessActionIDLine(t *testing.T) {
	// Test the actionID processing logic that's now in processLine
	line := "actionID = \"myAction\""
	got, _ := processLine(line, "agent", "1.0.0")
	want := "actionID = \"@agent/myAction:1.0.0\""
	if got != want {
		t.Errorf("unexpected replacement: got %s want %s", got, want)
	}

	// Already prefixed with @ should be unchanged
	orig := "actionID = \"@agent/other:1.0.0\""
	if res, _ := processLine(orig, "agent", "1.0.0"); res != orig {
		t.Errorf("line should remain unchanged when already prefixed; got %s", res)
	}
}

func TestParseActionID(t *testing.T) {
	cases := []struct {
		action  string
		name    string
		version string
	}{
		{"@agent/foo:2.1.0", "agent", "2.1.0"},
		{"foo:3.0.0", "default", "3.0.0"},
		{"bar", "default", "1.2.3"},
	}
	for _, c := range cases {
		gotName, gotVersion := parseActionID(c.action, "default", "1.2.3")
		if gotName != c.name || gotVersion != c.version {
			t.Errorf("parseActionID(%s) got (%s,%s) want (%s,%s)", c.action, gotName, gotVersion, c.name, c.version)
		}
	}
}

func TestProcessActionPatterns(t *testing.T) {
	line := "responseHeader(\"foo\", \"bar\")"
	out := processActionPatterns(line, "agent", "1.0.0")
	if out == line {
		t.Errorf("expected replacement in responseHeader pattern")
	}
	if wantSub := "@agent/foo:1.0.0"; !contains(out, wantSub) {
		t.Errorf("expected %s in output %s", wantSub, out)
	}
}

func contains(s, substr string) bool { return strings.Contains(s, substr) }
