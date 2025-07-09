package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/project"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// stubWf implements the workflow.Workflow interface with minimal logic needed for tests.
// Only Name and Version are significant for transformation functions; all other methods return zero values.
type stubWf struct{}

func (stubWf) GetAgentID() string             { return "agent" }
func (stubWf) GetDescription() *string        { desc := ""; return &desc }
func (stubWf) GetWebsite() *string            { return nil }
func (stubWf) GetAuthors() *[]string          { return nil }
func (stubWf) GetDocumentation() *string      { return nil }
func (stubWf) GetRepository() *string         { return nil }
func (stubWf) GetHeroImage() *string          { return nil }
func (stubWf) GetAgentIcon() *string          { return nil }
func (stubWf) GetVersion() string             { return "1.2.3" }
func (stubWf) GetTargetActionID() string      { return "" }
func (stubWf) GetWorkflows() []string         { return nil }
func (stubWf) GetSettings() *project.Settings { return nil }

// Ensure interface compliance at compile-time.
var (
	_ pklWf.Workflow = stubWf{}
	_ interface {
		GetAgentID() string
		GetVersion() string
	} = stubWf{}
)

func TestStubWfAllMethods(t *testing.T) {
	wf := stubWf{}
	if wf.GetAgentID() == "" || wf.GetVersion() == "" {
		t.Fatalf("name or version empty")
	}
	_ = wf.GetDescription()
	_ = wf.GetWebsite()
	_ = wf.GetAuthors()
	_ = wf.GetDocumentation()
	_ = wf.GetRepository()
	_ = wf.GetHeroImage()
	_ = wf.GetAgentIcon()
	_ = wf.GetTargetActionID()
	_ = wf.GetWorkflows()
	_ = wf.GetSettings()
}

func TestValidatePklResourcesMissingDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := ValidatePklResources(fs, ctx, "/not/exist", logger)
	if err == nil {
		t.Fatalf("expected error on missing directory")
	}
}

func TestCollectPklFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := "/pkl"
	_ = fs.MkdirAll(dir, 0o755)
	// create pkl and non-pkl files
	_ = afero.WriteFile(fs, filepath.Join(dir, "a.pkl"), []byte("x"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(dir, "b.txt"), []byte("y"), 0o644)

	files, err := collectPklFiles(fs, dir)
	if err != nil {
		t.Fatalf("collectPklFiles error: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "a.pkl" {
		t.Fatalf("unexpected files slice: %v", files)
	}
}
