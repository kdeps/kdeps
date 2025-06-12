package resolver

import (
	"net/url"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/tmc/langchaingo/llms"
)

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
	params := map[string]*pklLLM.ToolProperties{"v": {Required: &req, Type: &ptype, Description: &pdesc}}
	tools := []*pklLLM.Tool{{Name: &name, Script: &script, Description: &desc, Parameters: &params}}

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
	params := map[string]*pklLLM.ToolProperties{"n": {Required: &req, Type: &ptype, Description: &pdesc}}
	tools := []*pklLLM.Tool{{Name: &name, Script: &script, Description: &desc, Parameters: &params}}
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
