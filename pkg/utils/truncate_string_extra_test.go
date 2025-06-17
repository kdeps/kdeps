package utils

import "testing"

func TestTruncateStringEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},    // shorter than max
		{"longstring", 4, "l..."}, // truncated with ellipsis
		{"abc", 2, "..."},         // max <3, replace with dots
	}
	for _, c := range cases {
		got := TruncateString(c.in, c.max)
		if got != c.want {
			t.Fatalf("TruncateString(%q,%d)=%q want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestSafeDerefHelpersExtra(t *testing.T) {
	str := "hi"
	if SafeDerefString(nil) != "" || SafeDerefString(&str) != "hi" {
		t.Fatalf("SafeDerefString failed")
	}
	b := true
	if SafeDerefBool(nil) || !SafeDerefBool(&b) {
		t.Fatalf("SafeDerefBool failed")
	}
}
