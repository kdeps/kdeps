package docker_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/docker"
)

func TestGetCurrentArchitecture(t *testing.T) {
	ctx := context.Background()

	arch := docker.GetCurrentArchitecture(ctx, "apple/pkl")
	switch runtime.GOARCH {
	case "amd64":
		if arch != "amd64" {
			t.Fatalf("expected amd64 mapping, got %s", arch)
		}
	case "arm64":
		if arch != "aarch64" {
			t.Fatalf("expected aarch64 mapping for arm64 host, got %s", arch)
		}
	default:
		if arch != runtime.GOARCH {
			t.Fatalf("expected passthrough architecture %s, got %s", runtime.GOARCH, arch)
		}
	}

	// Unknown repo should fallback to default mapping
	arch = docker.GetCurrentArchitecture(ctx, "some/unknown")
	expected := runtime.GOARCH
	if runtime.GOARCH == "amd64" {
		expected = "x86_64"
	} else if runtime.GOARCH == "arm64" {
		expected = "aarch64"
	}
	if arch != expected {
		t.Fatalf("expected %s for default mapping, got %s", expected, arch)
	}
}

func TestCompareVersions(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		v1, v2  string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},  // higher patch
		{"1.3.0", "1.2.9", true},  // higher minor
		{"2.0.0", "1.9.9", true},  // higher major
		{"1.0.0", "1.0.0", false}, // equal
		{"1.2.3", "2.0.0", false}, // lower major
		{"1.2", "1.2.1", false},   // shorter version string
	}

	for _, c := range cases {
		got := docker.CompareVersions(ctx, c.v1, c.v2)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s) = %v, want %v", c.v1, c.v2, got, c.greater)
		}
	}
}

// No test for buildURL because it is an unexported helper; its
// behaviour is implicitly covered by higher-level GenerateURLs tests.
