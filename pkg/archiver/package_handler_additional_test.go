package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklProj "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
)

// minimal workflow stub satisfying the two getters used by PackageProject.
type simpleWf struct{}

func (simpleWf) GetName() string    { return "agent" }
func (simpleWf) GetVersion() string { return "0.0.1" }

// Unused methods – provide zero values to satisfy interface.
func (simpleWf) GetDescription() string         { return "" }
func (simpleWf) GetWebsite() *string            { return nil }
func (simpleWf) GetAuthors() *[]string          { return nil }
func (simpleWf) GetDocumentation() *string      { return nil }
func (simpleWf) GetRepository() *string         { return nil }
func (simpleWf) GetHeroImage() *string          { return nil }
func (simpleWf) GetAgentIcon() *string          { return nil }
func (simpleWf) GetTargetActionID() string      { return "" }
func (simpleWf) GetWorkflows() []string         { return nil }
func (simpleWf) GetSettings() *pklProj.Settings { return nil }

// compile-time assertion
var _ interface {
	GetName() string
	GetVersion() string
} = simpleWf{}

func TestPackageProjectHappyPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir := "/kdeps"
	compiled := "/compiled"

	// Create required structure.
	_ = fs.MkdirAll(filepath.Join(compiled, "resources"), 0o755)
	// minimal resource file
	_ = afero.WriteFile(fs, filepath.Join(compiled, "resources", "exec.pkl"), []byte("run { exec { ['x']='y' } }"), 0o644)
	// workflow file at root
	wfContent := `amends "package://schema.kdeps.com/core@0.0.0#/Workflow.pkl"`
	_ = afero.WriteFile(fs, filepath.Join(compiled, "workflow.pkl"), []byte(wfContent), 0o644)

	wf := simpleWf{}

	out, err := PackageProject(fs, ctx, wf, kdepsDir, compiled, logger)
	if err != nil {
		t.Fatalf("PackageProject returned error: %v", err)
	}
	exists, _ := afero.Exists(fs, out)
	if !exists {
		t.Fatalf("expected package file %s to exist", out)
	}
}

func TestPackageProjectMissingResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir := "/kdeps"
	compiled := "/badcompiled"
	// create compiled dir with unexpected file to violate folder structure rules
	_ = fs.MkdirAll(compiled, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(compiled, "unexpected.txt"), []byte("oops"), 0o644)

	_, err := PackageProject(fs, ctx, simpleWf{}, kdepsDir, compiled, logger)
	if err == nil {
		t.Fatalf("expected error when resources directory missing")
	}
}

func TestFindWorkflowFileSuccessAndFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	dir := "/proj"
	_ = fs.MkdirAll(dir, 0o755)
	// create file
	_ = afero.WriteFile(fs, filepath.Join(dir, "workflow.pkl"), []byte(""), 0o644)

	path, err := FindWorkflowFile(fs, dir, logger)
	if err != nil || filepath.Base(path) != "workflow.pkl" {
		t.Fatalf("expected to find workflow.pkl, got %s err %v", path, err)
	}

	// failure case
	emptyDir := "/empty"
	_ = fs.MkdirAll(emptyDir, 0o755)
	if _, err := FindWorkflowFile(fs, emptyDir, logger); err == nil {
		t.Fatalf("expected error when workflow file missing")
	}
}
