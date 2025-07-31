package archiver

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklProj "github.com/kdeps/schema/gen/project"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
)

// minimal workflow stub satisfying the two getters used by PackageProject.
type simpleWf struct{}

func (simpleWf) GetAgentID() string    { return "agent" }
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
	GetAgentID() string
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
	_ = afero.WriteFile(fs, filepath.Join(compiled, "resources", "exec.pkl"), []byte("run { Exec { ['x']='y' } }"), 0o644)
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

// We reuse stubWf from resource_compiler_edge_test for Workflow implementation.

// TestPrepareRunDir ensures archive extraction happens into expected run path.
func TestPrepareRunDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	wf := stubWf{}

	tmp := t.TempDir()
	kdepsDir := filepath.Join(tmp, "kdepssys")
	if err := os.MkdirAll(kdepsDir, 0o755); err != nil {
		t.Fatalf("mkdir kdepsDir: %v", err)
	}

	// Build minimal tar.gz archive containing a dummy file.
	pkgPath := filepath.Join(tmp, "pkg.kdeps")
	pkgFile, err := os.Create(pkgPath)
	if err != nil {
		t.Fatalf("create pkg: %v", err)
	}
	gz := gzip.NewWriter(pkgFile)
	tw := tar.NewWriter(gz)
	// add dummy.txt
	hdr := &tar.Header{Name: "dummy.txt", Mode: 0o644, Size: int64(len("hi"))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("hdr: %v", err)
	}
	if _, err := io.WriteString(tw, "hi"); err != nil {
		t.Fatalf("write: %v", err)
	}
	tw.Close()
	gz.Close()
	pkgFile.Close()

	runDir, err := PrepareRunDir(fs, ctx, wf, kdepsDir, pkgPath, logging.NewTestLogger())
	if err != nil {
		t.Fatalf("PrepareRunDir error: %v", err)
	}

	// Expect dummy.txt extracted inside runDir
	if ok, _ := afero.Exists(fs, filepath.Join(runDir, "dummy.txt")); !ok {
		t.Fatalf("extracted file missing")
	}
}

// TestPackageProjectHappy creates minimal compiled project and ensures .kdeps created.
func TestPackageProjectHappy(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	wf := stubWf{}
	kdepsDir := "/kdeps"
	compiled := "/compiled"

	// build minimal structure
	_ = fs.MkdirAll(filepath.Join(compiled, "resources"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(compiled, "resources", "client.pkl"), []byte("run { }"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(compiled, "workflow.pkl"), []byte("amends \"package://schema.kdeps.com/core@0.0.1#/Workflow.pkl\"\n"), 0o644)
	_ = fs.MkdirAll(kdepsDir, 0o755)

	pkg, err := PackageProject(fs, ctx, wf, kdepsDir, compiled, logging.NewTestLogger())
	if err != nil {
		t.Fatalf("PackageProject error: %v", err)
	}

	if ok, _ := afero.Exists(fs, pkg); !ok {
		t.Fatalf("package file not written: %s", pkg)
	}

	// call again to ensure overwrite logic works (should not error)
	if _, err := PackageProject(fs, ctx, wf, kdepsDir, compiled, logging.NewTestLogger()); err != nil {
		t.Fatalf("second PackageProject error: %v", err)
	}
}

// stubWorkflow implements the required methods of pklWf.Workflow for this unit test.
type stubWorkflowPkg struct{}

func (stubWorkflowPkg) GetAgentID() string                   { return "mini-agent" }
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
	_ = afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte("Name='x'\nVersion='0.0.1'"), 0o644)

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

func TestFindWorkflowFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Setup mock directory structure
	baseDir := "/project"
	workflowDir := filepath.Join(baseDir, "sub")
	pklPath := filepath.Join(workflowDir, "workflow.pkl")

	if err := fs.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := afero.WriteFile(fs, pklPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write workflow file: %v", err)
	}

	// Positive case
	found, err := FindWorkflowFile(fs, baseDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != pklPath {
		t.Errorf("expected %s, got %s", pklPath, found)
	}

	// Negative case: directory without workflow.pkl
	emptyDir := "/empty"
	if err := fs.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatalf("failed to create empty dir: %v", err)
	}
	if _, err := FindWorkflowFile(fs, emptyDir, logger); err == nil {
		t.Errorf("expected error for missing workflow.pkl, got nil")
	}
}
