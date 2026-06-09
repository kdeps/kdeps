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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"os/exec"
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

// ── adapter ─────────────────────────────────────────────────────────────────

func TestAdapter_Execute_InvalidConfig(t *testing.T) {
	a := NewAdapter("")
	_, err := a.Execute(&executor.ExecutionContext{}, "not-chat-config")
	require.Error(t, err)
}

func TestAdapter_Execute_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := NewAdapter("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	cfg, err := parseChatConfig(&domain.ChatConfig{Model: "m", Prompt: "p"})
	require.NoError(t, err)
	assert.Equal(t, "m", cfg.Model)
	_, _ = a.Execute(ctx, cfg)
}

func TestParseChatConfig_InvalidType(t *testing.T) {
	_, err := parseChatConfig(42)
	require.Error(t, err)
}

func TestParseChatConfig_Valid(t *testing.T) {
	cfg, err := parseChatConfig(&domain.ChatConfig{Model: "m", Prompt: "p"})
	require.NoError(t, err)
	assert.Equal(t, "m", cfg.Model)
}

// ── backend helpers ─────────────────────────────────────────────────────────

func TestMarshalBackendRequest_Error(t *testing.T) {
	_, err := marshalBackendRequest(map[string]interface{}{"bad": make(chan int)})
	require.Error(t, err)
}

func TestNewBackendPostRequest_InvalidURL(t *testing.T) {
	_, err := newBackendPostRequest(context.Background(), "://bad", []byte("{}"))
	require.Error(t, err)
}

func TestApplyBackendAuthHeaders(t *testing.T) {
	req, err := stdhttp.NewRequest(stdhttp.MethodPost, "http://localhost/", nil)
	require.NoError(t, err)

	applyBackendAuthHeaders(req, defaultOpenAIBackend, "sk-test")
	assert.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))

	req2, err := stdhttp.NewRequest(stdhttp.MethodPost, "http://localhost/", nil)
	require.NoError(t, err)
	applyBackendAuthHeaders(req2, &AnthropicBackend{}, "sk-ant")
	assert.Equal(t, "2023-06-01", req2.Header.Get("Anthropic-Version"))
}

func TestCallBackendWithEndpoint_Errors(t *testing.T) {
	e := NewExecutor("")
	backend := &OllamaBackend{}

	_, err := e.callBackendWithEndpoint(backend, "://invalid", map[string]interface{}{"model": "m"}, time.Second)
	require.Error(t, err)
}

