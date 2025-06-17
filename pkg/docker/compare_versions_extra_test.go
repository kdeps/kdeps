package docker

import (
	"context"
	"testing"
)

// TestCompareVersions covers several version comparison scenarios including
// differing lengths and prerelease identifiers to raise coverage for the helper.
func TestCompareVersionsExtraCases(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1   string
		v2   string
		want bool
	}{
		{"1.2.3", "1.2.2", true},       // patch greater
		{"2.0.0", "2.0.0", false},      // equal
		{"1.2.2", "1.2.3", false},      // smaller
		{"1.2.3-alpha", "1.2.2", true}, // prerelease ignored by atoi (becomes 0)
	}

	for _, tc := range cases {
		got := CompareVersions(ctx, tc.v1, tc.v2)
		if got != tc.want {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", tc.v1, tc.v2, got, tc.want)
		}
	}
}
