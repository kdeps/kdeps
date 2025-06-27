package archiver_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	pklProj "github.com/kdeps/schema/gen/project"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

func TestExtractPackage_Minimal(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	tmpDir, err := afero.TempDir(fs, "", "extractpkg")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	defer fs.RemoveAll(tmpDir)

	// Create a minimal tar.gz .kdeps file with workflow.pkl
	kdepsFile := filepath.Join(tmpDir, "test.kdeps")
	f, err := os.Create(kdepsFile)
	if err != nil {
		t.Fatalf("os.Create: %v", err)
	}
	gz := gzip.NewWriter(f)
	tarw := tar.NewWriter(gz)

	// Add workflow.pkl file with valid PKL content
	workflowContent := []byte(`amends "package://schema.kdeps.com/core@0.2.30#/Workflow.pkl"

name = "testAgent"
description = "Test Agent"
version = "1.0.0"
targetActionID = "hello"`)
	hdr := &tar.Header{Name: "workflow.pkl", Mode: 0o644, Size: int64(len(workflowContent))}
	if err := tarw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tarw.Write(workflowContent); err != nil {
		t.Fatalf("tar write: %v", err)
	}

	// Add a dummy file
	dummyContent := []byte("hello kdeps")
	hdr2 := &tar.Header{Name: "dummy.txt", Mode: 0o644, Size: int64(len(dummyContent))}
	if err := tarw.WriteHeader(hdr2); err != nil {
		t.Fatalf("tar header2: %v", err)
	}
	if _, err := tarw.Write(dummyContent); err != nil {
		t.Fatalf("tar write2: %v", err)
	}

	tarw.Close()
	gz.Close()
	f.Close()

	kdepsDir := tmpDir
	pkg, err := ExtractPackage(fs, ctx, kdepsDir, kdepsFile, logger)
	if err != nil {
		t.Fatalf("ExtractPackage: %v", err)
	}
	assert.NotNil(t, pkg)
	assert.Equal(t, filepath.Join(kdepsDir, "packages", "test.kdeps"), pkg.PkgFilePath)
	assert.NotEmpty(t, pkg.Workflow)
}

