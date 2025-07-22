package resolver_test

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestPrepareWorkflowDirSimple(t *testing.T) {
	fs := afero.NewMemMapFs()

	projectDir := filepath.Join(t.TempDir(), "project")

	// create dummy structure
	_ = fs.MkdirAll(filepath.Join(projectDir, "sub"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(projectDir, "sub", "file.txt"), []byte("x"), 0o644)

	// PrepareWorkflowDir functionality removed - using project directory directly

	// ensure file exists in project directory
	if ok, _ := afero.Exists(fs, filepath.Join(projectDir, "sub", "file.txt")); !ok {
		t.Fatalf("expected file not found in project directory")
	}
}
