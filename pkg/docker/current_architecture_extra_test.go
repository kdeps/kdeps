package docker

import (
	"context"
	"runtime"
	"testing"
)

func TestGetCurrentArchitectureMappingNew(t *testing.T) {
	ctx := context.Background()

	// When repo matches mapping for apple/pkl
	arch := GetCurrentArchitecture(ctx, "apple/pkl")
	if runtime.GOARCH == "amd64" && arch != "amd64" {
		t.Fatalf("expected amd64 mapping, got %s", arch)
	}
	if runtime.GOARCH == "arm64" && arch != "aarch64" {
		t.Fatalf("expected aarch64 mapping, got %s", arch)
	}

	// Default mapping for unknown repo; should fall back to x86_64 mapping
	arch2 := GetCurrentArchitecture(ctx, "unknown/repo")
	expected := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}
	if got := expected[runtime.GOARCH]; arch2 != got {
		t.Fatalf("expected %s, got %s", got, arch2)
	}
}