func TestPackageProject_NewPackage(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create project structure
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "data", "testAgent", "1.0.0"), 0o755))

	// Create workflow file
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resource file
	resourceContent := `id: "testResource"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "resources", "test.pkl"), []byte(resourceContent), 0o644))

	// Create data file
	dataContent := `test data`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "data", "testAgent", "1.0.0", "data.txt"), []byte(dataContent), 0o644))

	wf := stubWf{}

	// Test packaging project
	packagePath, err := PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, packagePath)

	// Verify package file was created
	exists, err := afero.Exists(fs, packagePath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify package directory was created
	packageDir := filepath.Join(kdepsDir, "packages")
	exists, err = afero.Exists(fs, packageDir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPackageProject_ExistingPackage(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create project structure
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// Create existing package file
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, fs.MkdirAll(packageDir, 0o755))
	existingPackage := filepath.Join(packageDir, fmt.Sprintf("%s-%s.kdeps", wf.GetName(), wf.GetVersion()))
	assert.NoError(t, afero.WriteFile(fs, existingPackage, []byte("old package"), 0o644))

	// Test packaging project (should remove old and create new)
	packagePath, err := PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, packagePath)

	// Verify new package was created
	exists, err := afero.Exists(fs, packagePath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify old package was replaced (content should be different)
	content, err := afero.ReadFile(fs, packagePath)
	assert.NoError(t, err)
	assert.NotEqual(t, []byte("old package"), content)
}

func TestPackageProject_InvalidProjectDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	wf := stubWf{}

	// Test with non-existent project directory
	packagePath, err := PackageProject(fs, ctx, wf, kdepsDir, "/nonexistent/project", logger)
	assert.Error(t, err)
	assert.Empty(t, packagePath)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestPackageProject_EmptyProject(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	wf := stubWf{}

	// Test packaging empty project (should still create package)
	packagePath, err := PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, packagePath)

	// Verify package file was created (even if empty)
	exists, err := afero.Exists(fs, packagePath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPackageProject_ComplexProject(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create complex project structure
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "data", "agent1", "1.0.0"), 0o755))
	assert.NoError(t, fs.MkdirAll(filepath.Join(compiledProjectDir, "data", "agent2", "2.0.0"), 0o755))

	// Create workflow file
	workflowContent := `targetActionID = "complexAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create multiple resource files
	resource1Content := `id: "resource1"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "resources", "resource1.pkl"), []byte(resource1Content), 0o644))

	resource2Content := `id: "resource2"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "resources", "resource2.pkl"), []byte(resource2Content), 0o644))

	// Create data files for multiple agents
	data1Content := `agent1 data`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "data", "agent1", "1.0.0", "data1.txt"), []byte(data1Content), 0o644))

	data2Content := `agent2 data`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "data", "agent2", "2.0.0", "data2.txt"), []byte(data2Content), 0o644))

	wf := stubWf{}

	// Test packaging complex project
	packagePath, err := PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, packagePath)

	// Verify package file was created
	exists, err := afero.Exists(fs, packagePath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify package has content (not empty)
	info, err := fs.Stat(packagePath)
	assert.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestFindWorkflowFile_StatError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Test with non-existent folder
	_, err := FindWorkflowFile(fs, "/nonexistent/folder", logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error accessing folder")
}

func TestFindWorkflowFile_NotDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp file
	tempFile, err := afero.TempFile(fs, "", "not-a-directory")
	assert.NoError(t, err)
	defer fs.Remove(tempFile.Name())
	tempFile.Close()

	// Test with a file path instead of directory
	_, err = FindWorkflowFile(fs, tempFile.Name(), logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the path provided is not a directory")
}

func TestFindWorkflowFile_WalkError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "walk"}
	logger := logging.NewTestLogger()

	// Create a temp directory
	tempDir, err := afero.TempDir(base, "", "walk-error-test")
	assert.NoError(t, err)
	defer base.RemoveAll(tempDir)

	// Trigger the walk by calling the function – the mocked Walk will return an error
	_, err = FindWorkflowFile(fs, tempDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.pkl not found")
}

func TestFindWorkflowFile_FileNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp directory without workflow.pkl
	tempDir, err := afero.TempDir(fs, "", "file-not-found-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	// Create some other files but not workflow.pkl
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "other.pkl"), []byte("content"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "not-workflow.txt"), []byte("content"), 0o644))

	_, err = FindWorkflowFile(fs, tempDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.pkl not found in folder")
}

func TestFindWorkflowFile_SuccessInRoot(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp directory
	tempDir, err := afero.TempDir(fs, "", "success-root-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	// Create workflow.pkl in the root directory
	workflowPath := filepath.Join(tempDir, "workflow.pkl")
	assert.NoError(t, afero.WriteFile(fs, workflowPath, []byte("workflow content"), 0o644))

	// Create some other files
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "other.pkl"), []byte("content"), 0o644))

	result, err := FindWorkflowFile(fs, tempDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, workflowPath, result)
}

func TestFindWorkflowFile_SuccessInSubdirectory(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp directory
	tempDir, err := afero.TempDir(fs, "", "success-subdir-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2", "nested")
	assert.NoError(t, fs.MkdirAll(subDir1, 0o755))
	assert.NoError(t, fs.MkdirAll(subDir2, 0o755))

	// Create workflow.pkl in a nested subdirectory
	workflowPath := filepath.Join(subDir2, "workflow.pkl")
	assert.NoError(t, afero.WriteFile(fs, workflowPath, []byte("workflow content"), 0o644))

	// Create some other files
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "other.pkl"), []byte("content"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(subDir1, "not-workflow.txt"), []byte("content"), 0o644))

	result, err := FindWorkflowFile(fs, tempDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, workflowPath, result)
}

