// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !js

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepsconfig "github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestExtractHostPortFromParsedURL_InvalidPort(t *testing.T) {
	host, port := extractHostPortFromParsedURL(&url.URL{Host: "host:badport"}, "default", 11434)
	assert.Equal(t, "host", host)
	assert.Equal(t, 11434, port)
}

func TestExecutor_Execute_ResolveConfigError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.Execute(ctx, &domain.ChatConfig{
		Model:  "m",
		Prompt: "p",
		Role:   "{{ unknown() }}",
	})
	require.Error(t, err)
}

func TestExecutor_Execute_UnknownBackend(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.Execute(ctx, &domain.ChatConfig{
		Model:   "m",
		Prompt:  "p",
		Backend: "nonexistent-backend",
	})
	require.Error(t, err)
}

func TestHandleToolCalls_FollowUpError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	calls := 0
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": map[string]interface{}{
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id": "1",
							"function": map[string]interface{}{
								"name": "my_tool", "arguments": `{}`,
							},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(stdhttp.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	e := NewExecutor(srv.URL)
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.handleToolCalls(
		ctx,
		&domain.ChatConfig{},
		[]domain.Tool{{Name: "my_tool", Execute: func(_ map[string]interface{}) (string, error) {
			return "ok", nil
		}}},
		"m",
		[]map[string]interface{}{{"role": "user", "content": "hi"}},
		ChatRequestConfig{},
		&OllamaBackend{},
		srv.URL,
		map[string]interface{}{
			"message": map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id": "1",
						"function": map[string]interface{}{
							"name": "my_tool", "arguments": `{}`,
						},
					},
				},
			},
		},
		time.Second,
	)
	require.Error(t, err)
}

func TestCallBackendWithFallback_ErrorAndRetryErr(t *testing.T) {
	e := NewExecutor("")
	e.SetHTTPClientForTesting(&MockHTTPClient{Error: errors.New("backend down")})
	cfg := &domain.ChatConfig{Model: "m", Backend: "ollama"}
	routes := []kdepsconfig.ModelEntry{
		{Model: "m", Backend: "ollama", BaseURL: "http://x"},
		{Model: "fb", Backend: "ollama", BaseURL: "http://y"},
	}
	out := e.callBackendWithFallback(
		&OllamaBackend{},
		"http://localhost:11434",
		map[string]interface{}{"model": "m"},
		time.Second,
		routes,
		cfg,
		[]map[string]interface{}{{"role": "user", "content": "hi"}},
		ChatRequestConfig{},
	)
	assert.Contains(t, out, "error")
}

