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

package executor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func TestNewExecutionContext_SessionStorageError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	roDir := filepath.Join(t.TempDir(), "ro")
	require.NoError(t, os.Mkdir(roDir, 0555))
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "t"},
		Settings: domain.WorkflowSettings{
			Session: &domain.SessionConfig{Path: filepath.Join(roDir, "sessions.db")},
		},
	}
	_, err := NewExecutionContext(wf)
	require.Error(t, err)
}

func TestFilterByMimeType_UnknownExtensionSkipped(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file.unknownext")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0600))
	ctx := &ExecutionContext{}
	filtered, err := ctx.FilterByMimeType([]string{f}, "application/octet-stream")
	require.NoError(t, err)
	assert.Empty(t, filtered)
}

func TestEvaluateAgentParams_EvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	_, err = evaluateAgentParams(e, &domain.AgentCallConfig{
		Name:   "a",
		Params: map[string]interface{}{"x": "{{ unknown() }}"},
	}, ctx)
	require.Error(t, err)
}

func TestEnsureComponentDotEnv_LoadErrorNonMissing(t *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{componentDotEnv: map[string]map[string]string{}}
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".env"), []byte("not\ndotenv"), 0000))
	comp := &domain.Component{Dir: tmp}
	e.ensureComponentDotEnv("comp", comp, ctx)
	_, ok := ctx.componentDotEnv["comp"]
	assert.True(t, ok)
}

func TestDispatchPrimaryResource_UnknownType(t *testing.T) {
	e := covTestEngine()
	_, err := e.dispatchPrimaryResource(&domain.Resource{ActionID: "x"}, &ExecutionContext{})
	require.Error(t, err)
}

func TestHandleOnErrorContinue_FallbackSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	out, err := e.handleOnErrorContinue(
		&domain.Resource{ActionID: "r"},
		&domain.OnErrorConfig{Fallback: "ok"},
		ctx,
		errors.New("e"),
	)
	require.NoError(t, err)
	assert.Equal(t, "ok", out)
}

func TestShouldHandleError_EvalErrorSkips(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	on := &domain.OnErrorConfig{When: []domain.Expression{{Raw: "{{ unknown() }}"}}}
	assert.False(t, e.shouldHandleError(on, errors.New("e"), ctx))
}

func TestEvaluateFallback_ArrayError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	_, err = e.evaluateFallback([]interface{}{"{{ unknown() }}"}, ctx)
	require.Error(t, err)
}

func TestExecuteSingleInlineResource_Telephony(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetTelephonyExecutor(&covMockExecutor{result: "tel"})
	e.SetRegistry(reg)
	ctx := &ExecutionContext{Workflow: &domain.Workflow{}}
	_, err := e.executeSingleInlineResource(domain.InlineResource{
		Telephony: &domain.TelephonyActionConfig{},
	}, 0, ctx)
	require.NoError(t, err)
}

func TestBuildHTTPAccessorEnv_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.SetOutput("http", map[string]interface{}{
		"body":    "body",
		"headers": map[string]interface{}{"X": "v"},
	})
	env := buildHTTPAccessorEnv(ctx)
	assert.Equal(t, "body", env["responseBody"].(func(string) interface{})("http"))
	assert.Equal(t, "v", env["responseHeader"].(func(string, string) interface{})("http", "X"))
}

func TestAddRequestEnv_FileSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Request = &RequestContext{
		Files: []FileUpload{{Name: "f.txt", Path: filepath.Join(t.TempDir(), "f.txt"), MimeType: "text/plain"}},
	}
	require.NoError(t, os.WriteFile(ctx.Request.Files[0].Path, []byte("data"), 0600))
	env := map[string]interface{}{}
	e.addRequestEnv(env, ctx)
	req := env["request"].(map[string]interface{})
	assert.NotNil(t, req["file"].(func(string) interface{})("f.txt"))
	assert.NotNil(t, req["filepath"].(func(string) interface{})("f.txt"))
	assert.NotNil(t, req["filetype"].(func(string) interface{})("f.txt"))
}

func TestEvaluateLLMModel_ParseAndEvalFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	assert.Equal(t, "plain", e.evaluateLLMModel("plain", ctx))
	assert.Equal(t, "{{ unknown() }}", e.evaluateLLMModel("{{ unknown() }}", ctx))
	ctx.Set("modelName", "resolved", "memory")
	assert.Equal(t, "resolved", e.evaluateLLMModel("{{ get('modelName') }}", ctx))
	assert.Equal(t, "{{ get('n') }}", e.evaluateLLMModel("{{ get('n') }}", ctx))
}

func TestStartLLMTimeoutCountdown_Expires(t *testing.T) {
	e := covTestEngine()
	e.debugMode = false
	done := e.startLLMTimeoutCountdown("r", 5*time.Millisecond)
	require.NotNil(t, done)
	time.Sleep(15 * time.Millisecond)
	close(done)
}

func TestExecuteWithItems_EvalErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	_, err = e.ExecuteWithItems(&domain.Resource{Items: []string{"{{"}}, ctx)
	require.Error(t, err)
	_, err = e.ExecuteWithItems(&domain.Resource{Items: []string{"{{ unknown() }}"}}, ctx)
	require.Error(t, err)
}

