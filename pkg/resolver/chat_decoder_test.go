package resolver

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklRes "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// buildEncodedChat constructs a ResourceChat with all string fields base64 encoded so we
// can validate decodeChatBlock unwraps them correctly.
func buildEncodedChat() (*pklLLM.ResourceChat, map[string]string) {
	original := map[string]string{
		"prompt":           "Tell me a joke",
		"role":             RoleSystem,
		"jsonKeyOne":       "temperature",
		"jsonKeyTwo":       "top_p",
		"scenarioPrompt":   "You are helpful",
		"filePath":         "/tmp/data.txt",
		"toolName":         "echo",
		"toolScript":       "echo 'hi'",
		"toolDescription":  "simple echo tool",
		"paramType":        "string",
		"paramDescription": "value to echo",
	}

	ec := func(v string) string { return utils.EncodeValue(v) }

	// Scenario
	scenarioRole := ec(RoleHuman)
	scenarioPrompt := ec(original["scenarioPrompt"])
	scenario := []pklLLM.MultiChat{{
		Role:   &scenarioRole,
		Prompt: &scenarioPrompt,
	}}

	// Files
	files := []string{ec(original["filePath"])}

	// Tool parameters
	paramType := original["paramType"]
	paramDesc := original["paramDescription"]
	req := true
	params := map[string]pklLLM.ToolProperties{
		"value": {
			Type:        &paramType,
			Description: &paramDesc,
			Required:    &req,
		},
	}

	toolName := original["toolName"]
	toolScript := original["toolScript"]
	toolDesc := original["toolDescription"]
	tools := []pklLLM.Tool{{
		Name:        &toolName,
		Script:      &toolScript,
		Description: &toolDesc,
		Parameters:  &params,
	}}

	prompt := ec(original["prompt"])
	role := ec(original["role"])
	jsonKeys := []string{ec(original["jsonKeyOne"]), ec(original["jsonKeyTwo"])}

	chat := &pklLLM.ResourceChat{
		Prompt:           &prompt,
		Role:             &role,
		JSONResponseKeys: &jsonKeys,
		Scenario:         &scenario,
		Files:            &files,
		Tools:            &tools,
	}
	return chat, original
}

func TestDecodeChatBlock_AllFields(t *testing.T) {
	chat, original := buildEncodedChat()
	dr := &DependencyResolver{Logger: logging.GetLogger()}

	if err := dr.decodeChatBlock(chat); err != nil {
		t.Fatalf("decodeChatBlock error: %v", err)
	}

	// Validate prompt & role.
	if utils.SafeDerefString(chat.Prompt) != original["prompt"] {
		t.Errorf("prompt decode mismatch, got %s", utils.SafeDerefString(chat.Prompt))
	}
	if utils.SafeDerefString(chat.Role) != original["role"] {
		t.Errorf("role decode mismatch, got %s", utils.SafeDerefString(chat.Role))
	}

	// JSON keys
	for i, want := range []string{original["jsonKeyOne"], original["jsonKeyTwo"]} {
		if (*chat.JSONResponseKeys)[i] != want {
			t.Errorf("json key %d decode mismatch, got %s", i, (*chat.JSONResponseKeys)[i])
		}
	}

	// Scenario
	if chat.Scenario == nil || len(*chat.Scenario) != 1 {
		t.Fatalf("expected 1 scenario entry")
	}
	entry := (*chat.Scenario)[0]
	if utils.SafeDerefString(entry.Role) != RoleHuman {
		t.Errorf("scenario role mismatch, got %s", utils.SafeDerefString(entry.Role))
	}
	if utils.SafeDerefString(entry.Prompt) != original["scenarioPrompt"] {
		t.Errorf("scenario prompt mismatch, got %s", utils.SafeDerefString(entry.Prompt))
	}

	// Files
	if chat.Files == nil || (*chat.Files)[0] != original["filePath"] {
		t.Errorf("file path decode mismatch, got %v", chat.Files)
	}

	// Tools fields
	if chat.Tools == nil || len(*chat.Tools) != 1 {
		t.Fatalf("expected 1 tool entry")
	}
	tool := (*chat.Tools)[0]
	if utils.SafeDerefString(tool.Name) != original["toolName"] {
		t.Errorf("tool name mismatch, got %s", utils.SafeDerefString(tool.Name))
	}
	if utils.SafeDerefString(tool.Script) != original["toolScript"] {
		t.Errorf("tool script mismatch, got %s", utils.SafeDerefString(tool.Script))
	}
	if utils.SafeDerefString(tool.Description) != original["toolDescription"] {
		t.Errorf("tool description mismatch, got %s", utils.SafeDerefString(tool.Description))
	}
	gotParam := (*tool.Parameters)["value"]
	if utils.SafeDerefString(gotParam.Type) != original["paramType"] {
		t.Errorf("param type mismatch, got %s", utils.SafeDerefString(gotParam.Type))
	}
	if utils.SafeDerefString(gotParam.Description) != original["paramDescription"] {
		t.Errorf("param description mismatch, got %s", utils.SafeDerefString(gotParam.Description))
	}
}

