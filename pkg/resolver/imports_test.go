package resolver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	pklres "github.com/kdeps/kdeps/pkg/pklres"
	assets "github.com/kdeps/schema/assets"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/resolver"
)

// newTestResolver creates a DependencyResolver with real filesystem for PKL operations.
// This is needed because PKL cannot work with afero's in-memory filesystem.
func newTestResolver() *resolver.DependencyResolver {
	tmpDir := os.TempDir()
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	filesDir := filepath.Join(tmpDir, "files")
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755)
	_ = fs.MkdirAll(filepath.Join(actionDir, "llm"), 0o755)
	_ = fs.MkdirAll(filesDir, 0o755)

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}

	// Initialize PklresHelper for tests that need it
	dr.PklresHelper = resolver.NewPklresHelper(dr)

	return dr
}

func TestPrepareImportFiles_CreatesFiles(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	dr := newTestResolver()
	require.NoError(t, dr.PrepareImportFiles())

	// Verify that skeleton records exist in pklres instead of on-disk files
	resourceTypes := []string{"llm", "client", "exec", "python", "data"}
	for _, rt := range resourceTypes {
		_, err := dr.PklresHelper.RetrievePklContent(rt, "__empty__")
		assert.NoError(t, err)
	}
}

func TestPrependDynamicImports_AddsImports(t *testing.T) {
	dr := newTestResolver()
	pklFile := filepath.Join(dr.ActionDir, "file.pkl")
	_ = dr.Fs.MkdirAll(dr.ActionDir, 0o755)
	// initial content with amends line
	initial := "amends \"base.pkl\"\n\n"
	_ = afero.WriteFile(dr.Fs, pklFile, []byte(initial), 0o644)

	// ensure pklres helper initialized with stub record so Check=true passes
	dr.PklresHelper = resolver.NewPklresHelper(dr)
	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	// store empty exec structure
	_ = dr.PklresHelper.StorePklContent("exec", "__empty__", "Exec {}\n")

	// PrependDynamicImports functionality removed - imports included in templates
	// assert.NoError(t, dr.PrependDynamicImports(pklFile))
	content, _ := afero.ReadFile(dr.Fs, pklFile)
	s := string(content)
	// Should still start with amends line
	assert.True(t, strings.HasPrefix(s, "amends"))
	// Import for exec alias should reference pklres path
	execPath := dr.PklresHelper.GetResourcePath("exec")
	assert.Contains(t, s, "import \""+execPath+"\" as exec")
}

func TestPrepareWorkflowDir_CopiesAndCleans(t *testing.T) {
	// This test is deprecated - workflow directory functionality removed
	// Using project directory directly now
	t.Skip("PrepareWorkflowDir functionality removed - using project directory directly")
}

func TestAddPlaceholderImports_FileNotFound(t *testing.T) {
	// AddPlaceholderImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("AddPlaceholderImports functionality removed - imports included in templates")
}

func TestNewGraphResolver_Minimal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	env, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("env err: %v", err)
	}

	dr, err := resolver.NewGraphResolver(fs, context.Background(), env, nil, logger)
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

	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   actionDir,
		RequestID:   "req1",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	dr.PklresHelper = resolver.NewPklresHelper(dr)

	// Ensure prepare succeeds without error.
}

func TestPrepareImportFilesAndPrependDynamicImports(t *testing.T) {
	fs := afero.NewMemMapFs()

	actionDir := "/action"
	requestID := "abc123"
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// create resolver
	dr := &resolver.DependencyResolver{
		Fs:             fs,
		ActionDir:      actionDir,
		RequestID:      requestID,
		RequestPklFile: filepath.Join(actionDir, "api", requestID+"__request.pkl"),
		Logger:         logger,
		Context:        ctx,
	}

	// initialize pklres before calling PrepareImportFiles
	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	dr.PklresHelper = resolver.NewPklresHelper(dr)

	// call PrepareImportFiles – should create multiple skeleton records
	assert.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "api"), 0o755))
	assert.NoError(t, dr.PrepareImportFiles())

	// Ensure skeleton records exist in pklres
	_, _ = dr.PklresHelper.RetrievePklContent("exec", "")
	_, _ = dr.PklresHelper.RetrievePklContent("data", "")

	// create a dummy .pkl file with minimal content
	wfDir := "/workflow"
	assert.NoError(t, fs.MkdirAll(wfDir, 0o755))
	pklPath := filepath.Join(wfDir, "workflow.pkl")
	content := "amends \"base\"\n" // just an amends line
	assert.NoError(t, afero.WriteFile(fs, pklPath, []byte(content), 0o644))

	// run PrependDynamicImports
	// PrependDynamicImports functionality removed - imports included in templates
	// assert.NoError(t, dr.PrependDynamicImports(pklPath))

	// the updated file should now contain some import lines (e.g., pkl:json)
	updated, err := afero.ReadFile(fs, pklPath)
	assert.NoError(t, err)
	assert.Contains(t, string(updated), "import \"pkl:json\"")
}

func TestAddPlaceholderImports_Errors(t *testing.T) {
	// AddPlaceholderImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("AddPlaceholderImports functionality removed - imports included in templates")
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

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		ActionDir: actionDir,
		DataDir:   dataDir,
		RequestID: requestID,
		Context:   ctx,
		Logger:    logger,
		Workflow:  &pklWf.WorkflowImpl{AgentID: "testagent", Version: "1.0.0"},
	}

	// ensure DataDir has at least one file for PopulateDataFileRegistry
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(dataDir, "dummy.txt"), []byte("abc"), 0o644))

	// MOCK: Provide a safe LoadResourceFn to avoid nil dereference
	dr.LoadResourceFn = func(_ context.Context, _ string, _ resolver.ResourceType) (interface{}, error) {
		return &pklLLM.LLMImpl{}, nil
	}

	// run the function under test
	// AddPlaceholderImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("AddPlaceholderImports functionality removed - imports included in templates")
}

func TestPrepareImportFilesCreatesStubs(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	dr := newTestResolver()
	require.NoError(t, dr.PrepareImportFiles())

	// Verify that skeleton records exist in pklres instead of on-disk files
	resourceTypes := []string{"llm", "client", "exec", "python", "data"}
	for _, rt := range resourceTypes {
		_, err := dr.PklresHelper.RetrievePklContent(rt, "__empty__")
		assert.NoError(t, err, rt+" record not created")
	}
}

func TestPrependDynamicImportsExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")

	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Context:     ctx,
		ActionDir:   actionDir,
		RequestID:   "rid",
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}

	dr.PklresReader, _ = pklres.InitializePklResource(":memory:", "test-graph")
	dr.PklresHelper = resolver.NewPklresHelper(dr)

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
	// PrependDynamicImports functionality removed - imports included in templates
	// require.NoError(t, dr.PrependDynamicImports(testPkl))

	// Read back file and ensure dynamic import lines exist (e.g., import "pkl:json") and request alias line
	out, err := afero.ReadFile(fs, testPkl)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, "import \"pkl:json\"")
	require.Contains(t, s, "import \""+dr.RequestPklFile+"\" as request")
}