func TestExecuteItemsIteration_ItemError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{err: errors.New("item fail")})
	e.SetRegistry(reg)
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	_, err = e.executeItemsIteration(
		&domain.Resource{Chat: &domain.ChatConfig{Model: "m", Prompt: "p"}},
		ctx,
		[]interface{}{1},
	)
	require.Error(t, err)
}

func TestMergeLLMItemIntoResult_NonMapResult(t *testing.T) {
	out := mergeLLMItemIntoResult(
		&domain.Resource{Chat: &domain.ChatConfig{}},
		map[string]interface{}{"id": 1},
		"not-map",
	)
	assert.Equal(t, "not-map", out)
}

func TestResolveAPIResponseSuccess_NilSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	ok, err := e.resolveAPIResponseSuccess(&domain.APIResponseConfig{}, e.BuildEvaluationEnvironment(ctx))
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestEvaluateResponseHeaders_StringMapBranch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expressionEvaluator(ctx)
	env := e.BuildEvaluationEnvironment(ctx)
	headers := e.evaluateResponseHeaders(map[string]string{"X": "v"}, env)
	assert.Equal(t, "v", headers["X"])
}

func TestApplyLLMMetadataToResponse_ExistingMeta(t *testing.T) {
	resp := map[string]interface{}{"_meta": map[string]interface{}{}}
	ctx := &ExecutionContext{LLMMetadata: &LLMMetadata{Model: "m", Backend: "b"}}
	covTestEngine().applyLLMMetadataToResponse(resp, ctx)
	meta := resp["_meta"].(map[string]interface{})
	assert.Equal(t, "m", meta["model"])
}

func TestHandleMimeTypeSelector_FilterByMimeTypeHookError(t *testing.T) {
	orig := errFilterByMimeType
	t.Cleanup(func() { errFilterByMimeType = orig })
	errFilterByMimeType = errors.New("filter fail")
	ctx := &ExecutionContext{}
	_, err := ctx.handleMimeTypeSelector([]string{"/tmp/x"}, "*.txt", []string{"mime:text/plain"})
	require.Error(t, err)
}

func TestFilterByMimeType_MimeMapFallback(t *testing.T) {
	orig := mimeTypeByExtension
	t.Cleanup(func() { mimeTypeByExtension = orig })
	mimeTypeByExtension = func(_ string) string { return "" }

	tmp := t.TempDir()
	f := filepath.Join(tmp, "data.json")
	require.NoError(t, os.WriteFile(f, []byte("{}"), 0600))
	ctx := &ExecutionContext{}
	filtered, err := ctx.FilterByMimeType([]string{f}, "application/json")
	require.NoError(t, err)
	assert.Contains(t, filtered, f)
}

func TestParseAgentWorkflow_ValidatorError(t *testing.T) {
	orig := newSchemaValidatorFunc
	t.Cleanup(func() { newSchemaValidatorFunc = orig })
	newSchemaValidatorFunc = func() (*validator.SchemaValidator, error) {
		return nil, errors.New("validator init fail")
	}
	_, err := parseAgentWorkflow("/any/path.yaml", "agent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema validator")
}

func TestEvaluateAgentParams_NonMapFallback(t *testing.T) {
	orig := agentParamsEvaluateFunc
	t.Cleanup(func() { agentParamsEvaluateFunc = orig })
	agentParamsEvaluateFunc = func(_ *Engine, _ interface{}, _ *ExecutionContext) (interface{}, error) {
		return "not-a-map", nil
	}
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	params, err := evaluateAgentParams(e, &domain.AgentCallConfig{Name: "a"}, ctx)
	require.NoError(t, err)
	assert.Empty(t, params)
}

func TestExecuteInlineAgent_ParamsError(t *testing.T) {
	orig := agentParamsEvaluateFunc
	t.Cleanup(func() { agentParamsEvaluateFunc = orig })
	agentParamsEvaluateFunc = func(_ *Engine, _ interface{}, _ *ExecutionContext) (interface{}, error) {
		return nil, errors.New("params eval failed")
	}
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	agentPath := filepath.Join("..", "..", "examples", "tools", "workflow.yaml")
	ctx.AgentPaths = map[string]string{"a": agentPath}
	_, err = e.executeInlineAgent(&domain.AgentCallConfig{Name: "a"}, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate params")
}

func TestHandleOnErrorContinue_FallbackExpressionEvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	out, err := e.handleOnErrorContinue(
		&domain.Resource{ActionID: "r"},
		&domain.OnErrorConfig{Fallback: "{{ unknown() }}"},
		ctx,
		errors.New("exec"),
	)
	require.NoError(t, err)
	assert.Equal(t, "{{ unknown() }}", out)
}

func TestExecuteOnErrorExpressions_NoExprs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	require.NoError(t, e.executeOnErrorExpressions(&domain.Resource{ActionID: "r"}, ctx, errors.New("e")))
}

