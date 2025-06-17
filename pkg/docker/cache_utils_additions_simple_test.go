package docker

import (
	"context"
	"runtime"
	"testing"
)

func TestCompareVersionsAdditional(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		v1, v2 string
		expect bool // true if v1 > v2
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.3.0", false},
		{"2.0", "1.999.999", true},
		{"1.2.3-alpha", "1.2.3", false},
	}
	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.expect {
			t.Fatalf("CompareVersions(%s,%s)=%v want %v", c.v1, c.v2, got, c.expect)
		}
	}
}

func TestGetCurrentArchitectureAdditional(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	if runtime.GOARCH == "amd64" {
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping for amd64 runtime, got %s", arch)
		}
	}
	// arm64 maps to aarch64 for apple/pkl mapping, verify deterministically
	fakeCtx := context.Background()
	expectedDefault := runtime.GOARCH
	if mapping, ok := archMappings["default"]; ok {
		if mapped, ok2 := mapping[runtime.GOARCH]; ok2 {
			expectedDefault = mapped
		}
	}
	got := GetCurrentArchitecture(fakeCtx, "unknown/repo")
	if got != expectedDefault {
		t.Fatalf("unexpected default mapping: got %s want %s", got, expectedDefault)
	}
}

func TestBuildURLAdditional(t *testing.T) {
	base := "https://example.com/{version}/{arch}/bin"
	out := buildURL(base, "v1.0.0", "x86_64")
	expected := "https://example.com/v1.0.0/x86_64/bin"
	if out != expected {
		t.Fatalf("buildURL mismatch got %s want %s", out, expected)
	}
}