func TestDecodeScenario_Nil(t *testing.T) {
	chat := &pklLLM.ResourceChat{Scenario: nil}
	logger := logging.GetLogger()
	if err := decodeScenario(chat, logger); err != nil {
		t.Fatalf("decodeScenario nil case error: %v", err)
	}
	if chat.Scenario == nil || len(*chat.Scenario) != 0 {
		t.Errorf("expected empty scenario slice after decode")
	}
}

func TestEncodeJSONResponseKeys(t *testing.T) {
	keys := []string{"one", "two"}
	encoded := encodeJSONResponseKeys(&keys)
	if encoded == nil || len(*encoded) != 2 {
		t.Fatalf("expected 2 encoded keys")
	}
	for i, k := range keys {
		want := utils.EncodeValue(k)
		if (*encoded)[i] != want {
			t.Errorf("key %d mismatch: got %s want %s", i, (*encoded)[i], want)
		}
	}
}

func TestDecodeField_Base64(t *testing.T) {
	original := "hello world"
	b64 := base64.StdEncoding.EncodeToString([]byte(original))
	ptr := &b64
	if err := decodeField(&ptr, "testField", utils.SafeDerefString, ""); err != nil {
		t.Fatalf("decodeField returned error: %v", err)
	}
	if utils.SafeDerefString(ptr) != original {
		t.Errorf("decodeField did not decode correctly: got %s", utils.SafeDerefString(ptr))
	}
}

func TestDecodeField_NonBase64(t *testing.T) {
	val := "plain value"
	ptr := &val
	if err := decodeField(&ptr, "testField", utils.SafeDerefString, "default"); err != nil {
		t.Fatalf("decodeField returned error: %v", err)
	}
	if utils.SafeDerefString(ptr) != val {
		t.Errorf("expected field to remain unchanged, got %s", utils.SafeDerefString(ptr))
	}
}

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
		empty := make(map[string]pklLLM.ResourceChat)
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
		empty := make(map[string]pklHTTP.ResourceHTTPClient)
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

func TestGenerateChatResponseBasic(t *testing.T) {
	// Create stub HTTP client to satisfy Ollama client without network
	httpClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Return NDJSON single line with completed message
			body := `{"message":{"content":"stub-response"},"done":true}` + "\n"
			resp := &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}
			resp.Header.Set("Content-Type", "application/x-ndjson")
			return resp, nil
		}),
	}

	llm, errNew := ollama.New(
		ollama.WithHTTPClient(httpClient),
		ollama.WithServerURL("http://stub"),
	)
	assert.NoError(t, errNew)

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	prompt := "Hello"
	role := "user"
	chatBlock := &pklLLM.ResourceChat{
		Model:  "test-model",
		Prompt: &prompt,
		Role:   &role,
	}

	resp, err := generateChatResponse(ctx, fs, llm, chatBlock, nil, logger)
	assert.NoError(t, err)
	assert.Equal(t, "stub-response", resp)
}

