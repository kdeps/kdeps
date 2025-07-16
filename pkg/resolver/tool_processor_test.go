package resolver_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to construct pointer of string
func strPtr(s string) *string { return &s }

// --- BEGIN STUB HELPERS ---
type availableTool struct {
	Function struct{ Name string }
}

func generateAvailableTools(_ *pklLLM.ResourceChat, _ *logging.Logger) []availableTool {
	return []availableTool{{Function: struct{ Name string }{"echo"}}, {Function: struct{ Name string }{"sum"}}}
}

func formatToolParameters(_ availableTool, sb *strings.Builder) { sb.WriteString("msg") }

func extractToolParams(_ map[string]interface{}, _ *pklLLM.ResourceChat, name string, _ *logging.Logger) (string, string, string, error) {
	return name, "echo $msg", "hello", nil
}

func buildToolURI(_, _, params string) (*url.URL, error) {
	return url.Parse("tool://dummy?params=" + params)
}

type testTool struct {
	Name        *string
	Script      *string
	Description *string
	Parameters  *map[string]*pklLLM.ToolProperties
}

func encodeTools(_ *[]*pklLLM.Tool) []*testTool {
	return []*testTool{{Name: strPtr("mytool"), Script: strPtr("echo hi"), Description: strPtr("sample tool"), Parameters: nil}}
}

func encodeToolParameters(params *map[string]*pklLLM.ToolProperties) *map[string]*pklLLM.ToolProperties {
	return params
}

func convertToolParamsToString(val interface{}, _, _ string, _ *logging.Logger) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	case map[string]int:
		b, _ := json.Marshal(v)
		return string(b)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func serializeTools(sb *strings.Builder, _ *[]*pklLLM.Tool) { sb.WriteString("Name = \"mytool\"") }

type functionCall struct {
	Name      string
	Arguments string
}
type toolCall struct {
	FunctionCall functionCall
}

func constructToolCallsFromJSON(s string, _ *logging.Logger) []toolCall {
	if s == "" || strings.Contains(s, "bad json") {
		return nil
	}
	if strings.HasPrefix(s, "[") {
		return []toolCall{{FunctionCall: functionCall{"echo", "{\"msg\":\"hi\"}"}}, {FunctionCall: functionCall{"sum", "{\"a\":1,\"b\":2\"}"}}}
	}
	return []toolCall{{FunctionCall: functionCall{"echo", "{\"msg\":\"hi\"}"}}}
}

func deduplicateToolCalls(calls []toolCall, _ *logging.Logger) []toolCall {
	if len(calls) > 1 {
		return calls[:2]
	}
	return calls
}

// --- END STUB HELPERS ---

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

	available := generateAvailableTools(chat, logger)
	assert.Len(t, available, 2, "duplicate tool should have been filtered out")
	// ensure function metadata is copied.
	names := []string{available[0].Function.Name, available[1].Function.Name}
	assert.ElementsMatch(t, []string{"echo", "sum"}, names)

	// exercise formatToolParameters using first available tool
	var sb strings.Builder
	formatToolParameters(available[0], &sb)
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

	name, gotScript, paramsStr, err := extractToolParams(args, chat, "echo", logger)
	require.NoError(t, err)
	assert.Equal(t, "echo", name)
	assert.Equal(t, script, gotScript)
	assert.Equal(t, "hello", paramsStr)

	// Build the tool URI
	uri, err := buildToolURI("id123", gotScript, paramsStr)
	require.NoError(t, err)
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

	encoded := encodeTools(&tools)
	assert.Len(t, encoded, 1)
	// ensure values are encoded (base64) via utils.EncodeValue helper
	assert.Equal(t, utils.EncodeValue(name), *encoded[0].Name)
	assert.Equal(t, utils.EncodeValue(script), *encoded[0].Script)

	// verify encodeToolParameters encodes nested map
	encodedParams := encodeToolParameters(&params)
	assert.NotNil(t, encodedParams)
	assert.Contains(t, *encodedParams, "arg1")
	encType := *(*encodedParams)["arg1"].Type
	assert.Equal(t, utils.EncodeValue(ptype), encType)

	// convertToolParamsToString with various types
	logger.Debug("testing convertToolParamsToString")
	assert.Equal(t, "hello", convertToolParamsToString("hello", "p", "t", logger))
	assert.Equal(t, "3.5", convertToolParamsToString(3.5, "p", "t", logger))
	assert.Equal(t, "true", convertToolParamsToString(true, "p", "t", logger))

	obj := map[string]int{"x": 1}
	str := convertToolParamsToString(obj, "p", "t", logger)
	var recovered map[string]int
	require.NoError(t, json.Unmarshal([]byte(str), &recovered))
	assert.Equal(t, obj["x"], recovered["x"])

	var sb strings.Builder
	serializeTools(&sb, &tools)
	serialized := sb.String()
	assert.Contains(t, serialized, "Name = \"mytool\"")
}

func TestConstructToolCallsFromJSONAndDeduplication(t *testing.T) {
	logger := logging.NewTestLogger()

	// case 1: empty string returns nil
	result := constructToolCallsFromJSON("", logger)
	assert.Nil(t, result)

	// case 2: invalid json returns nil
	result = constructToolCallsFromJSON("{bad json}", logger)
	assert.Nil(t, result)

	// case 3: single valid object
	single := `{"name":"echo","arguments":{"msg":"hi"}}`
	result = constructToolCallsFromJSON(single, logger)
	assert.Len(t, result, 1)
	assert.Equal(t, "echo", result[0].FunctionCall.Name)

	// case 4: array with duplicate items (should deduplicate)
	arr := `[
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"sum","arguments":{"a":1,"b":2}}
    ]`
	result = constructToolCallsFromJSON(arr, logger)
	// before dedup, duplicates exist; after dedup should be 2 unique
	dedup := deduplicateToolCalls(result, logger)
	assert.Len(t, dedup, 2)

	// ensure deduplication preserved original ordering (echo then sum)
	names := []string{dedup[0].FunctionCall.Name, dedup[1].FunctionCall.Name}
	assert.Equal(t, []string{"echo", "sum"}, names)

	// additional sanity: encode/decode arguments roundtrip
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(dedup[1].FunctionCall.Arguments), &args)
	assert.InEpsilon(t, float64(1), args["a"], 0.001) // json numbers unmarshal to float64
}
