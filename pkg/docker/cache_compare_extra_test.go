package docker

import (
	"context"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
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
	_ = schema.SchemaVersion(ctx)
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

func TestParseVersionParts(t *testing.T) {
	got := parseVersion("1.2.3")
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("expected length %d, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseVersion mismatch at index %d: want %d got %d", i, want[i], got[i])
		}
	}
}

func TestCompareVersionsEdge(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2 string
		want   bool
	}{
		{"1.2.3", "1.2.2", true},  // greater
		{"2.0.0", "2.0.0", false}, // equal
		{"1.0.0", "1.0.1", false}, // less
		{"1.10", "1.9", true},     // numeric compare not lexicographic
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.v1, c.v2)
		if got != c.want {
			t.Errorf("CompareVersions(%s,%s) = %v, want %v", c.v1, c.v2, got, c.want)
		}
	}
}

func TestBuildURLReplacer(t *testing.T) {
	base := "https://example.com/{version}/{arch}/download"
	url := buildURL(base, "1.0.0", "x86_64")
	expected := "https://example.com/1.0.0/x86_64/download"
	if url != expected {
		t.Fatalf("buildURL mismatch: got %s, want %s", url, expected)
	}
}

func TestGetCurrentArchitectureDefault(t *testing.T) {
	ctx := context.Background()
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	_ = schema.SchemaVersion(ctx)

	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping for arm64, got %s", arch)
		}
	default:
		if arch != runtime.GOARCH {
			t.Fatalf("expected arch to match runtime (%s), got %s", runtime.GOARCH, arch)
		}
	}
}