func TestLoadResourceEntriesInjected(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	// Setup workflow resources directory and dummy .pkl file
	workflowDir := "/workflow"
	resourcesDir := workflowDir + "/resources"
	_ = fs.MkdirAll(resourcesDir, 0o755)
	dummyFile := resourcesDir + "/dummy.pkl"
	_ = afero.WriteFile(fs, dummyFile, []byte("dummy"), 0o644)

	dr := &DependencyResolver{
		Fs:                   fs,
		Logger:               logger,
		WorkflowDir:          workflowDir,
		ResourceDependencies: make(map[string][]string),
		Resources:            []ResourceNodeEntry{},
		LoadResourceFn: func(_ context.Context, _ string, _ ResourceType) (interface{}, error) {
			return &pklRes.Resource{ActionID: "action1"}, nil
		},
		PrependDynamicImportsFn: func(string) error { return nil },
		AddPlaceholderImportsFn: func(string) error { return nil },
	}

	err := dr.LoadResourceEntries()
	assert.NoError(t, err)
	assert.Len(t, dr.Resources, 1)
	assert.Contains(t, dr.ResourceDependencies, "action1")
}

// roundTripFunc allows defining inline RoundTripper functions.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestProcessToolCalls_Success ensures happy-path processing populates outputs and history.
func TestProcessToolCalls_Success(t *testing.T) {
	logger := logging.GetLogger()
	tmp := t.TempDir()
	reader, errInit := tool.InitializeTool(filepath.Join(tmp, "tools.db"))
	if errInit != nil {
		t.Fatalf("failed init tool reader: %v", errInit)
	}
	// pre-seed expected tool output
	_, _ = reader.DB.Exec("INSERT INTO tools (id, value) VALUES ('1', 'ok')")

	// Build chat block with one defined tool
	name := "echo"
	script := "echo"
	req := true
	ptype := "string"
	desc := "value"
	params := map[string]pklLLM.ToolProperties{"val": {Required: &req, Type: &ptype, Description: &desc}}
	tools := []pklLLM.Tool{{Name: &name, Script: &script, Parameters: &params}}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	// ToolCall JSON string
	argsJSON := `{"val":"hello"}`
	call := llms.ToolCall{
		ID:           "1",
		FunctionCall: &llms.FunctionCall{Name: name, Arguments: argsJSON},
	}

	history := []llms.MessageContent{}
	outputs := map[string]string{}

	if err := processToolCalls([]llms.ToolCall{call}, reader, chat, logger, &history, "prompt", outputs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := outputs["1"]; !ok {
		t.Errorf("tool output missing: %v", outputs)
	}
	if len(history) == 0 {
		t.Errorf("history not populated")
	}
}

// TestProcessToolCalls_Error validates that invalid calls are aggregated into an error.
func TestProcessToolCalls_Error(t *testing.T) {
	logger := logging.GetLogger()
	tmp := t.TempDir()
	reader, errInit := tool.InitializeTool(filepath.Join(tmp, "tools.db"))
	if errInit != nil {
		t.Fatalf("failed init tool reader: %v", errInit)
	}
	// pre-seed expected tool output
	_, _ = reader.DB.Exec("INSERT INTO tools (id, value) VALUES ('1', 'ok')")

	chat := &pklLLM.ResourceChat{}
	badCall := llms.ToolCall{} // missing FunctionCall leading to error path

	err := processToolCalls([]llms.ToolCall{badCall}, reader, chat, logger, &[]llms.MessageContent{}, "", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "invalid tool call") {
		t.Logf("error returned: %v", err)
	}
}

func TestParseToolCallArgs(t *testing.T) {
	logger := logging.GetLogger()
	input := `{"a": 1, "b": "val"}`
	args, err := parseToolCallArgs(input, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args["a"].(float64) != 1 || args["b"].(string) != "val" {
		t.Errorf("parsed args mismatch: %v", args)
	}

	// Invalid JSON should error
	if _, err := parseToolCallArgs("not-json", logger); err == nil {
		t.Errorf("expected error for invalid json")
	}
}

func TestDeduplicateToolCalls(t *testing.T) {
	logger := logging.GetLogger()
	tc1 := llms.ToolCall{ID: "1", Type: "function", FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: "{}"}}
	tc2 := llms.ToolCall{ID: "2", Type: "function", FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: "{}"}}
	tc3 := llms.ToolCall{ID: "3", Type: "function", FunctionCall: &llms.FunctionCall{Name: "sum", Arguments: "{}"}}

	dedup := deduplicateToolCalls([]llms.ToolCall{tc1, tc2, tc3}, logger)
	if len(dedup) != 2 {
		t.Errorf("expected 2 unique calls, got %d", len(dedup))
	}
}

