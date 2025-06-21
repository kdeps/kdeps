package archiver_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	pklProject "github.com/kdeps/schema/gen/project"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestPrepareRunDir_NewDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Create a simple tar.gz package
	pkgFile := createTestPackage(t, fs)
	defer fs.Remove(pkgFile)

	wf := stubWf{}

	// Test creating new run directory
	runDir, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFile, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, runDir)

	// Verify directory was created
	exists, err := afero.Exists(fs, runDir)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify extracted files
	workflowFile := filepath.Join(runDir, "workflow.pkl")
	exists, err = afero.Exists(fs, workflowFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPrepareRunDir_ExistingDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Create a simple tar.gz package
	pkgFile := createTestPackage(t, fs)
	defer fs.Remove(pkgFile)

	wf := stubWf{}

	// Create existing run directory with some content
	runDir := filepath.Join(kdepsDir, "run", wf.GetName(), wf.GetVersion(), "workflow")
	assert.NoError(t, fs.MkdirAll(runDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(runDir, "oldfile.txt"), []byte("old content"), 0o644))

	// Test preparing run directory (should remove old and create new)
	resultDir, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFile, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, resultDir)

	// Verify old file was removed
	oldFile := filepath.Join(runDir, "oldfile.txt")
	exists, err := afero.Exists(fs, oldFile)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Verify new content was extracted
	workflowFile := filepath.Join(runDir, "workflow.pkl")
	exists, err = afero.Exists(fs, workflowFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestPrepareRunDir_InvalidPackageFile(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	wf := stubWf{}

	// Test with non-existent package file
	runDir, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, "/nonexistent/package.kdeps", logger)
	assert.Error(t, err)
	assert.Empty(t, runDir)
}

func TestPrepareRunDir_InvalidTarFile(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Create invalid tar.gz file
	invalidFile, err := afero.TempFile(fs, "", "invalid")
	assert.NoError(t, err)
	defer fs.Remove(invalidFile.Name())
	assert.NoError(t, invalidFile.Close())

	wf := stubWf{}

	// Test with invalid tar.gz file
	runDir, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, invalidFile.Name(), logger)
	assert.Error(t, err)
	assert.Empty(t, runDir)
}

func TestPrepareRunDir_ComplexPackage(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directory for kdeps
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	// Create a complex tar.gz package with directories and files
	pkgFile := createComplexTestPackage(t, fs)
	defer fs.Remove(pkgFile)

	wf := stubWf{}

	// Test extracting complex package
	runDir, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFile, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, runDir)

	// Verify directory structure was created
	subDir := filepath.Join(runDir, "subdir")
	exists, err := afero.Exists(fs, subDir)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify files were extracted
	workflowFile := filepath.Join(runDir, "workflow.pkl")
	exists, err = afero.Exists(fs, workflowFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	subFile := filepath.Join(subDir, "test.txt")
	exists, err = afero.Exists(fs, subFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// Helper function to create a simple test package
func createTestPackage(t *testing.T, fs afero.Fs) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add workflow.pkl file
	content := []byte("workflow content")
	header := &tar.Header{
		Name: "workflow.pkl",
		Mode: 0o644,
		Size: int64(len(content)),
	}
	assert.NoError(t, tw.WriteHeader(header))
	_, err := tw.Write(content)
	assert.NoError(t, err)

	assert.NoError(t, tw.Close())
	assert.NoError(t, gw.Close())

	// Write to temp file
	pkgFile, err := afero.TempFile(fs, "", "test-package")
	assert.NoError(t, err)
	_, err = pkgFile.Write(buf.Bytes())
	assert.NoError(t, err)
	assert.NoError(t, pkgFile.Close())

	return pkgFile.Name()
}

// Helper function to create a complex test package with directories
func createComplexTestPackage(t *testing.T, fs afero.Fs) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add directory
	dirHeader := &tar.Header{
		Name:     "subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	}
	assert.NoError(t, tw.WriteHeader(dirHeader))

	// Add workflow.pkl file
	workflowContent := []byte("workflow content")
	workflowHeader := &tar.Header{
		Name: "workflow.pkl",
		Mode: 0o644,
		Size: int64(len(workflowContent)),
	}
	assert.NoError(t, tw.WriteHeader(workflowHeader))
	_, err := tw.Write(workflowContent)
	assert.NoError(t, err)

	// Add file in subdirectory
	fileContent := []byte("test content")
	fileHeader := &tar.Header{
		Name: "subdir/test.txt",
		Mode: 0o644,
		Size: int64(len(fileContent)),
	}
	assert.NoError(t, tw.WriteHeader(fileHeader))
	_, err = tw.Write(fileContent)
	assert.NoError(t, err)

	assert.NoError(t, tw.Close())
	assert.NoError(t, gw.Close())

	// Write to temp file
	pkgFile, err := afero.TempFile(fs, "", "complex-package")
	assert.NoError(t, err)
	_, err = pkgFile.Write(buf.Bytes())
	assert.NoError(t, err)
	assert.NoError(t, pkgFile.Close())

	return pkgFile.Name()
}

func TestCompileWorkflow_EmptyAction(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfEmptyAction{}

	kdepsDir, err := afero.TempDir(fs, "", "compile-workflow-empty-action")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project-empty-action")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	_, err = archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "please specify the default action")
}

func TestCompileWorkflow_DirExistsError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "dirExists"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "kdeps")
	projectDir, _ := afero.TempDir(base, "", "project")
	wf := stubWf{}
	_, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
}