func TestFindWorkflowFile_MultipleWorkflowFiles(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp directory
	tempDir, err := afero.TempDir(fs, "", "multiple-workflow-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	// Create multiple workflow.pkl files in different locations
	workflow1 := filepath.Join(tempDir, "workflow.pkl")
	workflow2 := filepath.Join(tempDir, "subdir", "workflow.pkl")
	assert.NoError(t, fs.MkdirAll(filepath.Join(tempDir, "subdir"), 0o755))

	assert.NoError(t, afero.WriteFile(fs, workflow1, []byte("workflow1 content"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, workflow2, []byte("workflow2 content"), 0o644))

	// Should find the first one (in root directory)
	result, err := FindWorkflowFile(fs, tempDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, workflow1, result)
}

func TestFindWorkflowFile_CaseSensitive(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create a temp directory
	tempDir, err := afero.TempDir(fs, "", "case-sensitive-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	// Create files with similar names but different cases
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "Workflow.pkl"), []byte("uppercase"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "workflow.PKL"), []byte("uppercase extension"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(tempDir, "WORKFLOW.PKL"), []byte("all uppercase"), 0o644))

	// Should not find any of these since we're looking for exact "workflow.pkl"
	_, err = FindWorkflowFile(fs, tempDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.pkl not found in folder")
}

func TestFindWorkflowFile_EmptyDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create an empty temp directory
	tempDir, err := afero.TempDir(fs, "", "empty-directory-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	_, err = FindWorkflowFile(fs, tempDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.pkl not found in folder")
}

func TestExtractPackage_TempDirCreationFails(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "tempDir"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "extract-tempdir-error")
	defer base.RemoveAll(kdepsDir)

	// Create a valid kdeps package file so the open call succeeds
	kdepsPackage := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(kdepsPackage)

	_, err := ExtractPackage(fs, ctx, kdepsDir, kdepsPackage, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load the workflow file")
}

func TestExtractPackage_GzipReaderError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "extract-gzip-error")
	defer fs.RemoveAll(kdepsDir)
	// Create an invalid gzip file
	kdepsPackage, _ := afero.TempFile(fs, "", "invalid-gzip.kdeps")
	defer fs.Remove(kdepsPackage.Name())
	kdepsPackage.Close()
	_ = afero.WriteFile(fs, kdepsPackage.Name(), []byte("not a gzip file"), 0o644)
	_, err := ExtractPackage(fs, ctx, kdepsDir, kdepsPackage.Name(), logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create gzip reader")
}

func TestExtractPackage_TarReaderError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "extract-tar-error")
	defer fs.RemoveAll(kdepsDir)
	// Create a valid gzip file but invalid tar content
	kdepsPackage, _ := afero.TempFile(fs, "", "invalid-tar.kdeps")
	defer fs.Remove(kdepsPackage.Name())
	kdepsPackage.Close()
	// Create a gzip file with invalid tar content
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("not tar content"))
	gw.Close()
	_ = afero.WriteFile(fs, kdepsPackage.Name(), buf.Bytes(), 0o644)
	_, err := ExtractPackage(fs, ctx, kdepsDir, kdepsPackage.Name(), logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read tar header")
}

