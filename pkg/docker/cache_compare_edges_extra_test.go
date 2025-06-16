package docker

import (
	"context"
	"testing"
)

func TestCompareVersions_EdgeCases(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		a, b    string
		greater bool // whether a>b expected
	}{
		{"1.0.0-alpha", "1.0.0", false},
		{"1.0.1", "1.0.0-beta", true},
		{"1.0", "1.0.0", false},
		{"2", "10", false},
		{"0.0.0", "0", false},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.a, c.b, got, c.greater)
		}
	}
}