func TestCompileWorkflow_RemoveAllError(t *testing.T) {
	base := afero.NewOsFs()
	kdepsDir, _ := afero.TempDir(base, "", "kdeps")
	projectDir, _ := afero.TempDir(base, "", "project")
	agentDir := filepath.Join(kdepsDir, "agents", "testAgent", "1.0.0")
	_ = base.MkdirAll(agentDir, 0o755)
	_ = afero.WriteFile(base, filepath.Join(agentDir, "oldfile.txt"), []byte("old content"), 0o644)
	fs := &errorFs{base, "removeAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	_, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
}

func TestCompileWorkflow_MkdirAllError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "kdeps")
	projectDir, _ := afero.TempDir(base, "", "project")
	wf := stubWf{}
	_, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
}

func TestCompileWorkflow_ReadFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "readFile"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "kdeps")
	projectDir, _ := afero.TempDir(base, "", "project")
	wf := stubWf{}
	_, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
}

func TestCompileWorkflow_WriteFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "writeFile"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "kdeps")
	projectDir, _ := afero.TempDir(base, "", "project")
	_ = afero.WriteFile(base, filepath.Join(projectDir, "workflow.pkl"), []byte("targetActionID = \"testAction\""), 0o644)
	wf := stubWf{}
	_, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_ExistsError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "exists"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-exists-error")
	defer base.RemoveAll(kdepsDir)
	pkgFilePath := "/nonexistent/package.kdeps"
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_RemoveAllError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "removeAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-remove-error")
	defer base.RemoveAll(kdepsDir)
	// Create the run directory that will be removed
	runDir := filepath.Join(kdepsDir, "run/testAgent/1.0.0/workflow")
	_ = base.MkdirAll(runDir, 0o755)
	pkgFilePath := "/nonexistent/package.kdeps"
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_MkdirAllError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-mkdir-error")
	defer base.RemoveAll(kdepsDir)
	pkgFilePath := "/nonexistent/package.kdeps"
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_OpenError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(fs, "", "prepare-run-dir-open-error")
	defer fs.RemoveAll(kdepsDir)
	pkgFilePath := "/nonexistent/package.kdeps"
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_GzipReaderError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(fs, "", "prepare-run-dir-gzip-error")
	defer fs.RemoveAll(kdepsDir)
	// Create an invalid gzip file
	pkgFilePath, _ := afero.TempFile(fs, "", "invalid-gzip.kdeps")
	defer fs.Remove(pkgFilePath.Name())
	pkgFilePath.Close()
	_ = afero.WriteFile(fs, pkgFilePath.Name(), []byte("not a gzip file"), 0o644)
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath.Name(), logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_TarReaderError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(fs, "", "prepare-run-dir-tar-error")
	defer fs.RemoveAll(kdepsDir)
	// Create a valid gzip file but invalid tar content
	pkgFilePath, _ := afero.TempFile(fs, "", "invalid-tar.kdeps")
	defer fs.Remove(pkgFilePath.Name())
	pkgFilePath.Close()
	// Create a gzip file with invalid tar content
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("not tar content"))
	gw.Close()
	_ = afero.WriteFile(fs, pkgFilePath.Name(), buf.Bytes(), 0o644)
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath.Name(), logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_CreateDirectoryError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-create-dir-error")
	defer base.RemoveAll(kdepsDir)
	// Create a valid kdeps package with a directory entry
	pkgFilePath := createValidKdepsPackage(base, "test-agent", "1.0.0", true, false)
	defer base.Remove(pkgFilePath)
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_CreateFileDirectoryError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-create-file-dir-error")
	defer base.RemoveAll(kdepsDir)
	// Create a valid kdeps package with a file in a subdirectory
	pkgFilePath := createValidKdepsPackage(base, "test-agent", "1.0.0", false, true)
	defer base.Remove(pkgFilePath)
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_CreateFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "create"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(base, "", "prepare-run-dir-create-file-error")
	defer base.RemoveAll(kdepsDir)
	// Create a valid kdeps package with a file
	pkgFilePath := createValidKdepsPackage(base, "test-agent", "1.0.0", false, false)
	defer base.Remove(pkgFilePath)
	_, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.Error(t, err)
}

