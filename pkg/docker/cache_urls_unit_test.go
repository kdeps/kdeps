package docker

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestBuildURLAndArchMapping(t *testing.T) {
	ctx := context.Background()

	// Verify buildURL replaces tokens correctly.
	input := "https://example.com/{version}/{arch}"
	got := buildURL(input, "v1", "x86_64")
	want := "https://example.com/v1/x86_64"
	if got != want {
		t.Fatalf("buildURL mismatch: got %s want %s", got, want)
	}

	// Check architecture mapping for apple/pkl and default.
	apple := GetCurrentArchitecture(ctx, "apple/pkl")
	def := GetCurrentArchitecture(ctx, "some/repo")

	switch runtime.GOARCH {
	case "amd64":
		if apple != "amd64" {
			t.Fatalf("expected amd64 for apple mapping, got %s", apple)
		}
		if def != "x86_64" {
			t.Fatalf("expected x86_64 for default mapping, got %s", def)
		}
	case "arm64":
		if apple != "aarch64" {
			t.Fatalf("expected aarch64 for apple mapping, got %s", apple)
		}
		if def != "aarch64" {
			t.Fatalf("expected aarch64 for default mapping, got %s", def)
		}
	}
}

func TestCompareVersionsAndParse(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		a, b    string
		greater bool
	}{
		{"1.2.3", "1.2.2", true},
		{"1.2", "1.2.0", false},
		{"2.0.0", "2.0.0", false},
		{"1.10", "1.9", true}, // numeric comparison not lexicographic
	}

	for _, c := range cases {
		got := CompareVersions(ctx, c.a, c.b)
		if got != c.greater {
			t.Fatalf("CompareVersions(%s,%s) = %v want %v", c.a, c.b, got, c.greater)
		}
	}

	// parseVersion edge validation
	parts := parseVersion("10.20.3-alpha")
	if len(parts) < 3 || parts[0] != 10 || parts[1] != 20 {
		t.Fatalf("parseVersion unexpected result: %v", parts)
	}
}

func TestGenerateURLsStatic(t *testing.T) {
	// schema.UseLatest is false by default – exercise happy-path generation.
	ctx := context.Background()
	urls, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs unexpected error: %v", err)
	}
	if len(urls) == 0 {
		t.Fatalf("expected at least one URL item")
	}

	for _, it := range urls {
		if it.LocalName == "" {
			t.Fatalf("expected LocalName for item %+v", it)
		}
		if !strings.HasPrefix(it.URL, "http") {
			t.Fatalf("URL should start with http: %s", it.URL)
		}
	}
}
