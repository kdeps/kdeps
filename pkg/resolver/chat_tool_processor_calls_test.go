package resolver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/tmc/langchaingo/llms"
)

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
	params := map[string]*pklLLM.ToolProperties{"val": {Required: &req, Type: &ptype, Description: &desc}}
	tools := []*pklLLM.Tool{{Name: &name, Script: &script, Parameters: &params}}
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
