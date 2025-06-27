package resolver

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

func TestConstructToolCallsFromJSON(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "invalid json",
			input:    "{invalid json}",
			expected: 0,
		},
		{
			name:     "single object",
			input:    `{"name":"echo","arguments":{"msg":"hello"}}`,
			expected: 1,
		},
		{
			name:     "array of tool calls",
			input:    `[{"name":"echo","arguments":{"msg":"hello"}},{"name":"sum","arguments":{"a":1,"b":2}}]`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConstructToolCallsFromJSON(tt.input, logger)
			assert.Len(t, result, tt.expected)

			if tt.expected > 0 && len(result) > 0 {
				assert.NotNil(t, result[0].FunctionCall)
				assert.NotEmpty(t, result[0].FunctionCall.Name)
			}
		})
	}
}

func TestExtractToolParams(t *testing.T) {
	logger := logging.NewTestLogger()

	// Create test tool with parameters
	req := true
	ptype := "string"
	pdesc := "test parameter"
	params := map[string]*pklLLM.ToolProperties{
		"msg": {
			Required:    &req,
			Type:        &ptype,
			Description: &pdesc,
		},
	}

	toolName := "echo"
	script := "echo $msg"
	tool := &pklLLM.Tool{
		Name:       &toolName,
		Script:     &script,
		Parameters: &params,
	}
	tools := []*pklLLM.Tool{tool}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	t.Run("valid tool with args", func(t *testing.T) {
		args := map[string]interface{}{"msg": "hello"}
		name, gotScript, paramsStr, err := ExtractToolParams(args, chat, "echo", logger)

		assert.NoError(t, err)
		assert.Equal(t, "echo", name)
		assert.Equal(t, "echo $msg", gotScript)
		assert.Equal(t, "hello", paramsStr)
	})

	t.Run("missing required parameter", func(t *testing.T) {
		args := map[string]interface{}{}
		name, gotScript, paramsStr, err := ExtractToolParams(args, chat, "echo", logger)

		// Function should not error on missing required params, just warn
		assert.NoError(t, err)
		assert.Equal(t, "echo", name)
		assert.Equal(t, "echo $msg", gotScript)
		assert.Equal(t, "", paramsStr)
	})

	t.Run("nonexistent tool", func(t *testing.T) {
		args := map[string]interface{}{"msg": "hello"}
		_, _, _, err := ExtractToolParams(args, chat, "nonexistent", logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool nonexistent not found")
	})
}

func TestBuildToolURI(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		script   string
		params   string
		hasError bool
	}{
		{
			name:     "valid uri",
			id:       "test-id",
			script:   "echo hello",
			params:   "world",
			hasError: false,
		},
		{
			name:     "empty parameters",
			id:       "test-id",
			script:   "echo hello",
			params:   "",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := BuildToolURI(tt.id, tt.script, tt.params)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, uri)
				assert.Equal(t, "tool", uri.Scheme)
				// The script gets URL encoded, so check for encoded version
				assert.Contains(t, uri.String(), "echo+hello")
			}
		})
	}
}

func TestFormatToolParameters(t *testing.T) {
	// Create test tool with function definition
	tool := llms.Tool{
		Function: &llms.FunctionDefinition{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message to process",
					},
				},
			},
		},
	}

	var sb strings.Builder
	FormatToolParameters(tool, &sb)

	result := sb.String()
	assert.NotEmpty(t, result)
	// Should contain formatted parameter information
	assert.Contains(t, result, "message")
}

