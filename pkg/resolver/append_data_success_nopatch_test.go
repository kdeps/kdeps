package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
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
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	dataDir := filepath.Join(actionDir, "data")
	require.NoError(t, fs.MkdirAll(dataDir, 0o755))

	ctx := context.Background()
	schemaVer := schema.SchemaVersion(ctx)

	// Seed minimal valid PKL content so pklData.LoadFromPath succeeds.
	initialContent := "extends \"package://schema.kdeps.com/core@" + schemaVer + "#/Data.pkl\"\n\nFiles {}\n"
	pklPath := filepath.Join(dataDir, "req__data_output.pkl")
	require.NoError(t, afero.WriteFile(fs, pklPath, []byte(initialContent), 0o644))

	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: actionDir,
		RequestID: "req",
		Logger:    logging.NewTestLogger(),
	}

	// Prepare new data to merge.
	files := map[string]map[string]string{
		"agentX": {
			"hello.txt": "SGVsbG8=", // "Hello" already base64-encoded
		},
	}
	newData := &pklData.DataImpl{Files: &files}

	require.NoError(t, dr.AppendDataEntry("testResource", newData))

	// Validate merged content.
	mergedBytes, err := afero.ReadFile(fs, pklPath)
	require.NoError(t, err)
	merged := string(mergedBytes)
	require.Contains(t, merged, "[\"agentX\"]")
	require.Contains(t, merged, schemaVer)
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
			return &pklLLM.LLMImpl{Resources: &empty}, nil
		},
	}

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

	// Verify pkl file written
	pklPath := filepath.Join(dr.ActionDir, "llm", dr.RequestID+"__llm_output.pkl")
	if exists, _ := afero.Exists(fs, pklPath); !exists {
		t.Fatalf("expected output file %s to exist", pklPath)
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
			return &pklHTTP.HTTPImpl{Resources: &empty}, nil
		},
	}
	_ = fs.MkdirAll(filepath.Join(dr.ActionDir, "client"), 0o755)
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	client := &pklHTTP.ResourceHTTPClient{
		Method: "GET",
		Url:    "aHR0cHM6Ly93d3cuZXhhbXBsZS5jb20=", // base64 of https://www.example.com
	}

	if err := dr.AppendHTTPEntry("httpRes", client); err != nil {
		t.Fatalf("AppendHTTPEntry returned error: %v", err)
	}

	pklPath := filepath.Join(dr.ActionDir, "client", dr.RequestID+"__client_output.pkl")
	if exists, _ := afero.Exists(fs, pklPath); !exists {
		t.Fatalf("expected HTTP output pkl %s to exist", pklPath)
	}
}

func ptr(s string) *string { return &s }
