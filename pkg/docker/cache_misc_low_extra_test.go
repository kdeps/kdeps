package docker

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
)

func TestBuildURLAndArchMappingLow(t *testing.T) {
	_ = schema.SchemaVersion(context.Background())

	base := "https://example.com/{version}/{arch}/binary"
	url := buildURL(base, "1.2.3", "x86_64")
	want := "https://example.com/1.2.3/x86_64/binary"
	if url != want {
		t.Fatalf("buildURL mismatch: got %s want %s", url, want)
	}

	arch := runtime.GOARCH // expect mapping fall-through works
	ctx := context.Background()
	got := GetCurrentArchitecture(ctx, "unknown/repo")
	var expect string
	if m, ok := archMappings["default"]; ok {
		if v, ok2 := m[arch]; ok2 {
			expect = v
		} else {
			expect = arch
		}
	}
	if got != expect {
		t.Fatalf("GetCurrentArchitecture fallback = %s; want %s", got, expect)
	}
}

func TestGenerateURLs_NoLatestLow(t *testing.T) {
	// Ensure UseLatest is false for deterministic output
	schema.UseLatest = false
	ctx := context.Background()
	urls, err := GenerateURLs(ctx)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	if len(urls) == 0 {
		t.Fatalf("expected some URLs")
	}

	// Each item should have LocalName containing version, not "latest"
	for _, it := range urls {
		if strings.Contains(it.LocalName, "latest") {
			t.Fatalf("LocalName should not contain 'latest' when UseLatest=false: %s", it.LocalName)
		}
		if it.URL == "" || it.LocalName == "" {
			t.Fatalf("got empty fields in item %+v", it)
		}
	}
}
