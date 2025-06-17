package docker

import (
	"context"
	"runtime"
	"testing"
)

func TestCompareVersionsOrdering(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		a, b          string
		expectABigger bool
	}{
		{"1.2.3", "1.2.2", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"1.10.0", "1.9.9", true},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.expectABigger {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", c.a, c.b, got, c.expectABigger)
		}
	}
}

func TestGetCurrentArchitectureMappingCov(t *testing.T) {
	ctx := context.Background()

	arch := GetCurrentArchitecture(ctx, "apple/pkl")

	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping, got %s", arch)
		}
	}
}

func TestBuildURLTemplateSubstitution(t *testing.T) {
	base := "https://example.com/download/{version}/bin-{arch}"
	url := buildURL(base, "v1.2.3", "x86_64")
	expected := "https://example.com/download/v1.2.3/bin-x86_64"
	if url != expected {
		t.Fatalf("buildURL produced %s, want %s", url, expected)
	}
}
