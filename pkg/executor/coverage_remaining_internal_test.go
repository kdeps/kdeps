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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

type covMockExecutor struct {
	result interface{}
	err    error
}

func (m *covMockExecutor) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	return m.result, m.err
}

func covTestEngine() *Engine {
	e := NewEngine(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	return e
}

func covWorkflow(resources ...*domain.Resource) *domain.Workflow {
	return &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "cov-wf",
			Version:        "1.0.0",
			TargetActionID: "r",
		},
		Resources: resources,
	}
}

// ── component_dotenv ────────────────────────────────────────────────────────

func TestParseDotEnvLine_EmptyKey(t *testing.T) {
	_, _, ok := parseDotEnvLine("=value")
	assert.False(t, ok)
}

func TestMergeDotEnv_ExistingNilFromMissingFile(t *testing.T) {
	dir := t.TempDir()
	dotEnvPath := filepath.Join(dir, ".env")
	n, err := mergeDotEnv(&domain.Component{}, dotEnvPath)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestMergeDotEnv_ExistingNil(t *testing.T) {
	dir := t.TempDir()
	dotEnvPath := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(dotEnvPath, []byte("EXISTING=1\n"), 0600))
	comp := &domain.Component{
		Resources: []*domain.Resource{
			{Chat: &domain.ChatConfig{Prompt: `hello {{ env('NEW_VAR') }}`}},
		},
	}
	n, err := mergeDotEnv(comp, dotEnvPath)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestAppendMissingDotEnvVars_OpenError(t *testing.T) {
	err := appendMissingDotEnvVars(filepath.Join(t.TempDir(), "missing", ".env"), []string{"B"})
	require.Error(t, err)
}

// ── component_setup ───────────────────────────────────────────────────────────

func TestPythonManagerFactory_Default(t *testing.T) {
	dir := t.TempDir()
	mgr := pythonManagerFactory(dir)
	assert.Equal(t, dir, mgr.BaseDir)
}

func TestRunComponentSetup_NilSetupWithLegacyPythonPackages(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	comp := &domain.Component{
		Metadata:       domain.ComponentMetadata{Name: "legacy-py"},
		PythonPackages: []string{"requests"}, //nolint:staticcheck
	}
	ctx, err := NewExecutionContext(&domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "t"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{PythonVersion: "3.11"},
		},
	})
	require.NoError(t, err)
	require.NoError(t, e.runComponentSetup(comp, ctx))
}

// ── context session / NewExecutionContext ─────────────────────────────────────

func TestParseSessionTTL_InvalidFallsBack(t *testing.T) {
	got := parseSessionTTL("not-a-duration")
	assert.Equal(t, defaultSessionTTLMinutes*time.Minute, got)
}

func TestParseSessionTTL_Valid(t *testing.T) {
	got := parseSessionTTL("2h")
	assert.Equal(t, 2*time.Hour, got)
}

func TestDefaultSessionDBPath_UserHomeError(t *testing.T) {
	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) {
		return "", errors.New("no home")
	}
	assert.Empty(t, defaultSessionDBPath())
}

func TestCreateSessionStorage_MemoryType(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			Session: &domain.SessionConfig{TTL: "1h", Type: storageTypeMemory},
		},
	}
	storage, err := createSessionStorage(wf, "sess-1")
	require.NoError(t, err)
	require.NotNil(t, storage)
}

func TestCreateSessionStorage_InvalidDBPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	roDir := filepath.Join(t.TempDir(), "ro")
	require.NoError(t, os.Mkdir(roDir, 0555))
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			Session: &domain.SessionConfig{Path: filepath.Join(roDir, "sessions.db")},
		},
	}
	_, err := createSessionStorage(wf, "")
	require.Error(t, err)
}

func TestNewExecutionContext_ConfigLoadError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfgPath := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("bad: [\n"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)

	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "agent"}})
	require.NoError(t, err)
	assert.NotNil(t, ctx.Config)
}

func TestNewExecutionContext_MemoryStorageFailure(t *testing.T) {
	roHome := filepath.Join(t.TempDir(), "rohome")
	require.NoError(t, os.Mkdir(roHome, 0555))
	t.Setenv("HOME", roHome)

	_, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "x"}})
	require.Error(t, err)
}

