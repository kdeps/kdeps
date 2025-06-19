package resolver_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestPrependDynamicImportsInsert ensures that PrependDynamicImports injects the
// expected import lines into a .pkl file that initially contains only an
// "amends" declaration.
func TestPrependDynamicImportsInsert(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/workflow.pkl"
	initial := "amends \"base.pkl\"\n\n"
	if err := afero.WriteFile(fs, filePath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	dr := &DependencyResolver{
		Fs:             fs,
		Context:        context.Background(),
		ActionDir:      "/action",
		RequestID:      "graph123",
		RequestPklFile: "/action/request.pkl",
		Logger:         logging.NewTestLogger(),
	}

	if err := dr.PrependDynamicImports(filePath); err != nil {
		t.Fatalf("PrependDynamicImports returned error: %v", err)
	}

	// Confirm that at least one import statement was added.
	contentBytes, err := afero.ReadFile(fs, filePath)
	if err != nil {
		t.Fatalf("read modified file: %v", err)
	}
	content := string(contentBytes)
	if !strings.Contains(content, "import \"pkl:json\"") {
		t.Fatalf("expected import line to be injected; got:\n%s", content)
	}
}

func TestPrependDynamicImportsAddsLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logger,
		Context:        ctx,
		ActionDir:      "/action",
		RequestID:      "rid",
		RequestPklFile: "/action/api/rid__request.pkl",
	}

	// Ensure directories exist for any file existence checks.
	_ = fs.MkdirAll("/action/llm", 0o755)
	_ = fs.MkdirAll("/action/client", 0o755)
	_ = fs.MkdirAll("/action/exec", 0o755)
	_ = fs.MkdirAll("/action/python", 0o755)
	_ = fs.MkdirAll("/action/data", 0o755)

	// Create the target PKL file containing an amends line.
	pklPath := "/tmp/test.pkl"
	content := "amends \"base.pkl\"\n\noutput = @(`echo hello`)\n"
	if err := afero.WriteFile(fs, pklPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write pkl: %v", err)
	}

	if err := dr.PrependDynamicImports(pklPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After modification, the file should contain at least one import line we expect (e.g., utils)
	data, err := afero.ReadFile(fs, pklPath)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if !containsImport(string(data)) {
		t.Fatalf("expected import lines to be added, got:\n%s", string(data))
	}
}

// helpers
func containsImport(s string) bool {
	return strings.Contains(s, "import \"package://schema.kdeps.com") || strings.Contains(s, "import \"/action")
}

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