func TestProcessToolCalls(t *testing.T) {
	logger := logging.NewTestLogger()

	// Setup temporary tool database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "tools.db")
	reader, err := tool.InitializeTool(dbPath)
	require.NoError(t, err)

	// Seed tool output
	_, err = reader.DB.Exec("INSERT INTO tools (id, value) VALUES ('test-id', 'tool output')")
	require.NoError(t, err)

	// Create test chat block with tool
	name := "echo"
	script := "echo"
	req := true
	ptype := "string"
	desc := "test param"
	params := map[string]*pklLLM.ToolProperties{
		"msg": {Required: &req, Type: &ptype, Description: &desc},
	}
	tools := []*pklLLM.Tool{{Name: &name, Script: &script, Parameters: &params}}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	t.Run("successful tool call", func(t *testing.T) {
		call := llms.ToolCall{
			ID: "test-id",
			FunctionCall: &llms.FunctionCall{
				Name:      "echo",
				Arguments: `{"msg":"hello"}`,
			},
		}

		history := []llms.MessageContent{}
		outputs := map[string]string{}

		err := ProcessToolCalls([]llms.ToolCall{call}, reader, chat, logger, &history, "test prompt", outputs)
		assert.NoError(t, err)
		assert.Contains(t, outputs, "test-id")
		assert.NotEmpty(t, history)
	})

	t.Run("invalid tool call", func(t *testing.T) {
		call := llms.ToolCall{} // Missing FunctionCall

		history := []llms.MessageContent{}
		outputs := map[string]string{}

		err := ProcessToolCalls([]llms.ToolCall{call}, reader, chat, logger, &history, "test prompt", outputs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tool call")
	})
}

func TestParseToolCallArgs(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("valid json", func(t *testing.T) {
		input := `{"name":"value","number":42}`
		args, err := ParseToolCallArgs(input, logger)

		assert.NoError(t, err)
		assert.Equal(t, "value", args["name"])
		assert.Equal(t, float64(42), args["number"]) // JSON numbers become float64
	})

	t.Run("invalid json", func(t *testing.T) {
		input := `{invalid json}`
		_, err := ParseToolCallArgs(input, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tool arguments")
	})
}

func TestEncodeTools(t *testing.T) {
	// Create test tools
	name := "test_tool"
	script := "echo hello"
	desc := "test description"
	req := true
	ptype := "string"
	pdesc := "param description"

	params := map[string]*pklLLM.ToolProperties{
		"param1": {
			Required:    &req,
			Type:        &ptype,
			Description: &pdesc,
		},
	}

	tool := &pklLLM.Tool{
		Name:        &name,
		Script:      &script,
		Description: &desc,
		Parameters:  &params,
	}
	tools := []*pklLLM.Tool{tool}

	encoded := EncodeTools(&tools)

	assert.Len(t, encoded, 1)
	assert.Equal(t, utils.EncodeValue(name), *encoded[0].Name)
	assert.Equal(t, utils.EncodeValue(script), *encoded[0].Script)
	assert.Equal(t, utils.EncodeValue(desc), *encoded[0].Description)
	assert.NotNil(t, encoded[0].Parameters)
}

func TestEncodeToolParameters(t *testing.T) {
	req := true
	ptype := "string"
	desc := "test parameter"

	params := map[string]*pklLLM.ToolProperties{
		"param1": {
			Required:    &req,
			Type:        &ptype,
			Description: &desc,
		},
	}

	encoded := EncodeToolParameters(&params)

	assert.NotNil(t, encoded)
	assert.Contains(t, *encoded, "param1")

	param := (*encoded)["param1"]
	assert.Equal(t, utils.EncodeValue(ptype), *param.Type)
	assert.Equal(t, utils.EncodeValue(desc), *param.Description)
	assert.Equal(t, req, *param.Required)
}

func TestExtractToolNames(t *testing.T) {
	calls := []llms.ToolCall{
		{FunctionCall: &llms.FunctionCall{Name: "tool1"}},
		{FunctionCall: &llms.FunctionCall{Name: "tool2"}},
		{FunctionCall: nil}, // Should be skipped
	}

	names := ExtractToolNames(calls)

	assert.Len(t, names, 2)
	assert.Contains(t, names, "tool1")
	assert.Contains(t, names, "tool2")
}

func TestExtractToolNamesFromTools(t *testing.T) {
	tools := []llms.Tool{
		{Function: &llms.FunctionDefinition{Name: "echo"}},
		{Function: &llms.FunctionDefinition{Name: "sum"}},
		{Function: nil}, // Should be skipped
	}

	names := ExtractToolNamesFromTools(tools)

	assert.Len(t, names, 2)
	assert.Contains(t, names, "echo")
	assert.Contains(t, names, "sum")
}

func TestDeduplicateToolCalls(t *testing.T) {
	logger := logging.NewTestLogger()

	calls := []llms.ToolCall{
		{FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"msg":"hello"}`}},
		{FunctionCall: &llms.FunctionCall{Name: "echo", Arguments: `{"msg":"hello"}`}}, // Duplicate
		{FunctionCall: &llms.FunctionCall{Name: "sum", Arguments: `{"a":1,"b":2}`}},
		{FunctionCall: nil}, // Should be skipped
	}

	dedup := DeduplicateToolCalls(calls, logger)

	assert.Len(t, dedup, 2) // One duplicate removed, one nil removed

	names := ExtractToolNames(dedup)
	assert.Contains(t, names, "echo")
	assert.Contains(t, names, "sum")
}

func TestConvertToolParamsToString(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string value", "hello", "hello"},
		{"float value", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil value", nil, ""},
		{"object value", map[string]int{"x": 1}, `{"x":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToolParamsToString(tt.value, "param", "tool", logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSerializeTools(t *testing.T) {
	// Create test tools
	name := "test_tool"
	script := "echo hello"
	desc := "test description"

	tool := &pklLLM.Tool{
		Name:        &name,
		Script:      &script,
		Description: &desc,
	}
	tools := []*pklLLM.Tool{tool}

	var sb strings.Builder
	SerializeTools(&sb, &tools)

	result := sb.String()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "test_tool")
}

func TestDecodeToolEntry(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("valid tool entry", func(t *testing.T) {
		name := "test_tool"
		script := "echo hello"
		desc := "test description"
		req := true
		ptype := "string"
		pdesc := "param desc"

		params := map[string]*pklLLM.ToolProperties{
			"param1": {
				Required:    &req,
				Type:        &ptype,
				Description: &pdesc,
			},
		}

		entry := &pklLLM.Tool{
			Name:        &name,
			Script:      &script,
			Description: &desc,
			Parameters:  &params,
		}

		decoded, err := DecodeToolEntry(entry, 0, logger)

		assert.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Equal(t, name, utils.SafeDerefString(decoded.Name))
		assert.Equal(t, script, utils.SafeDerefString(decoded.Script))
		assert.Equal(t, desc, utils.SafeDerefString(decoded.Description))
		assert.NotNil(t, decoded.Parameters)
	})

	t.Run("nil tool entry", func(t *testing.T) {
		_, err := DecodeToolEntry(nil, 0, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool entry at index 0 is nil")
	})

	t.Run("tool entry with nil fields", func(t *testing.T) {
		entry := &pklLLM.Tool{
			Name:        nil,
			Script:      nil,
			Description: nil,
			Parameters:  nil,
		}

		decoded, err := DecodeToolEntry(entry, 0, logger)

		assert.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Equal(t, "", utils.SafeDerefString(decoded.Name))
		assert.Equal(t, "", utils.SafeDerefString(decoded.Script))
		assert.Equal(t, "", utils.SafeDerefString(decoded.Description))
		assert.NotNil(t, decoded.Parameters)
		assert.Empty(t, *decoded.Parameters)
	})
}

func TestDecodeToolParameters(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("valid parameters", func(t *testing.T) {
		req := true
		ptype := "string"
		desc := "test parameter"

		params := map[string]*pklLLM.ToolProperties{
			"param1": {
				Required:    &req,
				Type:        &ptype,
				Description: &desc,
			},
		}

		decoded, err := DecodeToolParameters(&params, 0, logger)

		assert.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Contains(t, *decoded, "param1")

		param := (*decoded)["param1"]
		assert.Equal(t, ptype, utils.SafeDerefString(param.Type))
		assert.Equal(t, desc, utils.SafeDerefString(param.Description))
		assert.Equal(t, req, *param.Required)
	})

	t.Run("nil parameter values", func(t *testing.T) {
		params := map[string]*pklLLM.ToolProperties{
			"nilParam": nil,
		}

		decoded, err := DecodeToolParameters(&params, 0, logger)

		assert.NoError(t, err)
		assert.NotNil(t, decoded)
		// Nil parameters should be skipped
		_, exists := (*decoded)["nilParam"]
		assert.False(t, exists)
	})

	t.Run("empty parameters", func(t *testing.T) {
		params := map[string]*pklLLM.ToolProperties{}

		decoded, err := DecodeToolParameters(&params, 0, logger)

		assert.NoError(t, err)
		assert.NotNil(t, decoded)
		assert.Empty(t, *decoded)
	})
}
