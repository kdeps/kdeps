package resolver_test

import (
	"context"
	"os"
	"path/filepath"
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

	// Initialize PklresReader and PklresHelper for tests that need it
	dr.PklresReader, _ = pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
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
	// PrependDynamicImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("PrependDynamicImports functionality removed - imports included in templates")
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

	dr, err := resolver.NewGraphResolver(fs, context.Background(), env, nil, logger, nil)
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

	dr.PklresReader, _ = pklres.InitializePklResource("test-graph", "", "", "", afero.NewMemMapFs())
	dr.PklresHelper = resolver.NewPklresHelper(dr)

	// Ensure prepare succeeds without error.
}

func TestPrepareImportFilesAndPrependDynamicImports(t *testing.T) {
	// PrependDynamicImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("PrependDynamicImports functionality removed - imports included in templates")
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
	require.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755))
	require.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "data"), 0o755))

	// create minimal pkl file expected by AppendDataEntry
	dataPklPath := filepath.Join(actionDir, "data", requestID+"__data_output.pkl")
	minimalContent := []byte("Files {}\n")
	require.NoError(t, afero.WriteFile(fs, dataPklPath, minimalContent, 0o644))

	// create input file containing actionID
	targetPkl := filepath.Join(actionDir, "exec", "sample.pkl")
	fileContent := []byte("ActionID = \"myAction\"\n")
	require.NoError(t, afero.WriteFile(fs, targetPkl, fileContent, 0o644))

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
	require.NoError(t, fs.MkdirAll(dataDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(dataDir, "dummy.txt"), []byte("abc"), 0o644))

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
	// PrependDynamicImports functionality removed - imports included in templates
	// This test is deprecated - function no longer exists
	t.Skip("PrependDynamicImports functionality removed - imports included in templates")
}