func TestEvaluateStringOrLiteral_LiteralType(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	got, err := e.evaluateStringOrLiteral(expression.NewEvaluator(ctx.API), ctx, "/absolute/path/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "/absolute/path/file.txt", got)
}

func TestExecuteTool_MCPPath(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.executeTool(domain.Tool{
		Name: "mcp_tool",
		MCP:  &domain.MCPConfig{Server: "false"},
	}, `{}`, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MCP tool execution failed")
}

func TestFormatToolResultContent_MarshalFallback(t *testing.T) {
	ch := make(chan int)
	got := formatToolResultContent(map[string]interface{}{"content": ch})
	assert.NotEmpty(t, got)
}

func TestRouter_SelectCostOptimized_EmptyModels(t *testing.T) {
	r := &Router{models: nil}
	entry, err := r.selectCostOptimized("hello")
	require.NoError(t, err)
	assert.Nil(t, entry)
}

func TestDownload_HTTPErrorStatus(t *testing.T) {
	origHTTP := httpGet
	t.Cleanup(func() { httpGet = origHTTP })
	httpGet = func(_ string) (*stdhttp.Response, error) {
		return &stdhttp.Response{StatusCode: stdhttp.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	m := NewLlamafileManagerWithDir(testLogger(), t.TempDir())
	_, err := m.download("https://example.com/missing.llamafile")
	require.Error(t, err)
}

func TestDownload_AlreadyCachedSkipsHTTP(t *testing.T) {
	dir := t.TempDir()
	m := NewLlamafileManagerWithDir(testLogger(), dir)
	cached := filepath.Join(dir, "cached.llamafile")
	require.NoError(t, os.WriteFile(cached, []byte("bin"), 0750))
	dest, err := m.download("https://example.com/cached.llamafile")
	require.NoError(t, err)
	assert.Equal(t, cached, dest)
}

func TestWriteDownloadToFile_CloseAndRenameErrors(t *testing.T) {
	origFS := AppFS
	origClose := closeDownloadFile
	origMove := fileflowMoveFunc
	t.Cleanup(func() {
		AppFS = origFS
		closeDownloadFile = origClose
		fileflowMoveFunc = origMove
	})
	AppFS = afero.NewMemMapFs()

	closeDownloadFile = func(_ interface{ Close() error }) error {
		return errors.New("close fail")
	}
	err := writeDownloadToFile("/dest.bin", strings.NewReader("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close downloaded file")

	closeDownloadFile = origClose
	fileflowMoveFunc = func(_, _ string) (string, error) { return "", errors.New("rename fail") }
	err = writeDownloadToFile("/dest2.bin", strings.NewReader("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "move downloaded file")
}

func TestLlamafileServe_FindFreePortError(t *testing.T) {
	orig := netListenConfigListen
	t.Cleanup(func() { netListenConfigListen = orig })
	netListenConfigListen = func(_ context.Context, _, _ string) (net.Listener, error) {
		return nil, errors.New("no port")
	}
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	mem := afero.NewMemMapFs()
	AppFS = mem
	require.NoError(t, afero.WriteFile(mem, "m.llamafile", []byte("bin"), 0755))
	m, err := NewLlamafileManager(testLogger())
	require.NoError(t, err)
	_, err = m.Serve("m.llamafile", 0)
	require.Error(t, err)
}

func TestServeFileModelIfNeeded_SetsBaseURLFromHealthyPort(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	}))
	t.Cleanup(srv.Close)
	var port int
	_, _ = fmt.Sscanf(strings.TrimPrefix(srv.URL, "http://127.0.0.1:"), "%d", &port)

	dir := t.TempDir()
	modelPath := filepath.Join(dir, "test.llamafile")
	require.NoError(t, os.WriteFile(modelPath, []byte("bin"), 0755))

	mgr := NewModelManagerFromServiceInterface(NewMockModelService())
	cfg := &domain.ChatConfig{Model: modelPath, Backend: backendFile}
	mgr.serveFileModelIfNeeded(cfg, port)
	assert.Contains(t, cfg.BaseURL, "127.0.0.1")
}

func TestServeFileModel_MakeExecutableFail(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	mem := afero.NewMemMapFs()
	AppFS = mem
	path := "ro.llamafile"
	require.NoError(t, afero.WriteFile(mem, path, []byte("data"), 0444))
	mgr := NewModelManagerFromServiceInterface(NewMockModelService())
	_, err := mgr.serveFileModel(path, 18082)
	require.Error(t, err)
}

func TestModelService_PrepareLlamafileErrors(t *testing.T) {
	s := NewModelService(slog.Default())
	_, _, err := s.prepareLlamafile("/nonexistent/model.llamafile")
	require.Error(t, err)
}

func TestModelService_DownloadOllamaModel_Success(t *testing.T) {
	orig := execCommandContext
	t.Cleanup(func() { execCommandContext = orig })
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "echo", "ok")
	}
	s := NewModelService(slog.Default())
	require.NoError(t, s.downloadOllamaModel("m"))
}

func TestCallBackendWithEndpoint_MarshalError(t *testing.T) {
	e := NewExecutor("")
	_, err := e.callBackendWithEndpoint(&OllamaBackend{}, "http://localhost/", map[string]interface{}{
		"bad": make(chan int),
	}, time.Second, "")
	require.Error(t, err)
}

func TestAssertImageRequest_ShortContentEarlyReturn(t *testing.T) {
	req := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(
		`{"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`,
	))
	assertImageRequest(t, req, "")
}

func TestResolveModelForExecution_EnsureModelIgnored(t *testing.T) {
	mock := NewMockModelService()
	mock.DownloadModelFunc = func(_, _ string) error { return errors.New("ensure fail") }
	mgr := NewModelManagerFromServiceInterface(mock)
	e := NewExecutor("")
	e.SetModelManager(mgr)
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, _, _, err = e.resolveModelForExecution(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Model:  "m",
		Prompt: "p",
	})
	require.NoError(t, err)
}