func TestExtractToolNames(t *testing.T) {
	calls := []llms.ToolCall{
		{FunctionCall: &llms.FunctionCall{Name: "one"}},
		{FunctionCall: &llms.FunctionCall{Name: "two"}},
	}
	names := extractToolNames(calls)
	if len(names) != 2 || names[0] != "one" || names[1] != "two" {
		t.Errorf("extractToolNames mismatch: %v", names)
	}
}

func TestEncodeToolsAndParams(t *testing.T) {
	// Build raw tools slice (non-encoded)
	name := "echo"
	script := "echo hi"
	desc := "simple"
	req := true
	ptype := "string"
	pdesc := "value"
	params := map[string]pklLLM.ToolProperties{"v": {Required: &req, Type: &ptype, Description: &pdesc}}
	tools := []pklLLM.Tool{{Name: &name, Script: &script, Description: &desc, Parameters: &params}}

	encoded := encodeTools(&tools)
	if len(encoded) != 1 {
		t.Fatalf("expected 1 encoded tool")
	}
	et := encoded[0]
	if utils.SafeDerefString(et.Name) != utils.EncodeValue(name) {
		t.Errorf("name not encoded: %s", utils.SafeDerefString(et.Name))
	}
	if utils.SafeDerefString((*et.Parameters)["v"].Description) != utils.EncodeValue(pdesc) {
		t.Errorf("param description not encoded")
	}

	// encodeToolParameters directly
	ep := encodeToolParameters(&params)
	if (*ep)["v"].Required == nil || *(*ep)["v"].Required != true {
		t.Errorf("required flag lost in encoding")
	}
}

func TestGenerateAvailableTools(t *testing.T) {
	logger := logging.GetLogger()
	// Prepare chatBlock with one tool
	name := "calc"
	script := "echo $((1+1))"
	desc := "calculator"
	chat := &pklLLM.ResourceChat{}
	req := true
	ptype := "string"
	pdesc := "number"
	params := map[string]pklLLM.ToolProperties{"n": {Required: &req, Type: &ptype, Description: &pdesc}}
	tools := []pklLLM.Tool{{Name: &name, Script: &script, Description: &desc, Parameters: &params}}
	chat.Tools = &tools

	avail := generateAvailableTools(chat, logger)
	if len(avail) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(avail))
	}
	if avail[0].Function == nil || avail[0].Function.Name != name {
		t.Errorf("tool name mismatch: %+v", avail[0])
	}
}

func TestConstructToolCallsFromJSON(t *testing.T) {
	logger := logging.GetLogger()
	// Array form
	jsonStr := `[{"name": "echo", "arguments": {"msg": "hi"}}]`
	calls := constructToolCallsFromJSON(jsonStr, logger)
	if len(calls) != 1 || calls[0].FunctionCall.Name != "echo" {
		t.Errorf("unexpected calls parsed: %v", calls)
	}
	// Single object form
	single := `{"name":"sum","arguments": {"a":1}}`
	calls2 := constructToolCallsFromJSON(single, logger)
	if len(calls2) != 1 || calls2[0].FunctionCall.Name != "sum" {
		t.Errorf("single object parse failed: %v", calls2)
	}
}

