package resolver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	assets "github.com/kdeps/schema/assets"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestResolver creates a DependencyResolver with real filesystem for PKL operations.
// Note: This resolver uses afero.NewOsFs() because many tests in this file involve
// PKL operations that require real file paths on disk.
func newTestResolver() *DependencyResolver {
	tmpDir := filepath.Join(os.TempDir(), "kdeps_test_", uuid.NewString())
	// We purposely ignore error for MkdirAll because temp dir should exist
	_ = os.MkdirAll(tmpDir, 0o755)

	// Use the real OS filesystem so that any spawned external tools (e.g. pkl)
	// can read the files. This still keeps everything inside a unique tmpdir.
	fs := afero.NewOsFs()

	actionDir := filepath.Join(tmpDir, "action")
	projectDir := filepath.Join(tmpDir, "project")
	workflowDir := filepath.Join(tmpDir, "workflow")

	_ = fs.MkdirAll(actionDir, 0o755)
	_ = fs.MkdirAll(projectDir, 0o755)
	_ = fs.MkdirAll(workflowDir, 0o755)

	return &DependencyResolver{
		Fs:          fs,
		Logger:      logging.NewTestLogger(),
		Context:     context.Background(),
		ActionDir:   actionDir,
		RequestID:   "test-request",
		ProjectDir:  projectDir,
		WorkflowDir: workflowDir,
	}
}

func TestPrepareImportFiles_CreatesFiles(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	dr := newTestResolver()
	assert.NoError(t, dr.PrepareImportFiles())

	// Expected files
	base := dr.ActionDir
	files := []string{
		filepath.Join(base, "llm/"+dr.RequestID+"__llm_output.pkl"),
		filepath.Join(base, "client/"+dr.RequestID+"__client_output.pkl"),
		filepath.Join(base, "exec/"+dr.RequestID+"__exec_output.pkl"),
		filepath.Join(base, "python/"+dr.RequestID+"__python_output.pkl"),
		filepath.Join(base, "data/"+dr.RequestID+"__data_output.pkl"),
	}
	for _, f := range files {
		exists, _ := afero.Exists(dr.Fs, f)
		assert.True(t, exists, "%s should exist", f)
		content, _ := afero.ReadFile(dr.Fs, f)
		// Verify the file contains an extends statement (should be schema reference)
		assert.Contains(t, string(content), "extends \"", "header present")
	}
}

func TestPrependDynamicImports_AddsImports(t *testing.T) {
	dr := newTestResolver()
	pklFile := filepath.Join(dr.ActionDir, "file.pkl")
	_ = dr.Fs.MkdirAll(dr.ActionDir, 0o755)
	// initial content with amends line
	initial := "amends \"base.pkl\"\n\n"
	_ = afero.WriteFile(dr.Fs, pklFile, []byte(initial), 0o644)

	// create exec file to satisfy import existence check
	execFile := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
	_ = dr.Fs.MkdirAll(filepath.Dir(execFile), 0o755)
	_ = afero.WriteFile(dr.Fs, execFile, []byte("{}"), 0o644)

	assert.NoError(t, dr.PrependDynamicImports(pklFile))
	content, _ := afero.ReadFile(dr.Fs, pklFile)
	s := string(content)
	// Should still start with amends line
	assert.True(t, strings.HasPrefix(s, "amends"))
	// Import for exec alias should be present
	assert.Contains(t, s, "import \""+execFile+"\" as exec")
}