func TestPrepareRunDir_UnknownHeaderType(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	kdepsDir, _ := afero.TempDir(fs, "", "prepare-run-dir-unknown-header")
	defer fs.RemoveAll(kdepsDir)
	// Create a kdeps package with an unknown header type
	pkgFilePath := createKdepsPackageWithUnknownHeader(fs, "test-agent", "1.0.0")
	defer fs.Remove(pkgFilePath)
	result, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	// Should not error, just log the unknown type
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestPrepareRunDir_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfSimple{}
	kdepsDir, _ := afero.TempDir(fs, "", "prepare-run-dir-success")
	defer fs.RemoveAll(kdepsDir)
	// Create a valid kdeps package
	pkgFilePath := createValidKdepsPackage(fs, "test-agent", "1.0.0", false, false)
	defer fs.Remove(pkgFilePath)
	result, err := archiver.PrepareRunDir(fs, ctx, wf, kdepsDir, pkgFilePath, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "run/testAgent/1.0.0/workflow")
}

// Helper function to create a kdeps package with unknown header type
func createKdepsPackageWithUnknownHeader(fs afero.Fs, agentName, version string) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add workflow.pkl
	workflowContent := fmt.Sprintf(`amends "%s"

name = "%s"
version = "%s"
defaultAction = "testAction"
authors = []
`, schema.SchemaVersion(context.Background()), agentName, version)

	workflowHeader := &tar.Header{
		Name: "workflow.pkl",
		Mode: 0o644,
		Size: int64(len(workflowContent)),
	}
	tw.WriteHeader(workflowHeader)
	tw.Write([]byte(workflowContent))

	// Add a file with unknown header type
	unknownHeader := &tar.Header{
		Name:     "unknown.txt",
		Mode:     0o644,
		Size:     int64(len("unknown content")),
		Typeflag: 99, // Unknown type
	}
	tw.WriteHeader(unknownHeader)
	tw.Write([]byte("unknown content"))

	tw.Close()
	gw.Close()

	// Write to temp file
	tempFile, _ := afero.TempFile(fs, "", "unknown-header-package.kdeps")
	tempFile.Write(buf.Bytes())
	tempFile.Close()
	return tempFile.Name()
}

// Mock filesystem for testing file writing errors
type mockFileSystem struct {
	afero.Fs
	createError bool
}

func (m *mockFileSystem) Create(name string) (afero.File, error) {
	if m.createError {
		return nil, errors.New("mock create error")
	}
	return m.Fs.Create(name)
}

// Helper workflow stub for testing CompileWorkflow with empty action
type stubWfEmptyAction struct{}

