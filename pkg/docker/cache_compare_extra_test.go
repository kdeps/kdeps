package docker

import (
	"context"
	"runtime"
	"testing"
)

func TestCompareVersionsMore(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2  string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2.0", "1.2", false},
		{"1.2.10", "1.3", false},
		{"2.0.0", "2.0.0", false},
		{"1.2.3-alpha", "1.2.3", false},
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.greater {
			t.Errorf("CompareVersions(%s,%s)=%v, want %v", c.v1, c.v2, got, c.greater)
		}
	}
}

func TestGetCurrentArchitectureMapping(t *testing.T) {
	ctx := context.Background()

	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	want := map[string]string{"amd64": "amd64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch != want {
		t.Errorf("mapping mismatch for apple/pkl: got %s want %s", arch, want)
	}

	// default mapping path
	arch2 := GetCurrentArchitecture(ctx, "unknown/repo")
	def := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[runtime.GOARCH]
	if arch2 != def {
		t.Errorf("default mapping mismatch: got %s want %s", arch2, def)
	}
}