func TestGetInputProcessorValues(t *testing.T) {
	ctx := &ExecutionContext{
		InputTranscript:  "hello",
		InputMediaFile:   "/media.wav",
		InputFileContent: "file-content",
		InputFilePath:    "/path/file.txt",
	}
	v, ok := ctx.getInputProcessorValue(keyInputTranscript)
	assert.True(t, ok)
	assert.Equal(t, "hello", v)
	v, ok = ctx.getInputProcessorValue(keyInputMedia)
	assert.True(t, ok)
	assert.Equal(t, "/media.wav", v)
	v, ok = ctx.getInputProcessorValue(keyInputFileContent)
	assert.True(t, ok)
	assert.Equal(t, "file-content", v)
	v, ok = ctx.getInputProcessorValue(keyInputFilePath)
	assert.True(t, ok)
	assert.Equal(t, "/path/file.txt", v)
}

func TestGetFilteredStringValue_NilSourceAllowedBlocked(t *testing.T) {
	ctx := &ExecutionContext{allowedParams: []string{"allowed"}}
	_, err := ctx.getFilteredStringValue(nil, "blocked", "query")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowedParams list")
}

func TestHandleMimeTypeSelector_FilterError(t *testing.T) {
	ctx := &ExecutionContext{}
	_, err := ctx.handleMimeTypeSelector(
		[]string{"/nonexistent/file.xyz"},
		"*.xyz",
		[]string{"mime:application/pdf", "first"},
	)
	require.Error(t, err)
}

func TestFilterByMimeType_FallbackExtensionMap(t *testing.T) {
	tmp := t.TempDir()
	txtPath := filepath.Join(tmp, "doc.txt")
	require.NoError(t, os.WriteFile(txtPath, []byte("hello"), 0600))

	ctx := &ExecutionContext{}
	filtered, err := ctx.FilterByMimeType([]string{txtPath}, "text/plain")
	require.NoError(t, err)
	assert.Contains(t, filtered, txtPath)
}

func TestGetSessionID_HeaderBranch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Request = &RequestContext{Headers: map[string]string{"X-Session-ID": "hdr-session"}}
	got, err := ctx.GetSessionID()
	require.NoError(t, err)
	assert.Equal(t, "hdr-session", got)
}

func TestGetSessionID_QueryBranch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Request = &RequestContext{Query: map[string]string{"session_id": "query-session"}}
	got, err := ctx.GetSessionID()
	require.NoError(t, err)
	assert.Equal(t, "query-session", got)
}

func TestGetSessionID_NoSessionStorage(t *testing.T) {
	ctx := &ExecutionContext{}
	got, err := ctx.GetSessionID()
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestGetUploadedFile_FieldNameMatch(t *testing.T) {
	ctx := &ExecutionContext{
		Request: &RequestContext{
			Files: []FileUpload{{FieldName: "cv", Name: "resume.pdf", Path: "/tmp/r.pdf"}},
		},
	}
	f, err := ctx.GetUploadedFile("cv")
	require.NoError(t, err)
	assert.Equal(t, "cv", f.FieldName)
}

func TestExtractLLMResponseFromMap_ViaGetLLMResponse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	ctx.SetOutput("llm-msg", map[string]interface{}{
		"message": map[string]interface{}{"content": "from message"},
	})
	got, err := ctx.GetLLMResponse("llm-msg")
	require.NoError(t, err)
	assert.Equal(t, "from message", got)

	ctx.SetOutput("llm-data", map[string]interface{}{"data": "payload"})
	got, err = ctx.GetLLMResponse("llm-data")
	require.NoError(t, err)
	assert.Equal(t, "payload", got)
}

func TestBuildEvaluatorEnv_ErrorAndItemBranches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Items["item"] = map[string]interface{}{"name": "x"}

	env := ctx.BuildEvaluatorEnv()
	execMap := env["exec"].(map[string]interface{})
	stdoutFn := execMap["stdout"].(func(string) interface{})
	assert.Equal(t, "", stdoutFn("missing"))

	pyMap := env["python"].(map[string]interface{})
	stderrFn := pyMap["stderr"].(func(string) interface{})
	assert.Equal(t, "", stderrFn("missing"))

	itemMap := env["item"].(map[string]interface{})
	valuesFn := itemMap["values"].(func(string) interface{})
	assert.NotNil(t, valuesFn("any"))

	ctx.Items["item"] = "not-a-map"
	env = ctx.BuildEvaluatorEnv()
	itemMap = env["item"].(map[string]interface{})
	valuesFn = itemMap["values"].(func(string) interface{})
	assert.NotNil(t, valuesFn("any"))
}

