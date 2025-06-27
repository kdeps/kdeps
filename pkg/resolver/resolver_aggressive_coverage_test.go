package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestResolverAggressiveCoverage(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("FormatDuration", func(t *testing.T) {
		// Test FormatDuration function - has 0.0% coverage
		duration := FormatDuration(3*time.Hour + 25*time.Minute + 45*time.Second)
		assert.Contains(t, duration, "3h")
		assert.Contains(t, duration, "25m")
		assert.Contains(t, duration, "45s")

		duration2 := FormatDuration(90 * time.Second)
		assert.Contains(t, duration2, "1m")
		assert.Contains(t, duration2, "30s")

		duration3 := FormatDuration(30 * time.Second)
		assert.Contains(t, duration3, "30s")
	})

	t.Run("MapRoleToLLMMessageType", func(t *testing.T) {
		// Test MapRoleToLLMMessageType function - has 0.0% coverage
		assert.Equal(t, llms.ChatMessageTypeHuman, MapRoleToLLMMessageType("human"))
		assert.Equal(t, llms.ChatMessageTypeHuman, MapRoleToLLMMessageType("user"))
		assert.Equal(t, llms.ChatMessageTypeSystem, MapRoleToLLMMessageType("system"))
		assert.Equal(t, llms.ChatMessageTypeAI, MapRoleToLLMMessageType("ai"))
		assert.Equal(t, llms.ChatMessageTypeAI, MapRoleToLLMMessageType("assistant"))
		assert.Equal(t, llms.ChatMessageTypeFunction, MapRoleToLLMMessageType("function"))
		assert.Equal(t, llms.ChatMessageTypeTool, MapRoleToLLMMessageType("tool"))
		assert.Equal(t, llms.ChatMessageTypeGeneric, MapRoleToLLMMessageType("unknown"))
	})

	t.Run("GetRoleAndType with 0% coverage", func(t *testing.T) {
		// Test GetRoleAndType function - improve from 0.0% coverage
		emptyStr := ""
		role, msgType := GetRoleAndType(&emptyStr)
		assert.Equal(t, "human", role)
		assert.Equal(t, llms.ChatMessageTypeHuman, msgType)

		systemStr := "system"
		role2, msgType2 := GetRoleAndType(&systemStr)
		assert.Equal(t, "system", role2)
		assert.Equal(t, llms.ChatMessageTypeSystem, msgType2)

		aiStr := "ai"
		role3, msgType3 := GetRoleAndType(&aiStr)
		assert.Equal(t, "ai", role3)
		assert.Equal(t, llms.ChatMessageTypeAI, msgType3)
	})

	t.Run("SummarizeMessageHistory with more coverage", func(t *testing.T) {
		// Test SummarizeMessageHistory function - improve from 81.8% coverage
		messages := []llms.MessageContent{
			{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.TextContent{Text: "First human message"},
					llms.TextContent{Text: "Second part"},
				},
			},
			{
				Role: llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{
					llms.TextContent{Text: "AI response message"},
				},
			},
		}
		result := SummarizeMessageHistory(messages)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "Role:human")
		assert.Contains(t, result, "Role:ai")
		assert.Contains(t, result, "|")
	})

	t.Run("ExtractToolNames with coverage", func(t *testing.T) {
		// Test ExtractToolNames function - improve from 0.0% coverage
		toolCalls := []llms.ToolCall{
			{
				ID:   "test1",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "test_function_1",
					Arguments: "{}",
				},
			},
			{
				ID:   "test2",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "test_function_2",
					Arguments: "{}",
				},
			},
			{
				ID:           "test3",
				Type:         "function",
				FunctionCall: nil, // Test nil function call
			},
		}
		result := ExtractToolNames(toolCalls)
		assert.Len(t, result, 2) // Should skip the nil function call
		assert.Contains(t, result, "test_function_1")
		assert.Contains(t, result, "test_function_2")
	})

	t.Run("ExtractToolNamesFromTools with coverage", func(t *testing.T) {
		// Test ExtractToolNamesFromTools function - improve from 0.0% coverage
		tools := []llms.Tool{
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "tool_1",
					Description: "First tool",
				},
			},
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "tool_2",
					Description: "Second tool",
				},
			},
			{
				Type:     "function",
				Function: nil, // Test nil function
			},
		}
		result := ExtractToolNamesFromTools(tools)
		assert.Len(t, result, 2) // Should skip the nil function
		assert.Contains(t, result, "tool_1")
		assert.Contains(t, result, "tool_2")
	})

	t.Run("DeduplicateToolCalls with coverage", func(t *testing.T) {
		// Test DeduplicateToolCalls function - improve from 0.0% coverage
		toolCalls := []llms.ToolCall{
			{
				ID:   "test1",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "duplicate_function",
					Arguments: `{"param": "value1"}`,
				},
			},
			{
				ID:   "test2",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "duplicate_function",
					Arguments: `{"param": "value1"}`, // Same name and args
				},
			},
			{
				ID:   "test3",
				Type: "function",
				FunctionCall: &llms.FunctionCall{
					Name:      "unique_function",
					Arguments: `{"param": "value2"}`,
				},
			},
			{
				ID:           "test4",
				Type:         "function",
				FunctionCall: nil, // Test nil function call
			},
		}
		result := DeduplicateToolCalls(toolCalls, logger)
		assert.Len(t, result, 2) // Should have 2 unique calls (skipping nil and duplicate)
	})

	t.Run("ConvertToolParamsToString comprehensive", func(t *testing.T) {
		// Test ConvertToolParamsToString function - improve from 80.0% coverage

		// Test string value
		result1 := ConvertToolParamsToString("test string", "param1", "tool1", logger)
		assert.Equal(t, "test string", result1)

		// Test float64 value
		result2 := ConvertToolParamsToString(42.5, "param2", "tool1", logger)
		assert.Equal(t, "42.5", result2)

		// Test bool value
		result3 := ConvertToolParamsToString(true, "param3", "tool1", logger)
		assert.Equal(t, "true", result3)

		result4 := ConvertToolParamsToString(false, "param4", "tool1", logger)
		assert.Equal(t, "false", result4)

		// Test nil value
		result5 := ConvertToolParamsToString(nil, "param5", "tool1", logger)
		assert.Equal(t, "", result5)

		// Test complex object (should be JSON marshaled)
		complexObj := map[string]interface{}{
			"nested": map[string]string{"key": "value"},
			"array":  []int{1, 2, 3},
		}
		result6 := ConvertToolParamsToString(complexObj, "param6", "tool1", logger)
		assert.Contains(t, result6, "nested")
		assert.Contains(t, result6, "array")
	})
}

