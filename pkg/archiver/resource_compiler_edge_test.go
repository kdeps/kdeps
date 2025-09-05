package archiver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/project"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// stubWf implements the workflow.Workflow interface with minimal logic needed for tests.
// Only Name and Version are significant for transformation functions; all other methods return zero values.
type stubWf struct{}

func (stubWf) GetAgentID() string            { return "agent" }
func (stubWf) GetDescription() string        { return "" }
func (stubWf) GetWebsite() *string           { return nil }
func (stubWf) GetAuthors() *[]string         { return nil }
func (stubWf) GetDocumentation() *string     { return nil }
func (stubWf) GetRepository() *string        { return nil }
func (stubWf) GetHeroImage() *string         { return nil }
func (stubWf) GetAgentIcon() *string         { return nil }
func (stubWf) GetVersion() string            { return "1.2.3" }
func (stubWf) GetTargetActionID() string     { return "" }
func (stubWf) GetWorkflows() []string        { return nil }
func (stubWf) GetSettings() project.Settings { return project.Settings{} }

// Ensure interface compliance at compile-time.
var (
	_ pklWf.Workflow = stubWf{}
	_ interface {
		GetAgentID() string
		GetVersion() string
	} = stubWf{}
)

func TestHandleRequiresBlockEdge(t *testing.T) {
	wf := stubWf{}
	in := "\"data\"\n\"@other/act\"\n\"@agent/act:4.5.6\"\n\"\""
	out := handleRequiresBlock(in, wf)
	if !strings.Contains(out, "@agent/data:1.2.3") {
		t.Fatalf("expected namespaced data, got %s", out)
	}
	if !strings.Contains(out, "@act:1.2.3") {
		t.Fatalf("expected version appended to external id, got %s", out)
	}
	if !strings.Contains(out, "@agent/act:4.5.6") {
		t.Fatalf("explicit version should remain unchanged")
	}
}

func TestProcessActionPatternsEdge(t *testing.T) {
	line := `responseBody("someID")`
	got := processActionPatterns(line, "agent", "0.1.0")
	if !strings.Contains(got, "@agent/someID:0.1.0") {
		t.Fatalf("unexpected transform: %s", got)
	}

	orig := `response("@other/x:2.0.0")`
	if res := processActionPatterns(orig, "agent", "0.1.0"); res != orig {
		t.Fatalf("already qualified IDs should stay untouched")
	}
}

func TestProcessActionIDLineEdge(t *testing.T) {
	// Test the actionID processing logic that's now in processLine
	got, _ := processLine("actionID = \"myAction\"", "agent", "2.0.0")
	if !strings.Contains(got, "@agent/myAction:2.0.0") {
		t.Fatalf("expected namespaced id, got %s", got)
	}

	// Already namespaced should remain unchanged.
	original := "actionID = \"@other/that:1.1.1\""
	if res, _ := processLine(original, "agent", "2.0.0"); res != original {
		t.Fatalf("should not modify already namespaced string")
	}
}

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
