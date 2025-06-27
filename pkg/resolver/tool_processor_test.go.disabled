package resolver_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	. "github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
)

// helper to construct pointer of string
func strPtr(s string) *string { return &s }

func TestGenerateAvailableToolsAndRelatedHelpers(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build a ResourceChat with two tools – one duplicate to hit duplicate filtering.
	desc := "echo something"
	req := true
	// Parameters definition
	params := map[string]*pklLLM.ToolProperties{
		"msg": {
			Required:    &req,
			Type:        strPtr("string"),
			Description: strPtr("message to echo"),
		},
	}

	tool1 := &pklLLM.Tool{
		Name:        strPtr("echo"),
		Description: &desc,
		Script:      strPtr("echo $msg"),
		Parameters:  &params,
	}
	// Duplicate with same name (should be skipped by generator)
	toolDup := &pklLLM.Tool{
		Name:       strPtr("echo"),
		Script:     strPtr("echo dup"),
		Parameters: &params,
	}
	tool2 := &pklLLM.Tool{
		Name:        strPtr("sum"),
		Description: strPtr("add numbers"),
		Script:      strPtr("expr $a + $b"),
	}

	toolsSlice := []*pklLLM.Tool{tool1, toolDup, tool2}
	chat := &pklLLM.ResourceChat{Tools: &toolsSlice}

	available := GenerateAvailableTools(chat, logger)
	assert.Len(t, available, 2, "duplicate tool should have been filtered out")
	// ensure function metadata is copied.
	names := []string{available[0].Function.Name, available[1].Function.Name}
	assert.ElementsMatch(t, []string{"echo", "sum"}, names)

	// exercise formatToolParameters using first available tool
	var sb strings.Builder
	FormatToolParameters(available[0], &sb)
	formatted := sb.String()
	assert.Contains(t, formatted, "msg", "expected parameter name in formatted output")
}

func TestBuildToolURIAndExtractParams(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build chatBlock for extractToolParams
	req := true
	script := "echo $msg"
	toolProps := map[string]*pklLLM.ToolProperties{
		"msg": {Required: &req, Type: strPtr("string"), Description: strPtr("m")},
	}
	toolEntry := &pklLLM.Tool{
		Name:       strPtr("echo"),
		Script:     &script,
		Parameters: &toolProps,
	}
	tools := []*pklLLM.Tool{toolEntry}
	chat := &pklLLM.ResourceChat{Tools: &tools}

	// Arguments map simulating parsed JSON args
	args := map[string]interface{}{"msg": "hello"}

	name, gotScript, paramsStr, err := ExtractToolParams(args, chat, "echo", logger)
	assert.NoError(t, err)
	assert.Equal(t, "echo", name)
	assert.Equal(t, script, gotScript)
	assert.Equal(t, "hello", paramsStr)

	// Build the tool URI
	uri, err := BuildToolURI("id123", gotScript, paramsStr)
	assert.NoError(t, err)
	// Should encode params as query param
	assert.Contains(t, uri.String(), "params=")

	// We omit executing through tool reader to keep the test lightweight.
}