// ── engine core ─────────────────────────────────────────────────────────────

func TestSetNewExecutionContextForAgency_Error(t *testing.T) {
	roHome := filepath.Join(t.TempDir(), "rohome")
	require.NoError(t, os.Mkdir(roHome, 0555))
	t.Setenv("HOME", roHome)

	e := covTestEngine()
	e.SetNewExecutionContextForAgency(map[string]string{"a": "/tmp"})
	e.newExecutionContext = func(_ *domain.Workflow, _ string) (*ExecutionContext, error) {
		return nil, errors.New("ctx fail")
	}
	_, err := e.newExecutionContext(&domain.Workflow{}, "sess")
	require.Error(t, err)
}

func TestEngine_Execute_InvalidRequestType(t *testing.T) {
	e := covTestEngine()
	_, err := e.Execute(covWorkflow(), "not-a-request-context")
	require.Error(t, err)
}

func TestEngine_Execute_WithSessionIDFactory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{result: "ok"})
	e.SetRegistry(reg)

	wf := covWorkflow(&domain.Resource{
		ActionID: "r",
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
	})
	_, err := e.Execute(wf, &RequestContext{Method: "GET", SessionID: "custom-sess"})
	require.NoError(t, err)
}

type panicExecutor struct{}

func (p *panicExecutor) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	panic("boom")
}

func TestEngine_Execute_PanicRecovery(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&panicExecutor{})
	e.SetRegistry(reg)

	wf := covWorkflow(&domain.Resource{
		ActionID: "r",
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
	})
	require.Panics(t, func() {
		_, _ = e.Execute(wf, nil)
	})
}

func TestPropagateFileInput_NilBody(t *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{}
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{Sources: []string{"file"}},
		},
	}
	e.propagateFileInput(ctx, wf, &RequestContext{Body: nil})
	assert.Empty(t, ctx.InputFileContent)
}

func TestInitWorkflowEvaluator_NilAPI(t *testing.T) {
	e := covTestEngine()
	err := e.initWorkflowEvaluator(&ExecutionContext{})
	require.Error(t, err)
}

func TestApplyResourceValidationFilters_ClearHeadersAndParams(_ *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{}
	e.applyResourceValidationFilters(&domain.Resource{ActionID: "r"}, ctx)
}

func TestValidateResourceInput_NilEvaluator(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Request = &RequestContext{Body: map[string]interface{}{"x": 1}}
	e.evaluator = nil

	res := &domain.Resource{
		ActionID: "r",
		Validations: &domain.ValidationsConfig{
			Expr: []domain.Expression{{Raw: "true"}},
		},
	}
	require.NoError(t, e.validateResourceInput(res, ctx))
}

func TestFormatInputValidationError_NonMultiple(t *testing.T) {
	e := covTestEngine()
	err := e.formatInputValidationError("r", errors.New("plain"))
	require.Error(t, err)
}

func TestFormatInputValidationError_WithValue(t *testing.T) {
	e := covTestEngine()
	mve := &validator.MultipleValidationError{
		Errors: []*domain.ValidationError{{
			Field: "f", Type: "required", Message: "missing", Value: "x",
		}},
	}
	err := e.formatInputValidationError("r", mve)
	require.Error(t, err)
}

func TestFormatCustomValidationError_NonMultiple(t *testing.T) {
	e := covTestEngine()
	err := e.formatCustomValidationError("r", errors.New("plain"))
	require.Error(t, err)
}

func TestRunWorkflowResource_RestrictionSkip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{result: "ok"})
	e.SetRegistry(reg)
	ctx, err := NewExecutionContext(covWorkflow())
	require.NoError(t, err)

	res := &domain.Resource{
		ActionID: "r",
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
		Validations: &domain.ValidationsConfig{
			Methods: []string{"POST"},
		},
	}
	req := &RequestContext{Method: "GET"}
	err = e.runWorkflowResource(covWorkflow(), res, ctx, req)
	require.NoError(t, err)
}

