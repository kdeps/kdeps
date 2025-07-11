package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklres "github.com/kdeps/kdeps/pkg/pklres"
	assets "github.com/kdeps/schema/assets"
	pklData "github.com/kdeps/schema/gen/data"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestAppendDataEntry_Direct verifies the happy-path where new files are merged
// into an existing (initially empty) data.pkl file without any monkey-patching.
// It uses a real EvalPkl run, so it depends on `pkl` binary being available in PATH –
// which the project's other tests already rely on.
func TestAppendDataEntry_Direct(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	dataDir := filepath.Join(actionDir, "data")
	require.NoError(t, fs.MkdirAll(dataDir, 0o755))

	ctx := context.Background()

	// Seed minimal valid PKL content using assets workspace instead of hardcoded URL
	initialContent := "extends \"" + workspace.GetImportPath("Data.pkl") + "\"\n\nFiles {}\n"
	pklPath := filepath.Join(dataDir, "req__data_output.pkl")
	require.NoError(t, afero.WriteFile(fs, pklPath, []byte(initialContent), 0o644))

	// Verify the initial content uses the workspace path
	initialBytes, err := afero.ReadFile(fs, pklPath)
	require.NoError(t, err)
	initialString := string(initialBytes)
	require.Contains(t, initialString, workspace.GetImportPath("Data.pkl"), "Initial PKL file should use workspace path")

	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: actionDir,
		RequestID: "req",
		Logger:    logging.NewTestLogger(),
	}

	// Initialize PklresHelper required by AppendDataEntry
	dr.PklresHelper = NewPklresHelper(dr)

	// Provide an in-memory PklResourceReader for pklres interactions
	readerDirect, errInit := pklres.InitializePklResource(":memory:")
	require.NoError(t, errInit)
	dr.PklresReader = readerDirect

	// Prepare new data to merge.
	files := map[string]map[string]string{
		"agentX": {
			"hello.txt": "SGVsbG8=", // "Hello" already base64-encoded
		},
	}
	newData := &pklData.DataImpl{Files: files}

	require.NoError(t, dr.AppendDataEntry("testResource", newData))

	// Legacy on-disk file is no longer modified – just ensure no error occurred and
	// the pklres record has been written.
	record, errRec := dr.PklresHelper.retrievePklContent("data", "testResource")
	require.NoError(t, errRec)
	require.Contains(t, record, "agentX")
}

// note: createStubPkl helper is provided by resource_response_eval_extra_test.go

func TestAppendChatEntry_Basic(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   context.Background(),
		ActionDir: "/action",
		FilesDir:  "/files",
		RequestID: "req1",
		LoadResourceFn: func(_ context.Context, path string, _ ResourceType) (interface{}, error) {
			// Return empty LLMImpl so AppendChatEntry has a map to update
			empty := make(map[string]*pklLLM.ResourceChat)
			return &pklLLM.LLMImpl{Resources: empty}, nil
		},
	}

	// Initialize PklresHelper and PklresReader for the test
	dr.PklresHelper = NewPklresHelper(dr)

	// Initialize an in-memory PklResourceReader
	reader, errInit := pklres.InitializePklResource(":memory:")
	require.NoError(t, errInit)
	dr.PklresReader = reader

	// Create dirs in memfs that AppendChatEntry expects
	_ = fs.MkdirAll(filepath.Join(dr.ActionDir, "llm"), 0o755)
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	chat := &pklLLM.ResourceChat{
		Model:  "test-model",
		Prompt: ptr("hello"),
	}

	if err := dr.AppendChatEntry("resA", chat); err != nil {
		t.Fatalf("AppendChatEntry returned error: %v", err)
	}

	// Verify pklres record exists
	record, err := dr.PklresHelper.retrievePklContent("llm", "resA")
	if err != nil {
		t.Fatalf("expected pklres record for llm/resA, got error: %v", err)
	}
	if record == "" {
		t.Fatalf("expected non-empty pklres record for llm/resA")
	}
}

func TestAppendHTTPEntry_Basic(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	dr := &DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   context.Background(),
		ActionDir: "/action",
		FilesDir:  "/files",
		RequestID: "req1",
		LoadResourceFn: func(_ context.Context, path string, _ ResourceType) (interface{}, error) {
			empty := make(map[string]*pklHTTP.ResourceHTTPClient)
			return &pklHTTP.HTTPImpl{Resources: empty}, nil
		},
	}

	dr.PklresHelper = NewPklresHelper(dr)
	r, _ := pklres.InitializePklResource(":memory:")
	dr.PklresReader = r

	_ = fs.MkdirAll(filepath.Join(dr.ActionDir, "client"), 0o755)
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	client := &pklHTTP.ResourceHTTPClient{
		Method: "GET",
		Url:    "aHR0cHM6Ly93d3cuZXhhbXBsZS5jb20=", // base64 of https://www.example.com
	}

	if err := dr.AppendHTTPEntry("httpRes", client); err != nil {
		t.Fatalf("AppendHTTPEntry returned error: %v", err)
	}

	// No disk output expected in new pklres flow
}

func ptr(s string) *string { return &s }