func TestExecuteSingleInlineResource_Browser(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetBrowserExecutor(&covMockExecutor{result: "browser"})
	e.SetRegistry(reg)
	ctx := &ExecutionContext{Workflow: &domain.Workflow{}}
	_, err := e.executeSingleInlineResource(domain.InlineResource{
		Browser: &domain.BrowserConfig{},
	}, 0, ctx)
	require.NoError(t, err)
}

func TestEvaluateLLMModel_ParseError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	assert.Equal(t, "{{", e.evaluateLLMModel("{{", ctx))
}

func TestStartLLMTimeoutCountdown_RemainingZero(_ *testing.T) {
	e := covTestEngine()
	e.debugMode = false
	done := e.startLLMTimeoutCountdown("r", 1*time.Millisecond)
	time.Sleep(1100 * time.Millisecond)
	close(done)
}

func TestEvaluatePreflightErrorMessage_EvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	msg := evaluatePreflightErrorMessage(e, "Error {{ unknown() }}", ctx)
	assert.Equal(t, "Error {{ unknown() }}", msg)
}

func TestExecuteAPIResponse_SuccessEvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	_, err = e.executeAPIResponse(&domain.Resource{
		APIResponse: &domain.APIResponseConfig{
			Success:  "{{ unknown() }}",
			Response: map[string]interface{}{"k": "v"},
		},
	}, ctx)
	require.Error(t, err)
}

func TestEvaluateResponseHeaders_InterfaceMap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)
	headers := e.evaluateResponseHeaders(map[string]interface{}{"X": "v"}, env)
	assert.Equal(t, "v", headers["X"])
}

func TestApplyLLMMetadataToResponse_EmptyFields(t *testing.T) {
	resp := map[string]interface{}{"success": true}
	ctx := &ExecutionContext{LLMMetadata: &LLMMetadata{}}
	covTestEngine().applyLLMMetadataToResponse(resp, ctx)
	_, ok := resp["_meta"]
	assert.False(t, ok)
}

func TestExecuteWithItems_IterationErrorPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{err: errors.New("item failed")})
	e.SetRegistry(reg)
	wf := covWorkflow(&domain.Resource{
		ActionID: "r",
		Items:    []string{`{"id":1}`},
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
	})
	ctx, err := NewExecutionContext(wf)
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	_, err = e.ExecuteWithItems(wf.Resources[0], ctx)
	require.Error(t, err)
}

func TestGraph_SubsetCycleError(t *testing.T) {
	g := NewGraph()
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "a", Requires: []string{"b"}}))
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "b", Requires: []string{"a"}}))
	_, err := g.topologicalSortSubset(map[string]bool{"a": true, "b": true})
	require.Error(t, err)
}

func TestTopologicalSortAllNodes_VisitedContinue(t *testing.T) {
	g := NewGraph()
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "a", Requires: []string{"b"}}))
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "b"}))
	require.NoError(t, g.Build())
	for range 50 {
		_, err := g.topologicalSortAllNodes()
		require.NoError(t, err)
	}
}

func TestGraph_TopologicalSortVisitedAndCycle(t *testing.T) {
	g := NewGraph()
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "a"}))
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "b", Requires: []string{"a"}}))
	visited := map[string]bool{"a": true}
	var result []*domain.Resource
	require.NoError(t, g.TopologicalSortUtil("a", visited, &result))

	g2 := NewGraph()
	require.NoError(t, g2.AddResource(&domain.Resource{ActionID: "a", Requires: []string{"b"}}))
	require.NoError(t, g2.AddResource(&domain.Resource{ActionID: "b", Requires: []string{"a"}}))
	_, err := g2.topologicalSortAllNodes()
	require.Error(t, err)
}

func expressionEvaluator(ctx *ExecutionContext) *expression.Evaluator {
	return expression.NewEvaluator(ctx.API)
}

func TestOutputMapHelpers_NonMapDefaults(t *testing.T) {
	assert.Equal(t, "default", outputMapFieldString("not-map", "stdout", "default"))
	assert.Equal(t, 2, outputMapFieldExitCode("not-map", 2))
	assert.Equal(t, "", outputMapFieldString(map[string]interface{}{"stdout": 1}, "stdout", ""))
	assert.Equal(t, 0, outputMapFieldExitCode(map[string]interface{}{"exitCode": "bad"}, 0))
}

func TestIsFilePattern_UnknownExtension(t *testing.T) {
	ctx := &ExecutionContext{}
	assert.False(t, ctx.IsFilePattern("data.zzzunknown"))
}

func TestIsFilePattern_CoverageBranches(t *testing.T) {
	ctx := &ExecutionContext{}
	assert.True(t, ctx.IsFilePattern("*.png"))
	assert.True(t, ctx.IsFilePattern("dir/file.txt"))
	assert.True(t, ctx.IsFilePattern(`dir\file.txt`))
	assert.True(t, ctx.IsFilePattern("config.yaml"))
	assert.False(t, ctx.IsFilePattern("workflow.name"))
	assert.False(t, ctx.IsFilePattern("plain-name"))
	assert.False(t, ctx.IsFilePattern("file"))
}
