package resolver

import (
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/tmc/langchaingo/llms"
)

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
	params := map[string]*pklLLM.ToolProperties{
		"msg": {Required: &req, Type: &ptype, Description: &pdesc},
	}

	entries := []*pklLLM.Tool{{
		Name:        &name,
		Script:      &scriptEnc,
		Description: &descEnc,
		Parameters:  &params,
	}}

	var sb strings.Builder
	serializeTools(&sb, &entries)
	out := sb.String()

	if !strings.Contains(out, "tools {") || !strings.Contains(out, "name = \""+name+"\"") {
		t.Errorf("serialized output missing fields: %s", out)
	}
	if !strings.Contains(out, "script = #\"\"\"") {
		t.Errorf("script block missing: %s", out)
	}
	if !strings.Contains(out, "parameters") {
		t.Errorf("parameters missing: %s", out)
	}
}
