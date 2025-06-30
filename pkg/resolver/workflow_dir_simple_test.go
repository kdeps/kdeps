package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestPrepareWorkflowDirSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	projectDir := filepath.Join(t.TempDir(), "project")
	wfDir := filepath.Join(t.TempDir(), "workflow")

	// create dummy structure
	_ = fs.MkdirAll(filepath.Join(projectDir, "sub"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(projectDir, "sub", "file.txt"), []byte("x"), 0o644)

	dr := &DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ProjectDir:  projectDir,
		WorkflowDir: wfDir,
	}

	if err := dr.PrepareWorkflowDir(); err != nil {
		t.Fatalf("PrepareWorkflowDir error: %v", err)
	}

	// ensure file copied
	if ok, _ := afero.Exists(fs, filepath.Join(wfDir, "sub", "file.txt")); !ok {
		t.Fatalf("expected file not copied")
	}
}