func TestPrepareWorkflowDir_CopiesAndCleans(t *testing.T) {
	dr := newTestResolver()
	fs := dr.Fs
	// setup project files
	_ = fs.MkdirAll(filepath.Join(dr.ProjectDir, "dir"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(dr.ProjectDir, "file.txt"), []byte("hello"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(dr.ProjectDir, "dir/file2.txt"), []byte("world"), 0o644)

	// first copy
	assert.NoError(t, dr.PrepareWorkflowDir())
	exists, _ := afero.Exists(fs, filepath.Join(dr.WorkflowDir, "file.txt"))
	assert.True(t, exists)

	// create stale file in workflow dir and ensure second run cleans it
	_ = afero.WriteFile(fs, filepath.Join(dr.WorkflowDir, "stale.txt"), []byte("x"), 0o644)
	assert.NoError(t, dr.PrepareWorkflowDir())
	staleExists, _ := afero.Exists(fs, filepath.Join(dr.WorkflowDir, "stale.txt"))
	assert.False(t, staleExists)
}

func TestAddPlaceholderImports_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{Fs: fs, Logger: logger}
	if err := dr.AddPlaceholderImports("/no/such/file.pkl"); err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestNewGraphResolver_Minimal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	env, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("env err: %v", err)
	}

	dr, err := NewGraphResolver(fs, context.Background(), env, nil, logger)
	if err == nil {
		// If resolver succeeded, sanity-check key fields
		if dr.Graph == nil || dr.FileRunCounter == nil {
			t.Errorf("expected Graph and FileRunCounter initialized")
		}
	}
}

func TestPrepareImportFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	dr := &DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   actionDir,
		RequestID:   "req1",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	// call function
	require.NoError(t, dr.PrepareImportFiles())

	// verify that expected stub files were created with minimal header lines
	expected := []struct{ folder, key string }{
		{"llm", "LLM.pkl"},
		{"client", "HTTP.pkl"},
		{"exec", "Exec.pkl"},
		{"python", "Python.pkl"},
		{"data", "Data.pkl"},
	}

	for _, e := range expected {
		p := filepath.Join(dr.ActionDir, e.folder, dr.RequestID+"__"+e.folder+"_output.pkl")
		exists, _ := afero.Exists(fs, p)
		require.True(t, exists, "file %s should exist", p)
		// simple read check
		b, err := afero.ReadFile(fs, p)
		require.NoError(t, err)
		require.Contains(t, string(b), e.key)
	}
}

func TestPrepareImportFilesAndPrependDynamicImports(t *testing.T) {
	fs := afero.NewMemMapFs()

	actionDir := "/action"
	requestID := "abc123"
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// create resolver
	dr := &DependencyResolver{
		Fs:             fs,
		ActionDir:      actionDir,
		RequestID:      requestID,
		RequestPklFile: filepath.Join(actionDir, "api", requestID+"__request.pkl"),
		Logger:         logger,
		Context:        ctx,
	}

	// call PrepareImportFiles – should create multiple skeleton files
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "api"), 0o755))
	assert.NoError(t, dr.PrepareImportFiles())

	// check a couple of expected files exist
	expected := []string{
		filepath.Join(actionDir, "exec", requestID+"__exec_output.pkl"),
		filepath.Join(actionDir, "data", requestID+"__data_output.pkl"),
	}
	for _, f := range expected {
		exists, _ := afero.Exists(fs, f)
		assert.True(t, exists, f)
	}

	// create a dummy .pkl file with minimal content
	wfDir := "/workflow"
	assert.NoError(t, fs.MkdirAll(wfDir, 0o755))
	pklPath := filepath.Join(wfDir, "workflow.pkl")
	content := "amends \"base\"\n" // just an amends line
	assert.NoError(t, afero.WriteFile(fs, pklPath, []byte(content), 0o644))

	// run PrependDynamicImports
	assert.NoError(t, dr.PrependDynamicImports(pklPath))

	// the updated file should now contain some import lines (e.g., pkl:json)
	updated, err := afero.ReadFile(fs, pklPath)
	assert.NoError(t, err)
	assert.Contains(t, string(updated), "import \"pkl:json\"")
}

func TestAddPlaceholderImports_Errors(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	fs := afero.NewMemMapFs()
	tmp := t.TempDir()
	actionDir := filepath.Join(tmp, "action")

	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logging.NewTestLogger(),
		ActionDir:      actionDir,
		DataDir:        filepath.Join(tmp, "data"),
		RequestID:      "req",
		RequestPklFile: filepath.Join(tmp, "request.pkl"),
	}

	// 1) file not found
	if err := dr.AddPlaceholderImports("/does/not/exist.pkl"); err == nil {
		t.Errorf("expected error for missing file path")
	}

	// 2) file without actionID line - use assets for extends
	filePath := filepath.Join(tmp, "no_id.pkl")
	_ = afero.WriteFile(fs, filePath, []byte("extends \""+workspace.GetImportPath("Exec.pkl")+"\"\n"), 0o644)

	if err := dr.AddPlaceholderImports(filePath); err == nil {
		t.Errorf("expected error when action id missing but got nil")
	}
}