func TestResourceTypeName_BotReplyEmail(t *testing.T) {
	assert.Equal(t, ExecutorBotReply, resourceTypeName(&domain.Resource{BotReply: &domain.BotReplyConfig{}}))
	assert.Equal(t, ExecutorEmail, resourceTypeName(&domain.Resource{Email: &domain.EmailConfig{}}))
}

// ── engine_agent ──────────────────────────────────────────────────────────────

func TestExecuteInlineAgent_NilConfig(t *testing.T) {
	e := covTestEngine()
	_, err := e.executeInlineAgent(nil, &ExecutionContext{})
	require.Error(t, err)
}

func TestParseAgentWorkflow_InvalidFile(t *testing.T) {
	_, err := parseAgentWorkflow("/nonexistent/agent.yaml", "missing")
	require.Error(t, err)
}

func TestEvaluateAgentParams_MapFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	cfg := &domain.AgentCallConfig{Name: "a", Params: map[string]interface{}{
		"x": "{{ unknown() }}",
	}}
	_, err = evaluateAgentParams(e, cfg, ctx)
	require.Error(t, err)
}

// ── engine_component ──────────────────────────────────────────────────────────

func TestValidateComponentCallConfig_Nil(t *testing.T) {
	_, err := validateComponentCallConfig(&domain.Resource{})
	require.Error(t, err)
}

func TestEnsureComponentDotEnv_LoadError(t *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{componentDotEnv: map[string]map[string]string{}}
	comp := &domain.Component{Dir: "/nonexistent/component/dir"}
	e.ensureComponentDotEnv("comp", comp, ctx)
	_, ok := ctx.componentDotEnv["comp"]
	assert.True(t, ok)
}

// ── engine_dispatch / executors ───────────────────────────────────────────────

func TestDispatchPrimaryResource_BrowserBotReplyEmail(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetBrowserExecutor(&covMockExecutor{result: "b"})
	reg.SetBotReplyExecutor(&covMockExecutor{result: "br"})
	reg.SetEmailExecutor(&covMockExecutor{result: "e"})
	e.SetRegistry(reg)
	ctx := &ExecutionContext{Workflow: &domain.Workflow{}}

	_, err := e.dispatchPrimaryResource(&domain.Resource{Browser: &domain.BrowserConfig{}}, ctx)
	require.NoError(t, err)
	_, err = e.dispatchPrimaryResource(&domain.Resource{BotReply: &domain.BotReplyConfig{}}, ctx)
	require.NoError(t, err)
	_, err = e.dispatchPrimaryResource(&domain.Resource{Email: &domain.EmailConfig{}}, ctx)
	require.NoError(t, err)
}

func TestFinalizeResourceResult_APIResponseWithPrimary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	res := &domain.Resource{
		ActionID: "r",
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: map[string]interface{}{"k": "v"},
		},
	}
	out, err := e.finalizeResourceResult(res, ctx, true, map[string]interface{}{"primary": true})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestHandleOnErrorContinue_FallbackEvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	onError := &domain.OnErrorConfig{Fallback: "{{ broken"}
	out, err := e.handleOnErrorContinue(
		&domain.Resource{ActionID: "r"},
		onError,
		ctx,
		errors.New("exec"),
	)
	require.NoError(t, err)
	assert.Equal(t, "{{ broken", out)
}

func TestShouldHandleError_EvalFailureContinues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	onError := &domain.OnErrorConfig{When: []domain.Expression{{Raw: "{{ broken"}}}
	assert.False(t, e.shouldHandleError(onError, errors.New("e"), ctx))
}

func TestExecuteOnErrorExpressions_ParseAndEvalErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	res := &domain.Resource{
		OnError: &domain.OnErrorConfig{Expr: []domain.Expression{{Raw: "{{"}}},
	}
	err = e.executeOnErrorExpressions(res, ctx, errors.New("e"))
	require.Error(t, err)

	res.OnError.Expr = []domain.Expression{{Raw: "{{ unknown() }}"}}
	err = e.executeOnErrorExpressions(res, ctx, errors.New("e"))
	require.Error(t, err)
}

func TestEvaluateFallback_ExpressionEvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	_, err = e.evaluateFallback("{{ unknown() }}", ctx)
	require.Error(t, err)
}