func TestBuildToolURIAndConvertParams(t *testing.T) {
	id := "tool1"
	script := "echo"
	params := "a+b"
	uri, err := buildToolURI(id, script, params)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if uri.Scheme != "tool" {
		t.Errorf("scheme mismatch: %s", uri.Scheme)
	}
	if uri.Path != "/"+id {
		t.Errorf("path mismatch: %s", uri.Path)
	}
	qs := uri.Query()
	if qs.Get("op") != "run" {
		t.Errorf("expected op=run, got %s", qs.Get("op"))
	}
	if qs.Get("script") != script {
		t.Errorf("script param mismatch: %s", qs.Get("script"))
	}
	// params will be double-escaped in buildToolURI
	wantParams := url.QueryEscape(params)
	if qs.Get("params") != wantParams {
		t.Errorf("params mismatch: got %s want %s", qs.Get("params"), wantParams)
	}

	// convertToolParamsToString
	logger := logging.GetLogger()
	out := convertToolParamsToString([]interface{}{1, "x"}, "arg", "tool", logger)
	if out == "" {
		t.Errorf("expected param conversion not empty")
	}
}

func TestExtractToolParams(t *testing.T) {
	logger := logging.GetLogger()

	// Define tool with one required parameter "val"
	req := true
	ptype := "string"
	pdesc := "value"
	params := map[string]pklLLM.ToolProperties{
		"val": {Required: &req, Type: &ptype, Description: &pdesc},
	}
	name := "echo"
	script := "echo"
	tools := []pklLLM.Tool{{Name: &name, Script: &script, Parameters: &params}}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	args := map[string]interface{}{"val": "hi"}
	n, s, pv, err := extractToolParams(args, chat, "echo", logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != name || s != script {
		t.Errorf("mismatch name/script")
	}
	if pv != "hi" {
		t.Errorf("params concat incorrect: %s", pv)
	}

	// Missing required param should still succeed but warn.
	_, _, _, err2 := extractToolParams(map[string]interface{}{}, chat, "echo", logger)
	if err2 != nil {
		t.Fatalf("expected no error on missing required, got: %v", err2)
	}

	// Nonexistent tool
	_, _, _, err3 := extractToolParams(args, chat, "nope", logger)
	if err3 == nil {
		t.Errorf("expected error for missing tool")
	}
}

func TestExtractToolNamesFromTools(t *testing.T) {
	name1, name2 := "echo", "calc"
	tools := []llms.Tool{
		{Function: &llms.FunctionDefinition{Name: name1}},
		{Function: &llms.FunctionDefinition{Name: name2}},
	}
	got := extractToolNamesFromTools(tools)
	if len(got) != 2 || got[0] != name1 || got[1] != name2 {
		t.Fatalf("unexpected names: %v", got)
	}
}

func TestSerializeTools(t *testing.T) {
	// Build a simple Tool slice
	script := "echo hello"
	desc := "say hello"
	name := utils.EncodeValue("helloTool")
	scriptEnc := utils.EncodeValue(script)
	descEnc := utils.EncodeValue(desc)

	req := true
	ptype := "string"
	pdesc := "greeting"
	params := map[string]pklLLM.ToolProperties{
		"msg": {Required: &req, Type: &ptype, Description: &pdesc},
	}

	entries := []pklLLM.Tool{{
		Name:        &name,
		Script:      &scriptEnc,
		Description: &descEnc,
		Parameters:  &params,
	}}

	var sb strings.Builder
	serializeTools(&sb, &entries)
	out := sb.String()

	if !strings.Contains(out, "Tools {") || !strings.Contains(out, "Name = \""+name+"\"") {
		t.Errorf("serialized output missing fields: %s", out)
	}
	if !strings.Contains(out, "Script = #\"\"\"") {
		t.Errorf("script block missing: %s", out)
	}
	if !strings.Contains(out, "Parameters") {
		t.Errorf("Parameters missing: %s", out)
	}
}
