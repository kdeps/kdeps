package enforcer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// createFiles helper creates nested files and dirs on provided fs.
func createFiles(t *testing.T, fsys afero.Fs, paths []string) {
	for _, p := range paths {
		dir := filepath.Dir(p)
		if err := fsys.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := afero.WriteFile(fsys, p, []byte("data"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func TestEnforceFolderStructure_Happy(t *testing.T) {
	fsys := afero.NewOsFs()
	tmpDir := t.TempDir()

	// required layout
	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "data", "agent", "1.0", "file.txt"),
	})

	if err := EnforceFolderStructure(fsys, context.Background(), tmpDir, logging.NewTestLogger()); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	_ = schema.SchemaVersion(context.Background())
}

func TestEnforceFolderStructure_BadExtraDir(t *testing.T) {
	fsys := afero.NewOsFs()
	tmpDir := t.TempDir()

	createFiles(t, fsys, []string{
		filepath.Join(tmpDir, "workflow.pkl"),
		filepath.Join(tmpDir, "resources", "foo.pkl"),
		filepath.Join(tmpDir, "extras", "bad.txt"),
	})

	if err := EnforceFolderStructure(fsys, context.Background(), tmpDir, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for unexpected folder")
	}

	_ = schema.SchemaVersion(context.Background())
}

func TestEnforcePklTemplateAmendsRules(t *testing.T) {
	fsys := afero.NewOsFs()
	tmp := t.TempDir()
	validFile := filepath.Join(tmp, "workflow.pkl")
	content := "amends \"package://schema.kdeps.com/core@" + schema.SchemaVersion(context.Background()) + "#/Workflow.pkl\"\n"
	if err := afero.WriteFile(fsys, validFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := EnforcePklTemplateAmendsRules(fsys, context.Background(), validFile, logging.NewTestLogger()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	invalidFile := filepath.Join(tmp, "bad.pkl")
	if err := afero.WriteFile(fsys, invalidFile, []byte("invalid line\n"), 0o644); err != nil {
		t.Fatalf("write2: %v", err)
	}
	if err := EnforcePklTemplateAmendsRules(fsys, context.Background(), invalidFile, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for bad amends line")
	}
}