func TestExecuteExecutors_NilConfigBranches(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetHTTPExecutor(&covMockExecutor{})
	reg.SetSQLExecutor(&covMockExecutor{})
	reg.SetPythonExecutor(&covMockExecutor{})
	reg.SetExecExecutor(&covMockExecutor{})
	reg.SetScraperExecutor(&covMockExecutor{})
	reg.SetEmbeddingExecutor(&covMockExecutor{})
	reg.SetSearchLocalExecutor(&covMockExecutor{})
	reg.SetSearchWebExecutor(&covMockExecutor{})
	reg.SetTelephonyExecutor(&covMockExecutor{})
	e.SetRegistry(reg)
	ctx := &ExecutionContext{Workflow: &domain.Workflow{}}
	r := &domain.Resource{ActionID: "r"}

	_, err := e.executeHTTP(r, ctx)
	require.Error(t, err)
	_, err = e.executeSQL(r, ctx)
	require.Error(t, err)
	_, err = e.executePython(r, ctx)
	require.Error(t, err)
	_, err = e.executeExec(r, ctx)
	require.Error(t, err)
	_, err = e.executeScraper(r, ctx)
	require.Error(t, err)
	_, err = e.executeEmbedding(r, ctx)
	require.Error(t, err)
	_, err = e.executeSearchLocal(r, ctx)
	require.Error(t, err)
	_, err = e.executeSearchWeb(r, ctx)
	require.Error(t, err)
	_, err = e.executeTelephony(r, ctx)
	require.Error(t, err)
}

func TestExecuteSingleInlineResource_AgentAndDefault(t *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{Workflow: &domain.Workflow{}}

	_, err := e.executeSingleInlineResource(domain.InlineResource{Agent: &domain.AgentCallConfig{Name: "a"}}, 0, ctx)
	require.Error(t, err)

	_, err = e.executeSingleInlineResource(domain.InlineResource{}, 3, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid resource type")
}

// ── engine_eval ───────────────────────────────────────────────────────────────

func TestBuildEvaluationEnvironment_AccessorErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.Request = &RequestContext{
		Body:    map[string]interface{}{},
		Query:   map[string]string{},
		Headers: map[string]string{},
	}

	env := e.BuildEvaluationEnvironment(ctx)
	httpEnv := env["http"].(map[string]interface{})
	assert.Equal(t, "", httpEnv["responseBody"].(func(string) interface{})("x"))
	assert.Nil(t, httpEnv["responseHeader"].(func(string, string) interface{})("x", "h"))

	telephonyEnv := env["telephony"].(map[string]interface{})
	assert.NotNil(t, telephonyEnv)

	reqEnv := env["request"].(map[string]interface{})
	assert.Nil(t, reqEnv["file"].(func(string) interface{})("missing"))
	assert.Nil(t, reqEnv["filepath"].(func(string) interface{})("missing"))
	assert.Nil(t, reqEnv["filetype"].(func(string) interface{})("missing"))
}

func TestConvertToSlice_ReflectArrayDebug(t *testing.T) {
	e := covTestEngine()
	e.debugMode = true
	slice := e.ConvertToSlice([]string{"a", "b"})
	assert.Equal(t, []interface{}{"a", "b"}, slice)
	none := e.ConvertToSlice(42)
	assert.Nil(t, none)
}

// ── engine_llm ────────────────────────────────────────────────────────────────

type covLLMWithOffline struct {
	result  interface{}
	err     error
	offline bool
}

func (m *covLLMWithOffline) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	return m.result, m.err
}

func (m *covLLMWithOffline) SetToolExecutor(_ interface {
	ExecuteResource(*domain.Resource, *ExecutionContext) (interface{}, error)
}) {
}

func (m *covLLMWithOffline) SetOfflineMode(v bool) { m.offline = v }

func TestExecuteLLM_NilChatAndOfflineEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_OFFLINE_MODE", "true")
	e := covTestEngine()
	reg := NewRegistry()
	llm := &covLLMWithOffline{result: "ok"}
	reg.SetLLMExecutor(llm)
	e.SetRegistry(reg)
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)

	_, err = e.executeLLM(&domain.Resource{ActionID: "r"}, ctx)
	require.Error(t, err)

	_, err = e.executeLLM(&domain.Resource{
		ActionID: "r",
		Chat:     &domain.ChatConfig{Model: "{{ model }}", Prompt: "p", Timeout: "bad"},
	}, ctx)
	require.NoError(t, err)
	assert.True(t, llm.offline)
}

