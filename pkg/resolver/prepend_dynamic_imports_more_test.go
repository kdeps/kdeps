package resolver

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestPrependDynamicImportsAddsLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logger,
		Context:        ctx,
		ActionDir:      "/action",
		RequestID:      "rid",
		RequestPklFile: "/action/api/rid__request.pkl",
	}

	// Ensure directories exist for any file existence checks.
	_ = fs.MkdirAll("/action/llm", 0o755)
	_ = fs.MkdirAll("/action/client", 0o755)
	_ = fs.MkdirAll("/action/exec", 0o755)
	_ = fs.MkdirAll("/action/python", 0o755)
	_ = fs.MkdirAll("/action/data", 0o755)

	// Create the target PKL file containing an amends line.
	pklPath := "/tmp/test.pkl"
	content := "amends \"base.pkl\"\n\noutput = @(`echo hello`)\n"
	if err := afero.WriteFile(fs, pklPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write pkl: %v", err)
	}

	if err := dr.PrependDynamicImports(pklPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After modification, the file should contain at least one import line we expect (e.g., utils)
	data, err := afero.ReadFile(fs, pklPath)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if !containsImport(string(data)) {
		t.Fatalf("expected import lines to be added, got:\n%s", string(data))
	}
}

// helpers
func containsImport(s string) bool {
	return strings.Contains(s, "import \"package://schema.kdeps.com") || strings.Contains(s, "import \"/action")
}