func TestExecutor_Execute_SuccessPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"message": map[string]interface{}{"content": "ok"},
		})
	}))
	t.Cleanup(srv.Close)

	e := NewExecutor(srv.URL)
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	out, err := e.Execute(ctx, &domain.ChatConfig{
		Model:   "m",
		Prompt:  "hi",
		BaseURL: srv.URL,
	})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestExecutor_Execute_BuildMessagesError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.Execute(ctx, &domain.ChatConfig{
		Model:  "m",
		Prompt: "p",
		Files:  []string{"/nonexistent/image.png"},
	})
	require.Error(t, err)
}

func TestExecutor_Execute_BuildRequestError(t *testing.T) {
	e := &Executor{backendRegistry: NewBackendRegistry()}
	e.backendRegistry.SetBackendsForTesting(map[string]Backend{
		"ollama": &htcBuildRequestErrorBackend{},
	})
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.Execute(ctx, &domain.ChatConfig{Model: "m", Prompt: "p", Backend: "ollama"})
	require.Error(t, err)
}

func TestExecutor_Execute_HandleToolCallsReturnError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	calls := 0
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"message": map[string]interface{}{
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":       "1",
							"function": map[string]interface{}{"name": "t", "arguments": `{}`},
						},
					},
				},
			})
			return
		}
		w.WriteHeader(stdhttp.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	e := NewExecutor(srv.URL)
	e.SetToolExecutor(&failingToolExecutor{})
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.Execute(ctx, &domain.ChatConfig{
		Model:   "m",
		Prompt:  "p",
		BaseURL: srv.URL,
		Tools: []domain.Tool{{
			Name: "t",
			Execute: func(_ map[string]interface{}) (string, error) {
				return "ok", nil
			},
		}},
	})
	require.Error(t, err)
}

func TestResolveModelForExecution_PromptError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, _, _, err = e.resolveModelForExecution(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Model:  "m",
		Prompt: "{{ unknown() }}",
	})
	require.Error(t, err)
}

func TestResolveModelForExecution_EnsureModelErrorBranch(t *testing.T) {
	orig := ensureModelForTest
	t.Cleanup(func() { ensureModelForTest = orig })
	ensureModelForTest = func(_ *ModelManager, _ *domain.ChatConfig) error {
		return errors.New("ensure fail")
	}
	mgr := NewModelManagerFromServiceInterface(NewMockModelService())
	e := NewExecutor("")
	e.SetModelManager(mgr)
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, _, _, err = e.resolveModelForExecution(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Model:  "m",
		Prompt: "p",
	})
	require.NoError(t, err)
}

func TestExecuteTool_MCPSuccess(t *testing.T) {
	orig := mcpExecuteToolFunc
	t.Cleanup(func() { mcpExecuteToolFunc = orig })
	mcpExecuteToolFunc = func(_ *domain.MCPConfig, _ string, _ map[string]interface{}) (string, error) {
		return `{"ok":true}`, nil
	}
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	out, err := e.executeTool(domain.Tool{Name: "mcp", MCP: &domain.MCPConfig{Server: "mock"}}, `{}`, ctx)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestExecuteTool_StoreToolArgumentsError(t *testing.T) {
	orig := storeToolArgumentSet
	t.Cleanup(func() { storeToolArgumentSet = orig })
	storeToolArgumentSet = func(_ *executor.ExecutionContext, _ string, _ interface{}, _ string) error {
		return errors.New("store fail")
	}
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Resources["res"] = &domain.Resource{ActionID: "res"}
	e.toolExecutor = &failingToolExecutor{}
	_, err = e.executeTool(domain.Tool{Name: "t", Script: "res"}, `{"k":"v"}`, ctx)
	require.Error(t, err)
}

func TestHandleToolCalls_ExecuteToolCallsInjectorError(t *testing.T) {
	orig := executeToolCallsErrInjector
	t.Cleanup(func() { executeToolCallsErrInjector = orig })
	executeToolCallsErrInjector = func() error { return errors.New("inject fail") }
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.handleToolCalls(
		ctx,
		&domain.ChatConfig{},
		[]domain.Tool{{Name: "t", Execute: func(_ map[string]interface{}) (string, error) { return "ok", nil }}},
		"m",
		[]map[string]interface{}{{"role": "user", "content": "hi"}},
		ChatRequestConfig{},
		&OllamaBackend{},
		"http://localhost",
		map[string]interface{}{
			"message": map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id":       "1",
						"function": map[string]interface{}{"name": "t", "arguments": `{}`},
					},
				},
			},
		},
		time.Second,
	)
	require.Error(t, err)
}

