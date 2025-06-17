package docker

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestGenerateParamsSectionEdge(t *testing.T) {
	items := map[string]string{
		"FOO":   "bar",
		"EMPTY": "",
	}
	out := generateParamsSection("ARG", items)

	if !strings.Contains(out, "ARG FOO=\"bar\"") {
		t.Fatalf("missing value param: %s", out)
	}
	if !strings.Contains(out, "ARG EMPTY") {
		t.Fatalf("missing empty param: %s", out)
	}
}

func TestCheckDevBuildModeMem(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	base := t.TempDir()
	kdepsDir := filepath.Join(base, "home")
	cacheDir := filepath.Join(kdepsDir, "cache")
	_ = fs.MkdirAll(cacheDir, 0o755)

	// Case 1: file absent => devBuildMode false
	dev, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dev {
		t.Fatalf("expected dev build mode to be false when file missing")
	}

	// Create dummy kdeps binary file
	filePath := filepath.Join(cacheDir, "kdeps")
	if err := afero.WriteFile(fs, filePath, []byte("hi"), 0o755); err != nil {
		t.Fatalf("write file: %v", err)
	}

	dev2, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dev2 {
		t.Fatalf("expected dev build mode true when file exists")
	}
}
