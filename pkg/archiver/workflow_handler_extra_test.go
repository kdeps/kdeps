package archiver

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestFindWorkflowFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Setup mock directory structure
	baseDir := "/project"
	workflowDir := filepath.Join(baseDir, "sub")
	pklPath := filepath.Join(workflowDir, "workflow.pkl")

	if err := fs.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := afero.WriteFile(fs, pklPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Positive case
	found, err := FindWorkflowFile(fs, baseDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != pklPath {
		t.Errorf("expected %s, got %s", pklPath, found)
	}

	// Negative case: directory without workflow.pkl
	emptyDir := "/empty"
	if err := fs.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}
	if _, err := FindWorkflowFile(fs, emptyDir, logger); err == nil {
		t.Errorf("expected error for missing workflow.pkl, got nil")
	}
}
