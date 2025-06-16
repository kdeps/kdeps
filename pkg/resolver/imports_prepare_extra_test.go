package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestPrepareImportFilesCreatesStubs(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:        fs,
		ActionDir: "/agent/action",
		RequestID: "abc",
		Context:   nil,
	}

	err := dr.PrepareImportFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for one of the generated files and its header content
	execPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
	content, err := afero.ReadFile(fs, execPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	header := fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"", schema.SchemaVersion(dr.Context))
	if !strings.Contains(string(content), header) {
		t.Errorf("header not found in file: %s", execPath)
	}
}