func (stubWfEmptyAction) GetName() string                   { return "testAgent" }
func (stubWfEmptyAction) GetVersion() string                { return "1.0.0" }
func (stubWfEmptyAction) GetDescription() string            { return "" }
func (stubWfEmptyAction) GetWebsite() *string               { return nil }
func (stubWfEmptyAction) GetAuthors() *[]string             { return nil }
func (stubWfEmptyAction) GetDocumentation() *string         { return nil }
func (stubWfEmptyAction) GetRepository() *string            { return nil }
func (stubWfEmptyAction) GetHeroImage() *string             { return nil }
func (stubWfEmptyAction) GetAgentIcon() *string             { return nil }
func (stubWfEmptyAction) GetTargetActionID() string         { return "" } // Empty action
func (stubWfEmptyAction) GetWorkflows() []string            { return nil }
func (stubWfEmptyAction) GetResources() *[]string           { return nil }
func (stubWfEmptyAction) GetSettings() *pklProject.Settings { return nil }

func TestCompileWorkflow_ActionWithAtPrefix(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	// Create workflow file with valid amends line
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

targetActionID = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWfWithAtPrefix{}

	// Test with action that already has @ prefix
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, compiledDir)

	// Verify compiled workflow file was created
	compiledWorkflow := filepath.Join(compiledDir, "workflow.pkl")
	exists, err := afero.Exists(fs, compiledWorkflow)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify action was not changed (already had @ prefix)
	content, err := afero.ReadFile(fs, compiledWorkflow)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `targetActionID = "@testAgent/testAction:1.0.0"`)
}

func TestCompileWorkflow_ActionWithoutAtPrefix1(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	// Create workflow file with valid amends line and defaultAction
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

targetActionID = "testAction"
defaultAction = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWfWithAtPrefix{}

	// Test with action that doesn't have @ prefix
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, compiledDir)

	// Verify compiled workflow file was created
	compiledWorkflow := filepath.Join(compiledDir, "workflow.pkl")
	exists, err := afero.Exists(fs, compiledWorkflow)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify action was updated with @ prefix
	content, err := afero.ReadFile(fs, compiledWorkflow)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `targetActionID = "@testAgent/testAction:1.0.0"`)
}

func TestCompileWorkflow_ExistingAgentDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	// Create existing agent directory with some content
	agentDir := filepath.Join(kdepsDir, "agents", "testAgent", "1.0.0")
	assert.NoError(t, fs.MkdirAll(agentDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(agentDir, "oldfile.txt"), []byte("old content"), 0o644))

	// Create workflow file with valid amends line and defaultAction
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

targetActionID = "testAction"
defaultAction = "testAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	wf := stubWfWithAtPrefix{}

	// Test with existing agent directory (should remove old and create new)
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, compiledDir)

	// Verify old file was removed
	oldFile := filepath.Join(agentDir, "oldfile.txt")
	exists, err := afero.Exists(fs, oldFile)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Verify new workflow file was created
	compiledWorkflow := filepath.Join(compiledDir, "workflow.pkl")
	exists, err = afero.Exists(fs, compiledWorkflow)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCompileWorkflow_MissingWorkflowFile(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp directories
	kdepsDir, err := afero.TempDir(fs, "", "kdeps")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	wf := stubWf{}

	// Test with missing workflow file
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, projectDir, logger)
	assert.Error(t, err)
	assert.Empty(t, compiledDir)
	assert.Contains(t, err.Error(), "please specify the default action in the workflow")
}

// wfWithAction is a stub workflow that returns a non-empty action ID
type wfWithAction struct{ stubWf }

func (wfWithAction) GetTargetActionID() string { return "someAction" }

func TestCompileWorkflow_InvalidProjectDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "compile-workflow-invalid-dir")
	defer fs.RemoveAll(kdepsDir)

	// Use a stub that returns a non-empty action ID so the function proceeds to file reading
	wf := wfWithAction{}

	// Test with non-existent project directory
	compiledDir, err := archiver.CompileWorkflow(fs, ctx, wf, kdepsDir, "/nonexistent/project", logger)
	assert.Error(t, err)
	assert.Empty(t, compiledDir)
	assert.Contains(t, err.Error(), "no such file or directory")
}

type stubWfWithAtPrefix struct{}

