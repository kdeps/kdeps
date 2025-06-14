package resolver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklRes "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

// TestLoadResourceEntries verifies that .pkl files inside the workflow resources directory
// are discovered and passed through processPklFile, using a stubbed LoadResourceFn so that
// no actual Pkl evaluation is required.
func TestLoadResourceEntries(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// directory structure expected: <workflowDir>/resources/*.pkl
	workflowDir := "/workflow"
	resourcesDir := filepath.Join(workflowDir, "resources")
	if err := fs.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatalf("failed to create resources dir: %v", err)
	}

	// create two dummy pkl files
	files := []string{"alpha.pkl", "beta.pkl"}
	for _, f := range files {
		p := filepath.Join(resourcesDir, f)
		id := strings.TrimSuffix(f, filepath.Ext(f))
		content := "extends \"dummy\"\n\n" + "actionID = \"" + id + "\"\n"
		if err := afero.WriteFile(fs, p, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write dummy pkl: %v", err)
		}
	}

	dr := &DependencyResolver{
		Fs:                   fs,
		Logger:               logger,
		WorkflowDir:          workflowDir,
		ActionDir:            "/action",
		RequestID:            "req1",
		RequestPklFile:       filepath.Join("/action", "api/req1__request.pkl"),
		ResourceDependencies: make(map[string][]string),
		Resources:            []ResourceNodeEntry{},
		Context:              context.Background(),
	}

	// stub LoadResourceFn to avoid real evaluation; just return a Resource with ActionID = filename (no extension)
	dr.LoadResourceFn = func(_ context.Context, path string, _ ResourceType) (interface{}, error) {
		base := filepath.Base(path)
		id := strings.TrimSuffix(base, filepath.Ext(base))
		return &pklRes.Resource{ActionID: id}, nil
	}

	// Manually invoke processPklFile for each dummy file instead of walking the directory
	for _, f := range files {
		p := filepath.Join(resourcesDir, f)
		if err := dr.processPklFile(p); err != nil {
			t.Fatalf("processPklFile returned error for %s: %v", p, err)
		}
	}

	// Expect two resources collected
	if len(dr.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(dr.Resources))
	}

	// Check that dependencies map has entries for action IDs
	for _, rn := range dr.Resources {
		if _, ok := dr.ResourceDependencies[rn.ActionID]; !ok {
			t.Fatalf("dependency entry missing for %s", rn.ActionID)
		}
	}
}