func TestResolverUtilityFunctions(t *testing.T) {
	t.Run("ParseToolCallArgs comprehensive", func(t *testing.T) {
		// Test ParseToolCallArgs function - improve from 0.0% coverage
		logger := logging.NewTestLogger()

		// Test valid JSON
		validJSON := `{"param1": "value1", "param2": 42, "param3": true}`
		result, err := ParseToolCallArgs(validJSON, logger)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "value1", result["param1"])
		assert.Equal(t, float64(42), result["param2"]) // JSON numbers are float64
		assert.Equal(t, true, result["param3"])

		// Test invalid JSON
		invalidJSON := `{"invalid": json}`
		result2, err2 := ParseToolCallArgs(invalidJSON, logger)
		assert.Error(t, err2)
		assert.Nil(t, result2)

		// Test empty JSON
		emptyJSON := `{}`
		result3, err3 := ParseToolCallArgs(emptyJSON, logger)
		assert.NoError(t, err3)
		assert.NotNil(t, result3)
		assert.Empty(t, result3)
	})

	t.Run("CreateMockExecBlock comprehensive", func(t *testing.T) {
		// Test CreateMockExecBlock function - improve from 0.0% coverage
		result := CreateMockExecBlock("echo test", []string{"arg1", "arg2"})
		assert.NotNil(t, result)

		mock, ok := result.(*MockExecBlock)
		assert.True(t, ok)
		assert.Equal(t, "echo test", mock.GetCommand())

		args := mock.GetArgs()
		assert.NotNil(t, args)
		assert.Equal(t, []string{"arg1", "arg2"}, *args)
	})

	t.Run("CreateMockPythonBlock comprehensive", func(t *testing.T) {
		// Test CreateMockPythonBlock function - improve from 0.0% coverage
		result := CreateMockPythonBlock("print('hello')", []string{"--verbose"})
		assert.NotNil(t, result)

		mock, ok := result.(*MockPythonBlock)
		assert.True(t, ok)
		assert.Equal(t, "print('hello')", mock.GetScript())

		args := mock.GetArgs()
		assert.NotNil(t, args)
		assert.Equal(t, []string{"--verbose"}, *args)
	})

	t.Run("SchemaVersionUsage", func(t *testing.T) {
		// Ensure we're using schema.SchemaVersion as required
		ctx := context.Background()
		version := schema.SchemaVersion(ctx)
		assert.NotEmpty(t, version)
		assert.True(t, len(version) > 0)
	})
}
