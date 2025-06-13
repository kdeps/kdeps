package resolver

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestAddPlaceholderImports_Errors(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmp := t.TempDir()
	actionDir := filepath.Join(tmp, "action")

	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logging.NewTestLogger(),
		ActionDir:      actionDir,
		DataDir:        filepath.Join(tmp, "data"),
		RequestID:      "req",
		RequestPklFile: filepath.Join(tmp, "request.pkl"),
	}

	// 1) file not found
	if err := dr.AddPlaceholderImports("/does/not/exist.pkl"); err == nil {
		t.Errorf("expected error for missing file path")
	}

	// 2) file without actionID line
	filePath := filepath.Join(tmp, "no_id.pkl")
	_ = afero.WriteFile(fs, filePath, []byte("extends \"package://schema.kdeps.com/core@1.0.0#/Exec.pkl\"\n"), 0o644)

	if err := dr.AddPlaceholderImports(filePath); err == nil {
		t.Errorf("expected error when action id missing but got nil")
	}
}
