package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklres "github.com/kdeps/kdeps/pkg/pklres"
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

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:")
	dr.PklresHelper = NewPklresHelper(dr)

	// Call the function under test
	if err := dr.PrepareImportFiles(); err != nil {
		t.Fatalf("PrepareImportFiles error: %v", err)
	}

	// Verify a python record exists in pklres
	_, err := dr.PklresHelper.retrievePklContent("python", "")
	if err != nil {
		t.Fatalf("expected python resource in pklres: %v", err)
	}
}