// Helper functions for creating test kdeps packages
func createValidKdepsPackage(fs afero.Fs, agentName, version string, hasDir, hasSubdir bool) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add workflow.pkl
	workflowContent := fmt.Sprintf(`amends "%s"

name = "%s"
version = "%s"
defaultAction = "testAction"
`, schema.SchemaVersion(context.Background()), agentName, version)

	workflowHeader := &tar.Header{
		Name: "workflow.pkl",
		Mode: 0o644,
		Size: int64(len(workflowContent)),
	}
	tw.WriteHeader(workflowHeader)
	tw.Write([]byte(workflowContent))

	// Add a resource file
	resourceContent := fmt.Sprintf(`amends "%s"

id = "testAction"
`, schema.SchemaVersion(context.Background()))

	resourceHeader := &tar.Header{
		Name: "resources/test.pkl",
		Mode: 0o644,
		Size: int64(len(resourceContent)),
	}
	tw.WriteHeader(resourceHeader)
	tw.Write([]byte(resourceContent))

	// Add a data file
	dataContent := "test data"
	dataHeader := &tar.Header{
		Name: fmt.Sprintf("data/%s/%s/test.txt", agentName, version),
		Mode: 0o644,
		Size: int64(len(dataContent)),
	}
	tw.WriteHeader(dataHeader)
	tw.Write([]byte(dataContent))

	if hasDir {
		dirHeader := &tar.Header{
			Name:     "testdir/",
			Mode:     0o755,
			Typeflag: tar.TypeDir,
		}
		tw.WriteHeader(dirHeader)
	}

	if hasSubdir {
		subdirHeader := &tar.Header{
			Name: "subdir/test.txt",
			Mode: 0o644,
			Size: int64(len("subdir content")),
		}
		tw.WriteHeader(subdirHeader)
		tw.Write([]byte("subdir content"))
	}

	tw.Close()
	gw.Close()

	// Write to temp file
	tempFile, _ := afero.TempFile(fs, "", "test-package.kdeps")
	tempFile.Write(buf.Bytes())
	tempFile.Close()
	return tempFile.Name()
}

func createInvalidKdepsPackage(fs afero.Fs, agentName, version string) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add invalid workflow.pkl
	workflowContent := "invalid workflow content"
	workflowHeader := &tar.Header{
		Name: "workflow.pkl",
		Mode: 0o644,
		Size: int64(len(workflowContent)),
	}
	tw.WriteHeader(workflowHeader)
	tw.Write([]byte(workflowContent))

	tw.Close()
	gw.Close()

	// Write to temp file
	tempFile, _ := afero.TempFile(fs, "", "invalid-package.kdeps")
	tempFile.Write(buf.Bytes())
	tempFile.Close()
	return tempFile.Name()
}

// errorFs for simulating errors in afero.Fs methods for PackageProject tests
// errOn: "stat", "exists", "remove", "create", "walk", "tempDir", "mkdirAll", "chmod"
type errorFs struct {
	afero.Fs
	errOn string
}

func (e *errorFs) Stat(name string) (os.FileInfo, error) {
	if e.errOn == "stat" {
		return nil, fmt.Errorf("stat error")
	}
	return e.Fs.Stat(name)
}

func (e *errorFs) MkdirAll(path string, perm os.FileMode) error {
	if e.errOn == "mkdirAll" {
		return fmt.Errorf("mkdirAll error")
	}
	return e.Fs.MkdirAll(path, perm)
}

func (e *errorFs) Remove(name string) error {
	if e.errOn == "remove" {
		return fmt.Errorf("remove error")
	}
	return e.Fs.Remove(name)
}

func (e *errorFs) Create(name string) (afero.File, error) {
	if e.errOn == "create" {
		return nil, fmt.Errorf("create error")
	}
	return e.Fs.Create(name)
}

func (e *errorFs) Open(name string) (afero.File, error) {
	if e.errOn == "open" {
		return nil, fmt.Errorf("open error")
	}
	return e.Fs.Open(name)
}

func (e *errorFs) Walk(root string, walkFn filepath.WalkFunc) error {
	if e.errOn == "walk" {
		return fmt.Errorf("walk error")
	}
	return afero.Walk(e.Fs, root, walkFn)
}

func (e *errorFs) Exists(name string) (bool, error) {
	if e.errOn == "exists" {
		return false, fmt.Errorf("exists error")
	}
	return afero.Exists(e.Fs, name)
}

func (e *errorFs) TempDir(dir, prefix string) (string, error) {
	if e.errOn == "tempDir" {
		return "", fmt.Errorf("tempDir error")
	}
	return afero.TempDir(e.Fs, dir, prefix)
}

