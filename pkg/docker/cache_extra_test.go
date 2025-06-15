package docker

import (
	"context"
	"runtime"
	"testing"
)

func TestGetCurrentArchitectureMappingExtra(t *testing.T) {
	ctx := context.Background()
	repo := "apple/pkl"
	arch := GetCurrentArchitecture(ctx, repo)
	// Validate against mapping table.
	goArch := runtime.GOARCH
	expected := archMappings[repo][goArch]
	if expected == "" {
		expected = archMappings["default"][goArch]
		if expected == "" {
			expected = goArch
		}
	}
	if arch != expected {
		t.Fatalf("expected %s, got %s", expected, arch)
	}
}

func TestCompareVersionsExtra(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		v1, v2 string
		newer  bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.3.0", false},
		{"2.0", "1.9.9", true},
		{"1.0.0", "1.0", false},
	}
	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.newer {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.v1, c.v2, got, c.newer)
		}
	}
}

func TestBuildURLExtra(t *testing.T) {
	url := buildURL("https://example.com/{version}/bin-{arch}", "v1.0", "x86_64")
	expected := "https://example.com/v1.0/bin-x86_64"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}
