package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"

	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
)

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