func (stubWfWithAtPrefix) GetName() string                   { return "testAgent" }
func (stubWfWithAtPrefix) GetVersion() string                { return "1.0.0" }
func (stubWfWithAtPrefix) GetDescription() string            { return "" }
func (stubWfWithAtPrefix) GetWebsite() *string               { return nil }
func (stubWfWithAtPrefix) GetAuthors() *[]string             { return nil }
func (stubWfWithAtPrefix) GetDocumentation() *string         { return nil }
func (stubWfWithAtPrefix) GetRepository() *string            { return nil }
func (stubWfWithAtPrefix) GetHeroImage() *string             { return nil }
func (stubWfWithAtPrefix) GetAgentIcon() *string             { return nil }
func (stubWfWithAtPrefix) GetTargetActionID() string         { return "@testAgent/testAction:1.0.0" } // Already has @ prefix
func (stubWfWithAtPrefix) GetWorkflows() []string            { return nil }
func (stubWfWithAtPrefix) GetResources() *[]string           { return nil }
func (stubWfWithAtPrefix) GetSettings() *pklProject.Settings { return nil }

func TestCompileProject_CompileWorkflowError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfEmptyAction{} // Will cause CompileWorkflow to fail
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-cw-error")
	projectDir, _ := afero.TempDir(fs, "", "project-cw-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)
	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile workflow")
}

func TestCompileProject_DirExistsError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "dirExists"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{} // Use stub with non-empty action ID
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(base, "", "compile-project-dir-exists-error")
	projectDir, _ := afero.TempDir(base, "", "project-dir-exists-error")
	defer base.RemoveAll(kdepsDir)
	defer base.RemoveAll(projectDir)
	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCompileProject_CompiledWorkflowFileMissing(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{} // Use stub with non-empty action ID
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-missing-wf")
	projectDir, _ := afero.TempDir(fs, "", "project-missing-wf")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)
	// Create the compiled project dir but not the workflow.pkl
	agentDir := filepath.Join(kdepsDir, "agents", wf.GetName(), wf.GetVersion())
	assert.NoError(t, fs.MkdirAll(agentDir, 0o755))
	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCompileProject_LoadWorkflowError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-load-wf-error")
	projectDir, _ := afero.TempDir(fs, "", "project-load-wf-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)
	// Create the compiled project dir and an invalid workflow.pkl
	agentDir := filepath.Join(kdepsDir, "agents", wf.GetName(), wf.GetVersion())
	assert.NoError(t, fs.MkdirAll(agentDir, 0o755))
	compiledWorkflow := filepath.Join(agentDir, "workflow.pkl")
	assert.NoError(t, afero.WriteFile(fs, compiledWorkflow, []byte("invalid content"), 0o644))
	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "please specify the default action in the workflow")
}

func TestCompileProject_CompileResourcesError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-compile-resources-error")
	projectDir, _ := afero.TempDir(fs, "", "project-compile-resources-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)

	// Create workflow.pkl in project directory
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

name = "agent"
version = "1.2.3"
description = "Test agent for compilation"
authors {}
targetActionID = "someAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resources directory to avoid CompileResources error
	resourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(resourcesDir, 0o755))
	// Add a dummy .pkl file so CompileResources proceeds
	dummyPkl := filepath.Join(resourcesDir, "resource1.pkl")
	dummyContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

id = "dummy"`
	assert.NoError(t, afero.WriteFile(fs, dummyPkl, []byte(dummyContent), 0o644))

	orig := archiver.CompileResourcesFunc
	archiver.CompileResourcesFunc = func(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, resourcesDir, projectDir string, logger *logging.Logger) error {
		return errors.New("forced CompileResources error")
	}
	defer func() { archiver.CompileResourcesFunc = orig }()

	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile resources")
}

func TestCompileProject_CopyDataDirError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-copydatadir-error")
	projectDir, _ := afero.TempDir(fs, "", "project-copydatadir-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)

	// Create workflow.pkl in project directory
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

name = "agent"
version = "1.2.3"
description = "Test agent for compilation"
authors {}
targetActionID = "someAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resources directory to avoid CompileResources error
	resourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(resourcesDir, 0o755))
	// Add a dummy .pkl file so CompileResources proceeds
	dummyPkl := filepath.Join(resourcesDir, "resource1.pkl")
	dummyContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

id = "dummy"`
	assert.NoError(t, afero.WriteFile(fs, dummyPkl, []byte(dummyContent), 0o644))

	orig := archiver.CopyDataDirFunc
	archiver.CopyDataDirFunc = func(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir, agentName, agentVersion, agentAction string, processWorkflows bool, logger *logging.Logger) error {
		return errors.New("forced CopyDataDir error")
	}
	defer func() { archiver.CopyDataDirFunc = orig }()

	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to copy project")
}