func TestCohereBackend_ParseResponse_NoText(t *testing.T) {
	b := &CohereBackend{}
	resp := &stdhttp.Response{
		StatusCode: stdhttp.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"meta":{}}`)),
	}
	out, err := b.ParseResponse(resp)
	require.NoError(t, err)
	assert.Empty(t, out)
}

// ── backend_file ──────────────────────────────────────────────────────────────

func TestDefaultModelsDir_UserHomeError(t *testing.T) {
	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) {
		return "", errors.New("no home")
	}
	_, err := DefaultModelsDir()
	require.Error(t, err)
}

// ── executor Execute paths ────────────────────────────────────────────────────

func TestExecutor_Execute_ResolveAndBuildErrors(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.Execute(ctx, &domain.ChatConfig{Model: "", Prompt: "p"})
	require.Error(t, err)

	_, _, _, err = e.resolveModelForExecution(nil, ctx, &domain.ChatConfig{Model: "{{ x }}", Prompt: "p"})
	require.Error(t, err)
}

func TestResolveModelForExecution_RouterMissing(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_ROUTER", "")

	_, _, _, err = e.resolveModelForExecution(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Model:  "router",
		Prompt: "p",
	})
	require.Error(t, err)
}

func TestResolveModelForExecution_AllowlistOverride(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_MODELS", "allowed-only")

	model, _, _, err := e.resolveModelForExecution(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Model:  "blocked",
		Prompt: "p",
	})
	require.NoError(t, err)
	assert.Equal(t, "allowed-only", model)
}

func TestResolveModelForExecution_EnsureModelErrorIgnored(t *testing.T) {
	mock := NewMockModelService()
	mock.DownloadModelFunc = func(_, _ string) error { return errors.New("dl fail") }
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

func TestCallBackendWithFallback_RetryError(t *testing.T) {
	e := NewExecutor("")
	e.SetHTTPClientForTesting(&MockHTTPClient{Error: errors.New("net fail")})

	cfg := &domain.ChatConfig{Model: "m"}
	routes := []kdepsconfig.ModelEntry{{Model: "fallback", Backend: "ollama", BaseURL: "http://x"}}
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

func TestFormatExecuteResult_JSONParseFallbackContent(t *testing.T) {
	e := NewExecutor("")
	out, err := e.formatExecuteResult(map[string]interface{}{
		"message": map[string]interface{}{"content": "not-json"},
	}, &domain.ChatConfig{JSONResponse: true}, 0)
	require.NoError(t, err)
	m := out.(map[string]interface{})
	assert.Contains(t, m, "error")
}

func TestFormatExecuteResult_JSONParseHardFail(t *testing.T) {
	e := NewExecutor("")
	_, err := e.formatExecuteResult(map[string]interface{}{"raw": true}, &domain.ChatConfig{JSONResponse: true}, 0)
	require.Error(t, err)
}

func TestHandleToolCalls_Error(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.handleToolCalls(
		ctx,
		&domain.ChatConfig{},
		nil,
		"m",
		[]map[string]interface{}{{"role": "user", "content": "hi"}},
		ChatRequestConfig{},
		&htcBuildRequestErrorBackend{},
		"http://localhost",
		map[string]interface{}{"message": map[string]interface{}{
			"tool_calls": []interface{}{
				map[string]interface{}{
					"id": "1",
					"function": map[string]interface{}{
						"name": "t", "arguments": `{}`,
					},
				},
			},
		}},
		time.Second,
	)
	require.Error(t, err)
}

func TestBuildMessages_ContentError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.buildMessages(expression.NewEvaluator(ctx.API), ctx, &domain.ChatConfig{
		Files: []string{"/nonexistent/image.png"},
	}, "user")
	require.Error(t, err)
}

func TestBuildContent_ImageReadError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.buildContent("hi", []string{"/nonexistent/image.png"}, ctx, expression.NewEvaluator(ctx.API))
	require.Error(t, err)
}

func TestEvaluateStringOrLiteral_Error(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.evaluateStringOrLiteral(nil, ctx, "{{ x }}")
	require.Error(t, err)
}

func TestExecuteTool_ResourceExecutorError(t *testing.T) {
	e := NewExecutor("")
	e.toolExecutor = &failingToolExecutor{}
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Resources["res"] = &domain.Resource{ActionID: "res"}

	_, err = e.executeTool(domain.Tool{Name: "t", Script: "res"}, `{}`, ctx)
	require.Error(t, err)
}

type failingToolExecutor struct{}

func (f *failingToolExecutor) ExecuteResource(_ *domain.Resource, _ *executor.ExecutionContext) (interface{}, error) {
	return nil, errors.New("tool exec failed")
}

func TestStoreToolArguments_SetError(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	err = e.storeToolArguments(domain.Tool{Name: "t"}, map[string]interface{}{"k": make(chan int)}, ctx)
	require.Error(t, err)
}

func TestFormatToolResultContent_AllBranches(t *testing.T) {
	assert.Equal(t, "Error: boom", formatToolResultContent(map[string]interface{}{"error": "boom"}))
	assert.Equal(t, "", formatToolResultContent(map[string]interface{}{}))
	assert.Equal(t, "text", formatToolResultContent(map[string]interface{}{"content": "text"}))
	assert.Equal(t, `{"k":"v"}`, formatToolResultContent(map[string]interface{}{
		"content": map[string]interface{}{"k": "v"},
	}))
}

// ── model_manager ─────────────────────────────────────────────────────────────

func TestResolveBackend_Defaults(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	cfg := &domain.ChatConfig{}
	assert.Equal(t, backendOllama, resolveBackend(cfg))

	t.Setenv("KDEPS_DEFAULT_BACKEND", "vllm")
	assert.Equal(t, "vllm", resolveBackend(cfg))
}

func TestDefaultPortForBackend_AllCases(t *testing.T) {
	assert.Equal(t, 11434, defaultPortForBackend(backendOllama))
	assert.Equal(t, backendFilePort, defaultPortForBackend(backendFile))
	assert.Equal(t, 8000, defaultPortForBackend("vllm"))
	assert.Equal(t, 16395, defaultPortForBackend("llamacpp"))
	assert.Equal(t, 16395, defaultPortForBackend("unknown"))
}

func TestModelManager_DownloadAndServeModel(t *testing.T) {
	mock := NewMockModelService()
	mgr := NewModelManagerFromServiceInterface(mock)
	require.NoError(t, mgr.DownloadModel("ollama", "m"))
	require.NoError(t, mgr.ServeModel("ollama", "m", "localhost", 11434))
}

func TestDownloadModelIfOnline_ErrorLogged(_ *testing.T) {
	mock := NewMockModelService()
	mock.DownloadModelFunc = func(_, _ string) error { return errors.New("dl fail") }
	mgr := NewModelManagerFromServiceInterface(mock)
	mgr.downloadModelIfOnline("ollama", "m")
}

func TestServeFileModelIfNeeded_SetsBaseURL(t *testing.T) {
	mock := NewMockModelService()
	mgr := NewModelManagerFromServiceInterface(mock)
	cfg := &domain.ChatConfig{Model: "test.llamafile", Backend: backendOllama}
	require.NoError(t, mgr.EnsureModel(cfg))
}

func TestServeFileModel_MakeExecutableError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewReadOnlyFs(afero.NewMemMapFs())

	mgr := NewModelManagerFromServiceInterface(NewMockModelService())
	_, err := mgr.serveFileModel("missing.llamafile", 0)
	require.Error(t, err)
}

// ── router ───────────────────────────────────────────────────────────────────

func TestRouter_SelectCostOptimized_DefaultFallback(t *testing.T) {
	r := &Router{models: []kdepsconfig.ModelEntry{{Model: "m", CostPerInputToken: ptrFloat(0.001)}}}
	entry, err := r.selectCostOptimized("hello")
	require.NoError(t, err)
	assert.Equal(t, "m", entry.Model)
}

func TestRouter_SelectRoundRobin_BadCounterType(t *testing.T) {
	r := &Router{models: []kdepsconfig.ModelEntry{{Model: "a"}, {Model: "b"}}}
	routerCounters.Store(routerFingerprint("id", r.models), "not-a-counter")
	entry, err := r.selectRoundRobin("id")
	require.NoError(t, err)
	assert.Equal(t, "a", entry.Model)
}

func ptrFloat(v float64) *float64 { return &v }

// ── service ───────────────────────────────────────────────────────────────────

func TestModelService_ServeModel_UnsupportedBackend(t *testing.T) {
	s := NewModelService(slog.Default())
	err := s.ServeModel("unsupported", "m", "h", 1)
	require.Error(t, err)
}

func TestModelService_DownloadOllamaModel_Error(t *testing.T) {
	orig := execCommandContext
	t.Cleanup(func() { execCommandContext = orig })
	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "false")
	}
	s := NewModelService(slog.Default())
	err := s.downloadOllamaModel("m")
	require.Error(t, err)
}

func TestModelService_ServeOllamaModel_SetEnvError(t *testing.T) {
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
	err := s.serveOllamaModel("m", "127.0.0.1", 11434)
	require.NoError(t, err)
}

// ── llamafile_manager ─────────────────────────────────────────────────────────

func TestResolveRelativeModel_AbsError(t *testing.T) {
	orig := filepathAbsFunc
	t.Cleanup(func() { filepathAbsFunc = orig })
	filepathAbsFunc = func(_ string) (string, error) {
		return "", errors.New("abs fail")
	}
	m, err := NewLlamafileManager(testLogger())
	require.NoError(t, err)
	_, err = m.resolveRelativeModel("../model.llamafile")
	require.Error(t, err)
}

func TestWriteDownloadToFile_CopyError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()

	err := writeDownloadToFile("/tmp/x.tmp", &failingReader{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "download write failed")
}

type failingReader struct{}

func (f *failingReader) Read(_ []byte) (int, error) { return 0, errors.New("read fail") }

func TestFindFreePort_UnexpectedAddrType(t *testing.T) {
	orig := netListenConfigListen
	t.Cleanup(func() { netListenConfigListen = orig })
	netListenConfigListen = func(_ context.Context, _, _ string) (net.Listener, error) {
		return badAddrListener{}, nil
	}
	_, err := FindFreePort()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected listener address type")
}

type badAddrListener struct{}

func (badAddrListener) Accept() (net.Conn, error) { return nil, nil }
func (badAddrListener) Close() error              { return nil }
func (badAddrListener) Addr() net.Addr            { return badAddr{} }

type badAddr struct{}

func (badAddr) Network() string { return "tcp" }
func (badAddr) String() string  { return "bad" }

func TestLlamafileServe_AlreadyHealthy(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	}))
	t.Cleanup(srv.Close)

	var port int
	_, _ = fmt.Sscanf(strings.TrimPrefix(srv.URL, "http://127.0.0.1:"), "%d", &port)

	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	mem := afero.NewMemMapFs()
	AppFS = mem
	path := "model.llamafile"
	require.NoError(t, afero.WriteFile(mem, path, []byte("bin"), 0755))

	m, err := NewLlamafileManager(testLogger())
	require.NoError(t, err)
	actualPort, err := m.Serve(path, port)
	require.NoError(t, err)
	assert.Equal(t, port, actualPort)
}

func TestLlamafileServe_StartFail(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	var port int
	_, _ = fmt.Sscanf(strings.TrimPrefix(srv.URL, "http://127.0.0.1:"), "%d", &port)

	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	mem := afero.NewMemMapFs()
	AppFS = mem
	path := "bad.llamafile"
	require.NoError(t, afero.WriteFile(mem, path, []byte("not-exec"), 0644))

	m, err := NewLlamafileManager(testLogger())
	require.NoError(t, err)
	_, err = m.Serve(path, port)
	require.Error(t, err)
}

func TestWaitForHealthy_Timeout(t *testing.T) {
	err := waitForHealthy("http://127.0.0.1:1", 1, 10*time.Millisecond)
	require.Error(t, err)
}

func TestDownload_WriteSuccess(t *testing.T) {
	origHTTP := httpGet
	t.Cleanup(func() { httpGet = origHTTP })

	httpGet = func(_ string) (*stdhttp.Response, error) {
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("llamafile-binary"))),
		}, nil
	}

	m := NewLlamafileManagerWithDir(testLogger(), t.TempDir())
	dest, err := m.download("https://example.com/newmodel.llamafile")
	require.NoError(t, err)
	assert.Contains(t, dest, "newmodel.llamafile")
}