func TestStoreToolArguments_UnprefixedSetError(t *testing.T) {
	origSet := storeToolArgumentSet
	t.Cleanup(func() { storeToolArgumentSet = origSet })
	calls := 0
	storeToolArgumentSet = func(ctx *executor.ExecutionContext, key string, value interface{}, storage string) error {
		calls++
		if calls == 2 {
			return errors.New("second set fail")
		}
		return ctx.Set(key, value, storage)
	}
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	err = e.storeToolArguments(domain.Tool{Name: "t"}, map[string]interface{}{"k": "v"}, ctx)
	require.Error(t, err)
}

func TestResolveCachedModel_NotFound(t *testing.T) {
	m := NewLlamafileManagerWithDir(testLogger(), t.TempDir())
	_, err := m.resolveCachedModel("missing.llamafile")
	require.Error(t, err)
}

func TestLlamafileServe_HealthTimeoutAfterStart(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "run.llamafile")
	require.NoError(t, os.WriteFile(modelPath, []byte("#!/bin/sh\nexit 0\n"), 0755))
	m := NewLlamafileManagerWithDir(testLogger(), dir)
	_, err := m.Serve(modelPath, 19998)
	require.Error(t, err)
}

func TestServeFileModel_MakeExecutableErrorPath(t *testing.T) {
	origChmod := chmodLlamafile
	t.Cleanup(func() { chmodLlamafile = origChmod })
	chmodLlamafile = func(_ string, _ os.FileMode) error { return errors.New("chmod fail") }

	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(modelPath, []byte("data"), 0644))
	mgr := NewModelManagerFromServiceInterface(NewMockModelService())
	_, err := mgr.serveFileModel(modelPath, 0)
	require.Error(t, err)
}

func TestModelService_PrepareLlamafile_MakeExecutableError(t *testing.T) {
	origChmod := chmodLlamafile
	t.Cleanup(func() { chmodLlamafile = origChmod })
	chmodLlamafile = func(_ string, _ os.FileMode) error { return errors.New("chmod fail") }

	dir := t.TempDir()
	modelPath := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(modelPath, []byte("data"), 0644))
	s := NewModelService(slog.Default())
	_, _, err := s.prepareLlamafile(modelPath)
	require.Error(t, err)
}

func TestModelService_ServeOllamaModel_SetenvWarn(t *testing.T) {
	orig := execCommandContext
	t.Cleanup(func() { execCommandContext = orig })
	execCommandContext = func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "list" {
			return exec.CommandContext(ctx, "false")
		}
		return exec.CommandContext(ctx, "echo", "ok")
	}
	origSetenv := osSetenv
	t.Cleanup(func() { osSetenv = origSetenv })
	osSetenv = func(_ string, _ string) error { return errors.New("setenv fail") }
	s := NewModelService(slog.Default())
	require.NoError(t, s.serveOllamaModel("m", "127.0.0.1", 11434))
}

func TestDownload_BasenameFallback(t *testing.T) {
	origHTTP := httpGet
	t.Cleanup(func() { httpGet = origHTTP })
	httpGet = func(_ string) (*stdhttp.Response, error) {
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("bin"))),
		}, nil
	}
	m := NewLlamafileManagerWithDir(testLogger(), t.TempDir())
	dest, err := m.download("/")
	require.NoError(t, err)
	assert.Contains(t, dest, "model.llamafile")
}