func (e *errorFs) Chmod(name string, mode os.FileMode) error {
	if e.errOn == "chmod" {
		return fmt.Errorf("chmod error")
	}
	return e.Fs.Chmod(name, mode)
}

// errorFsConditional for failing on specific file stat calls
type errorFsConditional struct {
	afero.Fs
	failOnPath string
}

func (e *errorFsConditional) Stat(name string) (os.FileInfo, error) {
	if name == e.failOnPath {
		return nil, fmt.Errorf("stat error on specific file")
	}
	return e.Fs.Stat(name)
}

// Additional PackageProject edge case tests to increase coverage

func TestPackageProject_PackageDirStatError(t *testing.T) {
	base := afero.NewOsFs()
	// When stat fails, it triggers mkdirAll, so we need to fail mkdirAll
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create minimal project structure
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// The Stat call fails, triggering MkdirAll which also fails
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error creating the system packages folder")
}

// TestPackageProject_PackageDirCreateError is covered by TestPackageProject_PackageDirStatError

func TestPackageProject_ExistsCheckError(t *testing.T) {
	base := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create project structure with packages directory already existing
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, base.MkdirAll(packageDir, 0o755))
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// Create a more sophisticated errorFs that fails on specific file stat calls
	fs := &errorFsConditional{base, filepath.Join(packageDir, fmt.Sprintf("%s-%s.kdeps", wf.GetName(), wf.GetVersion()))}

	// afero.Exists should fail when checking if existing package file exists
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error checking if package exists")
}

func TestPackageProject_RemoveExistingError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "remove"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create project structure and existing package file
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, base.MkdirAll(packageDir, 0o755))
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}
	existingPackage := filepath.Join(packageDir, fmt.Sprintf("%s-%s.kdeps", wf.GetName(), wf.GetVersion()))
	assert.NoError(t, afero.WriteFile(base, existingPackage, []byte("old package"), 0o644))

	// fs.Remove should fail when trying to remove existing package
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove existing package file")
}

func TestPackageProject_CreateFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "create"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create project structure
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, base.MkdirAll(packageDir, 0o755))
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// fs.Create should fail when creating the new package file
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create package file")
}

func TestPackageProject_FileOpenDuringWalkError(t *testing.T) {
	base := afero.NewOsFs()
	// The afero.Walk function opens files for reading during packaging
	fs := &errorFs{base, "open"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create project structure - but we won't be able to create the package file due to the Open error
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, base.MkdirAll(packageDir, 0o755))
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// This should fail when trying to open files during the packaging process
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open error")
}

// Additional tests for ExtractPackage error paths to increase coverage

func TestExtractPackage_OpenFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "open"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	// Create a package file
	kdepsPackage := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(kdepsPackage)

	// fs.Open should fail when opening the package file
	_, err = ExtractPackage(fs, ctx, kdepsDir, kdepsPackage, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open .kdeps file")
}

func TestExtractPackage_MkdirAllError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	// Create a package file
	kdepsPackage := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(kdepsPackage)

	// fs.MkdirAll should fail when creating temporary directory
	_, err = ExtractPackage(fs, ctx, kdepsDir, kdepsPackage, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temporary directory")
}

func TestExtractPackage_ChmodError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "chmod"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	// Create a package file
	kdepsPackage := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(kdepsPackage)

	// fs.Chmod should fail when setting file permissions after extraction
	_, err = ExtractPackage(fs, ctx, kdepsDir, kdepsPackage, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set file permissions")
}

// TestPackageProject_WalkRelPathError tests when filepath.Rel fails during walk
func TestPackageProject_WalkRelPathError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Use an invalid compiledProjectDir that doesn't exist
	// This will cause issues during Walk when trying to get relative paths
	compiledProjectDir := "/this/path/does/not/exist"

	wf := stubWf{}

	// Should fail during walk
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
}

