package resolver_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklres "github.com/kdeps/kdeps/pkg/pklres"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

func TestPrepareImportFilesCreatesExpectedFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolverpkg.DependencyResolver{
		Fs:         fs,
		Context:    context.Background(),
		ActionDir:  "/action",
		ProjectDir: "/project",
		RequestID:  "graph1",
		Logger:     logging.NewTestLogger(),
	}

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph", "", "", "")
	dr.PklresHelper = resolverpkg.NewPklresHelper(dr)

	// Call the function under test
	if err := dr.PrepareImportFiles(); err != nil {
		t.Fatalf("PrepareImportFiles error: %v", err)
	}

	// Verify a python record exists in pklres
	_, err := dr.PklresHelper.RetrievePklContent("python", "__empty__")
	if err != nil {
		t.Fatalf("expected python resource in pklres: %v", err)
	}
}
