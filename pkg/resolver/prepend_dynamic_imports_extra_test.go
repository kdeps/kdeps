package resolver

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
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
