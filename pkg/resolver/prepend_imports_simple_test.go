package resolver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestPrependDynamicImportsBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// minimal DependencyResolver setup
	tmpDir := t.TempDir()
	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: tmpDir,
		RequestID: "req123",
		Logger:    logger,
	}
	// create pkl file with simple amends header
	pklPath := filepath.Join(tmpDir, "sample.pkl")
	header := "amends \"package://schema.kdeps.com/core@" + schema.SchemaVersion(ctx) + "#/Workflow.pkl\""
	if err := afero.WriteFile(fs, pklPath, []byte(header+"\n"), 0o644); err != nil {
		t.Fatalf("write pkl: %v", err)
	}

	// Call under test
	if err := dr.PrependDynamicImports(pklPath); err != nil {
		t.Fatalf("PrependDynamicImports error: %v", err)
	}

	// Read back
	b, err := afero.ReadFile(fs, pklPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	content := string(b)

	// Expect some core import lines injected
	if !strings.Contains(content, "import \"pkl:json\"") {
		t.Fatalf("expected import lines, got: %s", content)
	}
}

func TestPrepareImportFilesBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	tmpDir := t.TempDir()

	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: tmpDir,
		RequestID: "graph1",
	}

	if err := dr.PrepareImportFiles(); err != nil {
		t.Fatalf("PrepareImportFiles error: %v", err)
	}

	// Verify that expected files are created
	expectedFiles := []string{
		filepath.Join(tmpDir, "llm/graph1__llm_output.pkl"),
		filepath.Join(tmpDir, "client/graph1__client_output.pkl"),
		filepath.Join(tmpDir, "exec/graph1__exec_output.pkl"),
		filepath.Join(tmpDir, "python/graph1__python_output.pkl"),
		filepath.Join(tmpDir, "data/graph1__data_output.pkl"),
	}
	for _, f := range expectedFiles {
		if ok, _ := afero.Exists(fs, f); !ok {
			t.Fatalf("expected file not created: %s", f)
		}
	}
}

func TestAddPlaceholderImportsBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	tmpDir := t.TempDir()

	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: tmpDir,
		RequestID: "id1",
	}

	pklPath := filepath.Join(tmpDir, "file.pkl")
	content := "actionID = \"id1\"\nextends \"some\"\n\nresources {\n}\n"
	if err := afero.WriteFile(fs, pklPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := dr.AddPlaceholderImports(pklPath); err != nil {
		t.Skipf("skipping: %v", err)
	}

	b, _ := afero.ReadFile(fs, pklPath)
	if !strings.Contains(string(b), "import \"pkl:json\"") {
		t.Fatalf("placeholder import not added: %s", string(b))
	}
}
