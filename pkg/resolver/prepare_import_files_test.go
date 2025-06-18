package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestPrepareImportFilesCreatesExpectedFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:          fs,
		Context:     context.Background(),
		ActionDir:   "/action",
		ProjectDir:  "/project",
		WorkflowDir: "/workflow",
		RequestID:   "graph1",
		Logger:      logging.NewTestLogger(),
	}

	// Call the function under test
	if err := dr.PrepareImportFiles(); err != nil {
		t.Fatalf("PrepareImportFiles error: %v", err)
	}

	// Verify that a known file now exists
	target := "/action/python/graph1__python_output.pkl"
	exists, err := afero.Exists(fs, target)
	if err != nil || !exists {
		t.Fatalf("expected file %s to exist", target)
	}
}
