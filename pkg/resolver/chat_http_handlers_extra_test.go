package resolver

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"

	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
)

// TestHandleLLMChat ensures that the handler spawns the processing goroutine and writes a PKL file
func TestHandleLLMChat(t *testing.T) {
	// reuse helper from other tests to stub the pkl binary
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
	}

	// directories for AppendChatEntry
	_ = fs.MkdirAll(filepath.Join(dr.ActionDir, "llm"), 0o755)
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	// stub LoadResourceFn so AppendChatEntry loads an empty map
	dr.LoadResourceFn = func(_ context.Context, _ string, _ ResourceType) (interface{}, error) {
		empty := make(map[string]*pklLLM.ResourceChat)
		return &pklLLM.LLMImpl{Resources: &empty}, nil
	}

	// stub chat helpers
	dr.NewLLMFn = func(model string) (*ollama.LLM, error) { return nil, nil }

	done := make(chan struct{})
	dr.GenerateChatResponseFn = func(ctx context.Context, fs afero.Fs, _ *ollama.LLM, chat *pklLLM.ResourceChat, _ *tool.PklResourceReader, _ *logging.Logger) (string, error) {
		close(done)
		return "stub", nil
	}

	chat := &pklLLM.ResourceChat{Model: "test"}
	if err := dr.HandleLLMChat("act1", chat); err != nil {
		t.Fatalf("HandleLLMChat error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GenerateChatResponseFn not called")
	}

	time.Sleep(100 * time.Millisecond)
	pklPath := filepath.Join(dr.ActionDir, "llm", dr.RequestID+"__llm_output.pkl")
	if exists, _ := afero.Exists(fs, pklPath); !exists {
		t.Fatalf("expected chat pkl %s", pklPath)
	}
}

// TestHandleHTTPClient verifies DoRequestFn is invoked and PKL file written
func TestHandleHTTPClient(t *testing.T) {
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
	}
	_ = fs.MkdirAll(filepath.Join(dr.ActionDir, "client"), 0o755)
	_ = fs.MkdirAll(dr.FilesDir, 0o755)

	dr.LoadResourceFn = func(_ context.Context, _ string, _ ResourceType) (interface{}, error) {
		empty := make(map[string]*pklHTTP.ResourceHTTPClient)
		return &pklHTTP.HTTPImpl{Resources: &empty}, nil
	}

	var mu sync.Mutex
	called := false
	dr.DoRequestFn = func(*pklHTTP.ResourceHTTPClient) error {
		mu.Lock()
		called = true
		mu.Unlock()
		return nil
	}

	block := &pklHTTP.ResourceHTTPClient{Method: "GET", Url: "aHR0cHM6Ly9leGFtcGxlLmNvbQ=="}
	if err := dr.HandleHTTPClient("act1", block); err != nil {
		t.Fatalf("HandleHTTPClient error: %v", err)
	}

	// wait a bit for goroutine
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !called {
		t.Fatal("DoRequestFn not called")
	}
	mu.Unlock()

	pklPath := filepath.Join(dr.ActionDir, "client", dr.RequestID+"__client_output.pkl")
	if exists, _ := afero.Exists(fs, pklPath); !exists {
		t.Fatalf("expected http pkl %s", pklPath)
	}
}