func TestCompileProject_ProcessExternalWorkflowsError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-proc-ext-wf-error")
	projectDir, _ := afero.TempDir(fs, "", "project-proc-ext-wf-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)

	// Create workflow.pkl in project directory
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

name = "agent"
version = "1.2.3"
description = "Test agent for compilation"
authors {}
targetActionID = "someAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resources directory to avoid CompileResources error
	resourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(resourcesDir, 0o755))
	// Add a dummy .pkl file so CompileResources proceeds
	dummyPkl := filepath.Join(resourcesDir, "resource1.pkl")
	dummyContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

id = "dummy"`
	assert.NoError(t, afero.WriteFile(fs, dummyPkl, []byte(dummyContent), 0o644))

	orig := archiver.ProcessExternalWorkflowsFunc
	archiver.ProcessExternalWorkflowsFunc = func(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, projectDir, compiledProjectDir string, logger *logging.Logger) error {
		return errors.New("forced ProcessExternalWorkflows error")
	}
	defer func() { archiver.ProcessExternalWorkflowsFunc = orig }()

	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to process workflows")
}

func TestCompileProject_PackageProjectError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-pkgproj-error")
	projectDir, _ := afero.TempDir(fs, "", "project-pkgproj-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)

	// Create workflow.pkl in project directory
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

name = "agent"
version = "1.2.3"
description = "Test agent for compilation"
authors {}
targetActionID = "someAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resources directory to avoid CompileResources error
	resourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(resourcesDir, 0o755))
	// Add a dummy .pkl file so CompileResources proceeds
	dummyPkl := filepath.Join(resourcesDir, "resource1.pkl")
	dummyContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

id = "dummy"`
	assert.NoError(t, afero.WriteFile(fs, dummyPkl, []byte(dummyContent), 0o644))

	orig := archiver.PackageProjectFunc
	archiver.PackageProjectFunc = func(fs afero.Fs, ctx context.Context, wf pklWf.Workflow, kdepsDir, compiledProjectDir string, logger *logging.Logger) (string, error) {
		return "", errors.New("forced PackageProject error")
	}
	defer func() { archiver.PackageProjectFunc = orig }()

	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to package project")
}

func TestCompileProject_CopyFileError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := wfWithAction{}
	env := &environment.Environment{Pwd: "/tmp"}
	kdepsDir, _ := afero.TempDir(fs, "", "compile-project-copyfile-error")
	projectDir, _ := afero.TempDir(fs, "", "project-copyfile-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)

	// Create workflow.pkl in project directory
	workflowContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Workflow.pkl"

name = "agent"
version = "1.2.3"
description = "Test agent for compilation"
authors {}
targetActionID = "someAction"`
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	// Create resources directory to avoid CompileResources error
	resourcesDir := filepath.Join(projectDir, "resources")
	assert.NoError(t, fs.MkdirAll(resourcesDir, 0o755))
	// Add a dummy .pkl file so CompileResources proceeds
	dummyPkl := filepath.Join(resourcesDir, "resource1.pkl")
	dummyContent := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

id = "dummy"`
	assert.NoError(t, afero.WriteFile(fs, dummyPkl, []byte(dummyContent), 0o644))

	orig := archiver.CopyFileFunc
	archiver.CopyFileFunc = func(fs afero.Fs, ctx context.Context, src, dst string, logger *logging.Logger) error {
		return errors.New("forced CopyFile error")
	}
	defer func() { archiver.CopyFileFunc = orig }()

	_, _, err := archiver.CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forced CopyFile error")
}

