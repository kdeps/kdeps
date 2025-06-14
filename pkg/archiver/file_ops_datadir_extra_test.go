package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
)

// mockWorkflow implements the minimal subset of the generated Workflow interface we need.
type mockWorkflow struct{ name, version string }

func (m mockWorkflow) GetName() string                   { return m.name }
func (m mockWorkflow) GetVersion() string                { return m.version }
func (m mockWorkflow) GetDescription() string            { return "" }
func (m mockWorkflow) GetWebsite() *string               { return nil }
func (m mockWorkflow) GetAuthors() *[]string             { return nil }
func (m mockWorkflow) GetDocumentation() *string         { return nil }
func (m mockWorkflow) GetRepository() *string            { return nil }
func (m mockWorkflow) GetHeroImage() *string             { return nil }
func (m mockWorkflow) GetAgentIcon() *string             { return nil }
func (m mockWorkflow) GetTargetActionID() string         { return "" }
func (m mockWorkflow) GetWorkflows() []string            { return nil }
func (m mockWorkflow) GetSettings() *pklProject.Settings { return nil }

// TestCopyDataDirBasic verifies that CopyDataDir copies files when present.
func TestCopyDataDirBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	compiledDir := filepath.Join(tmp, "compiled")

	// create source data file at projectDir/data/<wf.name>/<wf.version>/file.txt
	wf := mockWorkflow{"agent", "1.0.0"}
	dataSrc := filepath.Join(projectDir, "data")
	if err := fs.MkdirAll(dataSrc, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(dataSrc, "sample.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := fs.MkdirAll(compiledDir, 0o755); err != nil {
		t.Fatalf("mkdir compiled: %v", err)
	}

	kdepsDir := filepath.Join(tmp, "kdeps")

	if err := CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledDir, "", "", "", false, logger); err != nil {
		t.Fatalf("CopyDataDir error: %v", err)
	}

	destFile := filepath.Join(compiledDir, "data", wf.GetName(), wf.GetVersion(), "sample.txt")
	if ok, _ := afero.Exists(fs, destFile); !ok {
		t.Fatalf("destination file not copied")
	}

	_ = schema.SchemaVersion(ctx)
}

// TestResolveAgentVersionAndCopyResources verifies resource copy logic and auto-version bypass.
func TestResolveAgentVersionAndCopyResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	tmp := t.TempDir()
	kdepsDir := filepath.Join(tmp, "kdeps")
	compiledDir := filepath.Join(tmp, "compiled")

	// Set up resources src path kdepsDir/agents/agent/1.2.3/resources/res.txt
	resourcesDir := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "resources")
	if err := fs.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatalf("mkdir res: %v", err)
	}
	_ = afero.WriteFile(fs, filepath.Join(resourcesDir, "res.txt"), []byte("r"), 0o644)

	// And data path which function returns
	dataFile := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "data", "agent", "1.2.3", "d.txt")
	if err := fs.MkdirAll(filepath.Dir(dataFile), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	_ = afero.WriteFile(fs, dataFile, []byte("d"), 0o644)

	if err := fs.MkdirAll(compiledDir, 0o755); err != nil {
		t.Fatalf("mkdir compiled: %v", err)
	}

	newSrc, newDst, err := ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledDir, "agent", "1.2.3", logger)
	if err != nil {
		t.Fatalf("ResolveAgentVersion error: %v", err)
	}

	// The resources should now be copied into compiledDir/resources/res.txt
	if ok, _ := afero.Exists(fs, filepath.Join(compiledDir, "resources", "res.txt")); !ok {
		t.Fatalf("resource not copied")
	}

	// Returned paths should match expected data directories.
	expectedSrc := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "data", "agent", "1.2.3")
	expectedDst := filepath.Join(compiledDir, "data", "agent", "1.2.3")
	if newSrc != expectedSrc || newDst != expectedDst {
		t.Fatalf("unexpected src/dst: %s %s", newSrc, newDst)
	}

	_ = schema.SchemaVersion(ctx)
}
