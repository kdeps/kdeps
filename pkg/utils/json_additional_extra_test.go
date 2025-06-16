package utils

import "testing"

func TestIsJSONSimpleExtra(t *testing.T) {
	valid := `{"foo":123}`
	invalid := `{"foo":}`

	if !IsJSON(valid) {
		t.Fatalf("expected valid JSON to return true")
	}
	if IsJSON(invalid) {
		t.Fatalf("expected invalid JSON to return false")
	}
}

func TestFixJSONVariousExtra(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"rawStringWithNewline", `{"msg":"hello\nworld"}`},
		{"rawStringWithInteriorQuote", `{"quote":"He said "hello""}`},
		{"alreadyValid", `{"x":1}`},
		{"wrappedInQuotes", `"{\"y\":2}"`},
	}

	for _, c := range cases {
		output := FixJSON(c.input)
		if !IsJSON(output) {
			t.Fatalf("case %s produced invalid JSON: %s", c.name, output)
		}
	}
}