func TestAddPlaceholderImports(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	baseDir := t.TempDir()
	actionDir := filepath.Join(baseDir, "action")
	dataDir := filepath.Join(baseDir, "data")
	requestID := "req1"

	// create directories for placeholder files
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755))
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755))

	// create minimal pkl file expected by AppendDataEntry
	dataPklPath := filepath.Join(actionDir, "data", requestID+"__data_output.pkl")
	minimalContent := []byte("Files {}\n")
	assert.NoError(t, afero.WriteFile(fs, dataPklPath, minimalContent, 0o644))

	// create input file containing actionID
	targetPkl := filepath.Join(actionDir, "exec", "sample.pkl")
	fileContent := []byte("ActionID = \"myAction\"\n")
	assert.NoError(t, afero.WriteFile(fs, targetPkl, fileContent, 0o644))

	dr := &DependencyResolver{
		Fs:        fs,
		ActionDir: actionDir,
		DataDir:   dataDir,
		RequestID: requestID,
		Context:   ctx,
		Logger:    logger,
	}

	// ensure DataDir has at least one file for PopulateDataFileRegistry
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(dataDir, "dummy.txt"), []byte("abc"), 0o644))

	// MOCK: Provide a safe LoadResourceFn to avoid nil dereference
	dr.LoadResourceFn = func(ctx context.Context, path string, typ ResourceType) (interface{}, error) {
		return &pklLLM.LLMImpl{}, nil
	}

	// run the function under test
	err := dr.AddPlaceholderImports(targetPkl)
	assert.Error(t, err)
}

func TestPrepareImportFilesCreatesStubs(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:        fs,
		ActionDir: "/agent/action",
		RequestID: "abc",
		Context:   context.Background(),
	}

	err = dr.PrepareImportFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for one of the generated files and its header content with assets
	execPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
	content, err := afero.ReadFile(fs, execPath)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	// Should contain extends statement with schema reference
	if !strings.Contains(string(content), "extends \"") {
		t.Errorf("extends header not found in file: %s", execPath)
	}
}

func TestPrependDynamicImportsExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	dr := &DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   actionDir,
		RequestID:   "rid",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	// create directories and dummy files for Check=true imports
	folders := []string{"llm", "client", "exec", "python", "data"}
	for _, f := range folders {
		p := filepath.Join(dr.ActionDir, f, dr.RequestID+"__"+f+"_output.pkl")
		require.NoError(t, fs.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, afero.WriteFile(fs, p, []byte(""), 0o644))
	}
	// Also the request pkl file itself counted with alias "request" (Check=true)
	dr.RequestPklFile = filepath.Join(dr.ActionDir, "req.pkl")
	require.NoError(t, fs.MkdirAll(filepath.Dir(dr.RequestPklFile), 0o755))
	require.NoError(t, afero.WriteFile(fs, dr.RequestPklFile, []byte(""), 0o644))

	// Create test file with only amends line
	testPkl := filepath.Join(dr.ActionDir, "test.pkl")
	content := "amends \"something\"\n"
	require.NoError(t, afero.WriteFile(fs, testPkl, []byte(content), 0o644))

	// Call function
	require.NoError(t, dr.PrependDynamicImports(testPkl))

	// Read back file and ensure dynamic import lines exist (e.g., import "pkl:json") and request alias line
	out, err := afero.ReadFile(fs, testPkl)
	require.NoError(t, err)
	s := string(out)
	require.True(t, strings.Contains(s, "import \"pkl:json\""))
	require.True(t, strings.Contains(s, "import \""+dr.RequestPklFile+"\" as request"))
}