func TestEncodeToolsAndParamsUnit(t *testing.T) {
	logger := logging.NewTestLogger()

	name := "mytool"
	script := "echo hi"
	desc := "sample tool"
	req := true
	ptype := "string"

	params := map[string]*pklLLM.ToolProperties{
		"arg1": {
			Required:    &req,
			Type:        &ptype,
			Description: &desc,
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
	// ensure values are encoded (base64) via utils.EncodeValue helper
	assert.Equal(t, utils.EncodeValue(name), *encoded[0].Name)
	assert.Equal(t, utils.EncodeValue(script), *encoded[0].Script)

	// verify encodeToolParameters encodes nested map
	encodedParams := EncodeToolParameters(&params)
	assert.NotNil(t, encodedParams)
	assert.Contains(t, *encodedParams, "arg1")
	encType := *(*encodedParams)["arg1"].Type
	assert.Equal(t, utils.EncodeValue(ptype), encType)

	// convertToolParamsToString with various types
	logger.Debug("testing convertToolParamsToString")
	assert.Equal(t, "hello", ConvertToolParamsToString("hello", "p", "t", logger))
	assert.Equal(t, "3.5", ConvertToolParamsToString(3.5, "p", "t", logger))
	assert.Equal(t, "true", ConvertToolParamsToString(true, "p", "t", logger))

	obj := map[string]int{"x": 1}
	str := ConvertToolParamsToString(obj, "p", "t", logger)
	var recovered map[string]int
	assert.NoError(t, json.Unmarshal([]byte(str), &recovered))
	assert.Equal(t, obj["x"], recovered["x"])

	var sb strings.Builder
	SerializeTools(&sb, &tools)
	serialized := sb.String()
	assert.Contains(t, serialized, "name = \"mytool\"")
}

func TestConstructToolCallsFromJSONAndDeduplication(t *testing.T) {
	logger := logging.NewTestLogger()

	// case 1: empty string returns nil
	result := ConstructToolCallsFromJSON("", logger)
	assert.Nil(t, result)

	// case 2: invalid json returns nil
	result = ConstructToolCallsFromJSON("{bad json}", logger)
	assert.Nil(t, result)

	// case 3: single valid object
	single := `{"name":"echo","arguments":{"msg":"hi"}}`
	result = ConstructToolCallsFromJSON(single, logger)
	assert.Len(t, result, 1)
	assert.Equal(t, "echo", result[0].FunctionCall.Name)

	// case 4: array with duplicate items (should deduplicate)
	arr := `[
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"sum","arguments":{"a":1,"b":2}}
    ]`
	result = ConstructToolCallsFromJSON(arr, logger)
	// before dedup, duplicates exist; after dedup should be 2 unique
	dedup := DeduplicateToolCalls(result, logger)
	assert.Len(t, dedup, 2)

	// ensure deduplication preserved original ordering (echo then sum)
	names := []string{dedup[0].FunctionCall.Name, dedup[1].FunctionCall.Name}
	assert.Equal(t, []string{"echo", "sum"}, names)

	// additional sanity: encode/decode arguments roundtrip
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(dedup[1].FunctionCall.Arguments), &args)
	assert.Equal(t, float64(1), args["a"]) // json numbers unmarshal to float64
}

func TestFormatToolParameters(t *testing.T) {
	t.Run("ValidToolWithParameters", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"description": "First parameter",
							"type":        "string",
						},
						"param2": map[string]interface{}{
							"description": "Second parameter",
							"type":        "number",
						},
					},
					"required": []interface{}{"param1"},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Contains(t, result, "param1: First parameter (required)")
		require.Contains(t, result, "param2: Second parameter")
		require.Contains(t, result, "  - param1:")
		require.Contains(t, result, "  - param2:")
	})

	t.Run("ToolWithNilFunction", func(t *testing.T) {
		tool := llms.Tool{
			Function: nil,
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Empty(t, result)
	})

	t.Run("ToolWithNilParameters", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: nil,
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Empty(t, result)
	})

	t.Run("ToolWithInvalidParametersType", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: "not a map",
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Empty(t, result)
	})

	t.Run("ToolWithInvalidPropertiesType", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": "not a map",
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Empty(t, result)
	})

	t.Run("ToolWithInvalidParameterType", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": "not a map",
					},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		// The function should skip invalid parameter types but still add a newline at the end
		require.Equal(t, "\n", result)
	})

	t.Run("ToolWithMissingDescription", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Contains(t, result, "param1: ")
		require.Contains(t, result, "  - param1:")
	})

	t.Run("ToolWithMultipleRequiredParameters", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"description": "First required parameter",
							"type":        "string",
						},
						"param2": map[string]interface{}{
							"description": "Second required parameter",
							"type":        "number",
						},
						"param3": map[string]interface{}{
							"description": "Optional parameter",
							"type":        "boolean",
						},
					},
					"required": []interface{}{"param1", "param2"},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Contains(t, result, "param1: First required parameter (required)")
		require.Contains(t, result, "param2: Second required parameter (required)")
		require.Contains(t, result, "param3: Optional parameter")
		require.NotContains(t, result, "param3: Optional parameter (required)")
	})

	t.Run("ToolWithEmptyRequiredList", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"description": "Optional parameter",
							"type":        "string",
						},
					},
					"required": []interface{}{},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Contains(t, result, "param1: Optional parameter")
		require.NotContains(t, result, "(required)")
	})

	t.Run("ToolWithNilRequiredList", func(t *testing.T) {
		tool := llms.Tool{
			Function: &llms.FunctionDefinition{
				Parameters: map[string]interface{}{
					"properties": map[string]interface{}{
						"param1": map[string]interface{}{
							"description": "Optional parameter",
							"type":        "string",
						},
					},
				},
			},
		}

		var sb strings.Builder
		FormatToolParameters(tool, &sb)
		result := sb.String()

		require.Contains(t, result, "param1: Optional parameter")
		require.NotContains(t, result, "(required)")
	})
}

