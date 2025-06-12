package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
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
	scenario := []*pklLLM.MultiChat{{
		Role:   &scenarioRole,
		Prompt: &scenarioPrompt,
	}}

	// Files
	files := []string{ec(original["filePath"])}

	// Tool parameters
	paramType := original["paramType"]
	paramDesc := original["paramDescription"]
	req := true
	params := map[string]*pklLLM.ToolProperties{
		"value": {
			Type:        &paramType,
			Description: &paramDesc,
			Required:    &req,
		},
	}

	toolName := original["toolName"]
	toolScript := original["toolScript"]
	toolDesc := original["toolDescription"]
	tools := []*pklLLM.Tool{{
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