// TestPackageProject_EnforcerError tests when enforcer.EnforceFolderStructure fails
func TestPackageProject_EnforcerError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create an improper project structure - resources as a file instead of directory
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "resources"), []byte("should be a directory"), 0o644))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWf{}

	// Should fail during enforcement
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
}

// errorFsWrite simulates write errors
type errorFsWrite struct {
	afero.Fs
	failOnWrite bool
}

type errorFile struct {
	afero.File
	failOnWrite bool
}

func (e *errorFsWrite) Create(name string) (afero.File, error) {
	f, err := e.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return &errorFile{File: f, failOnWrite: e.failOnWrite}, nil
}

func (e *errorFile) Write(p []byte) (n int, err error) {
	if e.failOnWrite {
		return 0, fmt.Errorf("write error")
	}
	return e.File.Write(p)
}

// TestPackageProject_WriteErrors tests write errors during tar creation
func TestPackageProject_WriteErrors(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFsWrite{Fs: base, failOnWrite: true}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(base, "", "compiled")
	assert.NoError(t, err)
	defer base.RemoveAll(compiledProjectDir)

	// Create project structure
	packageDir := filepath.Join(kdepsDir, "packages")
	assert.NoError(t, base.MkdirAll(packageDir, 0o755))
	assert.NoError(t, base.MkdirAll(filepath.Join(compiledProjectDir, "resources"), 0o755))
	workflowContent := `targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "workflow.pkl"), []byte(workflowContent), 0o644))
	assert.NoError(t, afero.WriteFile(base, filepath.Join(compiledProjectDir, "resources", "test.pkl"), []byte("content"), 0o644))

	wf := stubWf{}

	// Should fail when writing to tar
	_, err = PackageProject(fs, ctx, wf, kdepsDir, compiledProjectDir, logger)
	assert.Error(t, err)
}

// TestExtractPackage_CreateFileError tests file creation error during extraction
func TestExtractPackage_CreateFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "create"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, err := afero.TempDir(base, "", "kdeps")
	assert.NoError(t, err)
	defer base.RemoveAll(kdepsDir)

	// Create a valid package
	kdepsPackage := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(kdepsPackage)

	// Should fail when creating extracted files
	_, err = ExtractPackage(fs, ctx, kdepsDir, kdepsPackage, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create file")
}

// TestExtractPackage_SanitizePathError tests when SanitizeArchivePath fails
func TestExtractPackage_SanitizePathError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Create a package with path traversal attempts
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a file with path traversal
	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0o644,
		Size: 10,
	}
	tw.WriteHeader(header)
	tw.Write([]byte("malicious"))

	tw.Close()
	gw.Close()

	// Write to temp file
	tempFile, _ := afero.TempFile(fs, "", "malicious.kdeps")
	tempFile.Write(buf.Bytes())
	tempFile.Close()

	// Should fail due to path traversal
	_, err = ExtractPackage(fs, ctx, kdepsDir, tempFile.Name(), logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content filepath is tainted")

	fs.Remove(tempFile.Name())
}

// TestFindWorkflowFile_SkipDirBehavior tests that SkipDir is handled correctly
func TestFindWorkflowFile_SkipDirBehavior(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// Create temp directory with workflow.pkl in root
	tempDir, err := afero.TempDir(fs, "", "skipdir-test")
	assert.NoError(t, err)
	defer fs.RemoveAll(tempDir)

	workflowFile := filepath.Join(tempDir, "workflow.pkl")
	assert.NoError(t, afero.WriteFile(fs, workflowFile, []byte("root workflow"), 0o644))

	// Create a subdirectory with another workflow.pkl that should be skipped
	subDir := filepath.Join(tempDir, "subdir")
	assert.NoError(t, fs.MkdirAll(subDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(subDir, "workflow.pkl"), []byte("sub workflow"), 0o644))

	// Should find the root one and skip subdirectories
	result, err := FindWorkflowFile(fs, tempDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, workflowFile, result)
}