func TestStartLLMTimeoutCountdown_NonDebug(t *testing.T) {
	e := covTestEngine()
	e.debugMode = false
	done := e.startLLMTimeoutCountdown("r", 50*time.Millisecond)
	require.NotNil(t, done)
	close(done)
}

// ── engine_loop ───────────────────────────────────────────────────────────────

func TestExecuteWithItems_FullPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{result: map[string]interface{}{"answer": "ok"}})
	e.SetRegistry(reg)
	e.debugMode = true

	wf := covWorkflow(&domain.Resource{
		ActionID: "r",
		Items:    []string{"[1, 2]"},
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
	})
	ctx, err := NewExecutionContext(wf)
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	out, err := e.ExecuteWithItems(wf.Resources[0], ctx)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestEvaluateResourceItems_Errors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	_, err = e.evaluateResourceItems(&domain.Resource{Items: []string{"{{"}}, ctx)
	require.Error(t, err)

	_, err = e.evaluateResourceItems(&domain.Resource{Items: []string{"{{ unknown() }}"}}, ctx)
	require.Error(t, err)
}

func TestMergeLLMItemIntoResult_NonMapItem(t *testing.T) {
	out := mergeLLMItemIntoResult(
		&domain.Resource{Chat: &domain.ChatConfig{}},
		"plain",
		map[string]interface{}{"x": 1},
	)
	assert.Equal(t, map[string]interface{}{"x": 1}, out)
}

// ── engine_response ───────────────────────────────────────────────────────────

func TestExecuteAPIResponse_NilContext(t *testing.T) {
	e := covTestEngine()
	_, err := e.executeAPIResponse(&domain.Resource{APIResponse: &domain.APIResponseConfig{}}, nil)
	require.Error(t, err)
}

func TestResolveAPIResponseSuccess_InvalidBool(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)

	ok, err := e.resolveAPIResponseSuccess(&domain.APIResponseConfig{Success: "maybe"}, env)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestEvaluateResponseHeaders_StringMap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)

	headers := e.evaluateResponseHeaders(map[string]string{"X": "plain"}, env)
	assert.Equal(t, "plain", headers["X"])
}

func TestApplyLLMMetadataToResponse_NewMeta(t *testing.T) {
	resp := map[string]interface{}{"success": true}
	ctx := &ExecutionContext{LLMMetadata: &LLMMetadata{Model: "m", Backend: "b"}}
	covTestEngine().applyLLMMetadataToResponse(resp, ctx)
	meta := resp["_meta"].(map[string]interface{})
	assert.Equal(t, "m", meta["model"])
}

func TestEvaluateResponseValue_ArrayError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)

	_, err = e.evaluateResponseValue([]interface{}{"{{ broken"}, env)
	require.Error(t, err)
}

// ── engine_preflight ──────────────────────────────────────────────────────────

func TestEvaluatePreflightErrorMessage_Expression(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	msg := evaluatePreflightErrorMessage(e, "Error {{ broken", ctx)
	assert.Equal(t, "Error {{ broken", msg)
}

// ── graph ─────────────────────────────────────────────────────────────────────

func TestGraph_CycleAndSubset(t *testing.T) {
	g := NewGraph()
	r1 := &domain.Resource{ActionID: "a", Requires: []string{"b"}}
	r2 := &domain.Resource{ActionID: "b", Requires: []string{"a"}}
	require.NoError(t, g.AddResource(r1))
	require.NoError(t, g.AddResource(r2))

	_, err := g.TopologicalSort()
	require.Error(t, err)

	g2 := NewGraph()
	require.NoError(t, g2.AddResource(&domain.Resource{ActionID: "only"}))
	order, err := g2.GetExecutionOrder("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	_ = order
}

func TestGraph_TopologicalSortVisitedSkip(t *testing.T) {
	g := NewGraph()
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "a"}))
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "b"}))
	visited := map[string]bool{"a": true}
	var result []*domain.Resource
	require.NoError(t, g.TopologicalSortUtil("a", visited, &result))
}

