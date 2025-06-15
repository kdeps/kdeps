package docker

import (
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestGenerateParamsSectionLight(t *testing.T) {
	params := map[string]string{
		"FOO": "bar",
		"BAZ": "", // param without value
	}
	got := generateParamsSection("ENV", params)
	if !containsAll(got, []string{"ENV FOO=\"bar\"", "ENV BAZ"}) {
		t.Fatalf("unexpected section: %s", got)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestGenerateUniqueOllamaPortLight(t *testing.T) {
	p1 := generateUniqueOllamaPort(3000)
	p2 := generateUniqueOllamaPort(3000)
	if p1 == p2 {
		t.Fatalf("expected different ports when called twice, got %s %s", p1, p2)
	}
}

func TestCheckDevBuildModeLight(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	kdepsDir := "/kd"
	// No cache/kdeps binary present -> dev build mode should be false.
	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil || ok {
		t.Fatalf("expected false dev mode, got %v %v", ok, err)
	}

	// Simulate presence of a downloaded kdeps binary to enable dev build mode.
	if err := fs.MkdirAll("/kd/cache", 0o755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}
	_ = afero.WriteFile(fs, "/kd/cache/kdeps", []byte("binary"), 0o755)

	ok, err = checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil || !ok {
		t.Fatalf("expected dev mode true, got %v %v", ok, err)
	}
}