func TestConvertToolParamsToString(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("StringValue", func(t *testing.T) {
		result := ConvertToolParamsToString("test string", "param1", "tool1", logger)
		require.Equal(t, "test string", result)
	})

	t.Run("Float64Value", func(t *testing.T) {
		result := ConvertToolParamsToString(3.14159, "param2", "tool1", logger)
		require.Equal(t, "3.14159", result)
	})

	t.Run("BoolValue", func(t *testing.T) {
		result := ConvertToolParamsToString(true, "param3", "tool1", logger)
		require.Equal(t, "true", result)
	})

	t.Run("NilValue", func(t *testing.T) {
		result := ConvertToolParamsToString(nil, "param4", "tool1", logger)
		require.Equal(t, "", result)
	})

	t.Run("IntValue", func(t *testing.T) {
		result := ConvertToolParamsToString(42, "param5", "tool1", logger)
		require.Equal(t, "42", result)
	})

	t.Run("SliceValue", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		result := ConvertToolParamsToString(slice, "param6", "tool1", logger)
		require.Equal(t, `["a","b","c"]`, result)
	})

	t.Run("MapValue", func(t *testing.T) {
		m := map[string]interface{}{"key": "value", "num": 123}
		result := ConvertToolParamsToString(m, "param7", "tool1", logger)
		require.Contains(t, result, `"key":"value"`)
		require.Contains(t, result, `"num":123`)
	})

	t.Run("StructValue", func(t *testing.T) {
		type testStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		obj := testStruct{Name: "test", Value: 42}
		result := ConvertToolParamsToString(obj, "param8", "tool1", logger)
		require.Contains(t, result, `"name":"test"`)
		require.Contains(t, result, `"value":42`)
	})

	t.Run("ComplexValue", func(t *testing.T) {
		complex := []map[string]interface{}{
			{"type": "string", "required": true},
			{"type": "number", "required": false},
		}
		result := ConvertToolParamsToString(complex, "param9", "tool1", logger)
		require.Contains(t, result, `"type":"string"`)
		require.Contains(t, result, `"required":true`)
		require.Contains(t, result, `"type":"number"`)
		require.Contains(t, result, `"required":false`)
	})

	t.Run("EmptySlice", func(t *testing.T) {
		result := ConvertToolParamsToString([]string{}, "param10", "tool1", logger)
		require.Equal(t, "[]", result)
	})

	t.Run("EmptyMap", func(t *testing.T) {
		result := ConvertToolParamsToString(map[string]interface{}{}, "param11", "tool1", logger)
		require.Equal(t, "{}", result)
	})
}
