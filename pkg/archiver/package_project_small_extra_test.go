package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
)

// stubWorkflow implements the required methods of pklWf.Workflow for this unit test.
type stubWorkflowPkg struct{}

func (stubWorkflowPkg) GetName() string                   { return "mini-agent" }
func (stubWorkflowPkg) GetVersion() string                { return "0.0.1" }
func (stubWorkflowPkg) GetDescription() string            { return "" }
func (stubWorkflowPkg) GetWebsite() *string               { return nil }
func (stubWorkflowPkg) GetAuthors() *[]string             { return nil }
func (stubWorkflowPkg) GetDocumentation() *string         { return nil }
func (stubWorkflowPkg) GetRepository() *string            { return nil }
func (stubWorkflowPkg) GetHeroImage() *string             { return nil }
func (stubWorkflowPkg) GetAgentIcon() *string             { return nil }
func (stubWorkflowPkg) GetTargetActionID() string         { return "run" }
func (stubWorkflowPkg) GetWorkflows() []string            { return nil }
func (stubWorkflowPkg) GetSettings() *pklProject.Settings { return nil }

func TestPackageProject_MinimalAndOverwrite(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	kdepsDir, _ := afero.TempDir(fs, "", "kdeps_sys")
	projectDir, _ := afero.TempDir(fs, "", "agent")

	// Minimal workflow file so EnforceFolderStructure passes.
	_ = afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte("name='x'\nversion='0.0.1'"), 0o644)

	wf := stubWorkflowPkg{}

	// First packaging.
	out1, err := PackageProject(fs, ctx, wf, kdepsDir, projectDir, logger)
	if err != nil {
		t.Fatalf("first PackageProject: %v", err)
	}
	if ok, _ := afero.Exists(fs, out1); !ok {
		t.Fatalf("package not created: %s", out1)
	}

	// Second packaging should overwrite.
	out2, err := PackageProject(fs, ctx, wf, kdepsDir, projectDir, logger)
	if err != nil {
		t.Fatalf("second PackageProject: %v", err)
	}
	if out1 != out2 {
		t.Fatalf("expected identical output path, got %s vs %s", out1, out2)
	}
}