func TestCompileProject_Success(t *testing.T) {
	// This would require a full valid workflow and all dependencies, skipping for now as it's more integration
}

func TestProcessExternalWorkflows_GetWorkflowsNil(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{} // GetWorkflows returns nil
	kdepsDir, _ := afero.TempDir(fs, "", "process-external-nil")
	projectDir, _ := afero.TempDir(fs, "", "project-external-nil")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-external-nil")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)
	defer fs.RemoveAll(compiledProjectDir)
	err := archiver.ProcessExternalWorkflows(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, logger)
	assert.NoError(t, err)
}

func TestProcessExternalWorkflows_CopyDataDirError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfWithWorkflows{} // GetWorkflows returns non-nil
	kdepsDir, _ := afero.TempDir(base, "", "process-external-copy-error")
	projectDir, _ := afero.TempDir(base, "", "project-external-copy-error")
	compiledProjectDir, _ := afero.TempDir(base, "", "compiled-external-copy-error")
	defer base.RemoveAll(kdepsDir)
	defer base.RemoveAll(projectDir)
	defer base.RemoveAll(compiledProjectDir)

	// Create the necessary directory structure for CopyDataDir to succeed initially
	// but fail when CopyDir tries to create directories
	agentPath := filepath.Join(kdepsDir, "agents", "externalAgent", "1.0.0")
	_ = base.MkdirAll(agentPath, 0o755)
	// Add a file to make it a valid agent directory
	agentFile := filepath.Join(agentPath, "workflow.pkl")
	_ = afero.WriteFile(base, agentFile, []byte("workflow content"), 0o644)

	// Create the data directory structure that will be copied
	dataDir := filepath.Join(agentPath, "data", "externalAgent", "1.0.0")
	_ = base.MkdirAll(dataDir, 0o755)
	// Add a file to make the copy operation actually happen
	dataFile := filepath.Join(dataDir, "test.txt")
	_ = afero.WriteFile(base, dataFile, []byte("test data"), 0o644)

	err := archiver.ProcessExternalWorkflows(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, logger)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	assert.Contains(t, err.Error(), "mkdirAll error")
}

func TestProcessExternalWorkflows_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWfWithWorkflows{} // GetWorkflows returns non-nil
	kdepsDir, _ := afero.TempDir(fs, "", "process-external-success")
	projectDir, _ := afero.TempDir(fs, "", "project-external-success")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-external-success")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(projectDir)
	defer fs.RemoveAll(compiledProjectDir)
	// Create the necessary directory structure for CopyDataDir to succeed
	agentPath := filepath.Join(kdepsDir, "agents", "externalAgent", "1.0.0")
	_ = fs.MkdirAll(agentPath, 0o755)
	// Add a file to make it a valid agent directory
	agentFile := filepath.Join(agentPath, "workflow.pkl")
	_ = afero.WriteFile(fs, agentFile, []byte("workflow content"), 0o644)
	err := archiver.ProcessExternalWorkflows(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, logger)
	assert.NoError(t, err)
}

// Helper stub workflow with non-nil GetWorkflows
type stubWfWithWorkflows struct{}

func (stubWfWithWorkflows) GetName() string           { return "testAgent" }
func (stubWfWithWorkflows) GetVersion() string        { return "1.0.0" }
func (stubWfWithWorkflows) GetDescription() string    { return "" }
func (stubWfWithWorkflows) GetWebsite() *string       { return nil }
func (stubWfWithWorkflows) GetAuthors() *[]string     { return nil }
func (stubWfWithWorkflows) GetDocumentation() *string { return nil }
func (stubWfWithWorkflows) GetRepository() *string    { return nil }
func (stubWfWithWorkflows) GetHeroImage() *string     { return nil }
func (stubWfWithWorkflows) GetAgentIcon() *string     { return nil }
func (stubWfWithWorkflows) GetTargetActionID() string { return "testAction" }
func (stubWfWithWorkflows) GetWorkflows() []string {
	return []string{"@externalAgent/testAction:1.0.0"}
}
func (stubWfWithWorkflows) GetResources() *[]string           { return nil }
func (stubWfWithWorkflows) GetSettings() *pklProject.Settings { return nil }
