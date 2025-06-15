package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklLLM "github.com/kdeps/schema/gen/llm"
)

func TestExtractToolParams(t *testing.T) {
	logger := logging.GetLogger()

	// Define tool with one required parameter "val"
	req := true
	ptype := "string"
	pdesc := "value"
	params := map[string]*pklLLM.ToolProperties{
		"val": {Required: &req, Type: &ptype, Description: &pdesc},
	}
	name := "echo"
	script := "echo"
	tools := []*pklLLM.Tool{{Name: &name, Script: &script, Parameters: &params}}
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