func TestGraph_TopologicalSortSubsetCycle(t *testing.T) {
	g := NewGraph()
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "a", Requires: []string{"b"}}))
	require.NoError(t, g.AddResource(&domain.Resource{ActionID: "b", Requires: []string{"a"}}))
	_, err := g.GetExecutionOrder("a")
	require.Error(t, err)
}

func TestEnsureNewExecutionContextFactory_Default(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := &Engine{}
	e.ensureNewExecutionContextFactory()
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}}
	ctx, err := e.newExecutionContext(wf, "custom-session")
	require.NoError(t, err)
	assert.NotNil(t, ctx)
	ctx2, err := e.newExecutionContext(wf, "")
	require.NoError(t, err)
	assert.NotNil(t, ctx2)
}

func TestApplyResourceValidationFilters_WithFilters(_ *testing.T) {
	e := covTestEngine()
	ctx := &ExecutionContext{}
	e.applyResourceValidationFilters(&domain.Resource{
		ActionID: "r",
		Validations: &domain.ValidationsConfig{
			Headers: []string{"X-Custom"},
			Params:  []string{"allowed"},
		},
	}, ctx)
}

func TestSetNewExecutionContextForAgency_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	e.SetNewExecutionContextForAgency(map[string]string{"a": "/tmp"})
	ctx, err := e.newExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}}, "")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "/tmp"}, ctx.AgentPaths)
}

func TestBuildEvaluatorEnv_SuccessPaths(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	ctx.SetOutput("py", map[string]interface{}{"stdout": "out", "stderr": "err"})
	ctx.SetOutput("ex", map[string]interface{}{"stdout": "exec-out"})
	env := ctx.BuildEvaluatorEnv()
	py := env["python"].(map[string]interface{})
	assert.Equal(t, "out", py["stdout"].(func(string) interface{})("py"))
	assert.Equal(t, "err", py["stderr"].(func(string) interface{})("py"))
	execMap := env["exec"].(map[string]interface{})
	assert.Equal(t, "exec-out", execMap["stdout"].(func(string) interface{})("ex"))
}

type telephonyAccessor struct{}

func (telephonyAccessor) ToEnvMap() map[string]interface{} {
	return map[string]interface{}{"ok": true}
}

func TestBuildTelephonyAccessorEnv_Impl(t *testing.T) {
	ctx := &ExecutionContext{Items: map[string]interface{}{telephonySessionKey: telephonyAccessor{}}}
	env := buildTelephonyAccessorEnv(ctx)
	assert.True(t, env["ok"].(bool))
}

func TestEvaluateResponseHeaders_EvalError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)
	headers := e.evaluateResponseHeaders("{{ unknown() }}", env)
	assert.Nil(t, headers)
}

func TestResolveAPIResponseSuccess_Error(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	env := e.BuildEvaluationEnvironment(ctx)
	_, err = e.resolveAPIResponseSuccess(&domain.APIResponseConfig{Success: "{{ unknown() }}"}, env)
	require.Error(t, err)
}

func TestEvaluatePreflightErrorMessage_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	msg := evaluatePreflightErrorMessage(e, "Error {{ 'x' }}", ctx)
	assert.Contains(t, msg, "x")
}

func TestEvaluateLLMModel_Expression(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	ctx.Set("modelName", "resolved-model", "memory")
	got := e.evaluateLLMModel("{{ get('modelName') }}", ctx)
	assert.Equal(t, "resolved-model", got)
}

func TestExecuteWithItems_MergeLLMMap(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{result: map[string]interface{}{"answer": "ok"}})
	e.SetRegistry(reg)
	wf := covWorkflow(&domain.Resource{
		ActionID: "r",
		Items:    []string{`{"id": 1}`},
		Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
	})
	ctx, err := NewExecutionContext(wf)
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)
	out, err := e.ExecuteWithItems(wf.Resources[0], ctx)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestAppendMissingDotEnvVars_WriteError(t *testing.T) {
	dir := t.TempDir()
	err := appendMissingDotEnvVars(dir, []string{"B"})
	require.Error(t, err)
}

type failDotEnvWriter struct{}

func (failDotEnvWriter) WriteString(_ string) (int, error) { return 0, errors.New("write fail") }
func (failDotEnvWriter) Close() error                      { return nil }

type failDotEnvCloser struct {
	*os.File
}

func (f *failDotEnvCloser) Close() error { return errors.New("close fail") }

func TestAppendMissingDotEnvVars_WriteAndCloseErrors(t *testing.T) {
	orig := openDotEnvForAppend
	t.Cleanup(func() { openDotEnvForAppend = orig })

	dotEnvPath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(dotEnvPath, []byte("A=1\n"), 0600))

	openDotEnvForAppend = func(_ string) (dotEnvAppendFile, error) {
		return failDotEnvWriter{}, nil
	}
	err := appendMissingDotEnvVars(dotEnvPath, []string{"B"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append to .env")

	f, err := os.OpenFile(dotEnvPath, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	openDotEnvForAppend = func(_ string) (dotEnvAppendFile, error) {
		return &failDotEnvCloser{File: f}, nil
	}
	err = appendMissingDotEnvVars(dotEnvPath, []string{"C"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close .env after append")
}

func TestEngine_Execute_ContextCreationFailure(t *testing.T) {
	e := covTestEngine()
	e.newExecutionContext = func(_ *domain.Workflow, _ string) (*ExecutionContext, error) {
		return nil, errors.New("ctx create failed")
	}
	_, err := e.Execute(
		covWorkflow(&domain.Resource{
			ActionID: "r",
			Chat:     &domain.ChatConfig{Model: "m", Prompt: "p"},
		}),
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create execution context")
}

func TestEngine_Execute_InitEvaluatorFailure(t *testing.T) {
	e := covTestEngine()
	e.newExecutionContext = func(wf *domain.Workflow, _ string) (*ExecutionContext, error) {
		return &ExecutionContext{Workflow: wf}, nil
	}
	_, err := e.Execute(covWorkflow(), nil)
	require.Error(t, err)
}

func TestRunWorkflowResource_SkipCondition(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	ctx, err := NewExecutionContext(covWorkflow())
	require.NoError(t, err)
	e.evaluator = expression.NewEvaluator(ctx.API)

	res := &domain.Resource{
		ActionID: "r",
		Validations: &domain.ValidationsConfig{
			Skip: []domain.Expression{{Raw: "true"}},
		},
		Chat: &domain.ChatConfig{Model: "m", Prompt: "p"},
	}
	err = e.runWorkflowResource(covWorkflow(), res, ctx, nil)
	require.NoError(t, err)
}

func TestSetNewExecutionContextForAgency_CreateError(t *testing.T) {
	roHome := filepath.Join(t.TempDir(), "ro")
	require.NoError(t, os.Mkdir(roHome, 0555))
	t.Setenv("HOME", roHome)

	e := covTestEngine()
	e.SetNewExecutionContextForAgency(map[string]string{"a": "/tmp"})
	_, err := e.newExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}}, "sess")
	require.Error(t, err)
}

// sanity: ensure reflect import used
func TestCovReflectUsed(t *testing.T) {
	assert.Equal(t, reflect.Kind(0), reflect.Invalid)
	_ = fmt.Sprintf
}

func TestIsNilConfig_TypedNil(t *testing.T) {
	var p *domain.Resource
	var iface any = p
	assert.True(t, isNilConfig(iface))
	assert.True(t, isNilConfig(nil))
	assert.False(t, isNilConfig("x"))
}

func TestClearLoopContext_Nil(_ *testing.T) {
	clearLoopContext(nil)
}

func TestClearLoopContext_ClearsKeys(t *testing.T) {
	ctx := &ExecutionContext{Items: map[string]interface{}{
		loopKeyIndex: 1, loopKeyCount: 2, loopKeyResults: []interface{}{"x"},
	}}
	clearLoopContext(ctx)
	assert.NotContains(t, ctx.Items, loopKeyIndex)
	assert.NotContains(t, ctx.Items, loopKeyCount)
	assert.NotContains(t, ctx.Items, loopKeyResults)
}

func TestUploadedFileByIndex_InvalidSyntax(t *testing.T) {
	ctx := &ExecutionContext{
		Request: &RequestContext{
			Files: []FileUpload{{Name: "a.txt", Path: "/a"}},
		},
	}
	assert.Nil(t, ctx.uploadedFileByIndex("plain"))
	assert.Nil(t, ctx.uploadedFileByIndex("[0]"))
	assert.Nil(t, ctx.uploadedFileByIndex("file[-1]"))
}
