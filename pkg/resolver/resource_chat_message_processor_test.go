package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestSummarizeMessageHistory(t *testing.T) {
	tests := []struct {
		name     string
		history  []llms.MessageContent
		expected string
	}{
		{
			name:     "empty history",
			history:  []llms.MessageContent{},
			expected: "",
		},
		{
			name: "single message",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
			},
			expected: "Role:human Parts:Hello",
		},
		{
			name: "multiple messages",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hi there"}},
				},
			},
			expected: "Role:human Parts:Hello; Role:ai Parts:Hi there",
		},
		{
			name: "message with multiple parts",
			history: []llms.MessageContent{
				{
					Role: llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{
						llms.TextContent{Text: "Part 1"},
						llms.TextContent{Text: "Part 2"},
					},
				},
			},
			expected: "Role:human Parts:Part 1|Part 2",
		},
		{
			name: "long message truncated",
			history: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "This is a very long message that should be truncated to 50 characters"}},
				},
			},
			expected: "Role:human Parts:This is a very long message that should be trun...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := summarizeMessageHistory(tt.history)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name             string
		jsonResponse     *bool
		jsonResponseKeys *[]string
		tools            []llms.Tool
		expected         string
	}{
		{
			name:         "no tools",
			jsonResponse: nil,
			tools:        []llms.Tool{},
			expected:     "No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:         "with JSON response",
			jsonResponse: boolPtr(true),
			tools:        []llms.Tool{},
			expected:     "Respond in JSON format. No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:             "with JSON response and keys",
			jsonResponse:     boolPtr(true),
			jsonResponseKeys: &[]string{"key1", "key2"},
			tools:            []llms.Tool{},
			expected:         "Respond in JSON format, include `key1`, `key2` in response keys. No tools are available. Respond with the final result as a string.\n",
		},
		{
			name:         "with tools",
			jsonResponse: nil,
			tools: []llms.Tool{
				{
					Function: &llms.FunctionDefinition{
						Name:        "test_tool",
						Description: "A test tool",
						Parameters: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"param1": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
			expected: "\n\nYou have access to the following tools. Use tools only when necessary to fulfill the request. Consider all previous tool outputs when deciding which tools to use next. After tool execution, you will receive the results in the conversation history. Do NOT suggest the same tool with identical parameters unless explicitly required by new user input. Once all necessary tools are executed, return the final result as a string (e.g., '12345', 'joel').\n\nWhen using tools, respond with a JSON array of tool call objects, each containing 'name' and 'arguments' fields, even for a single tool:\n[\n  {\n    \"name\": \"tool1\",\n    \"arguments\": {\n      \"param1\": \"value1\"\n    }\n  }\n]\n\nRules:\n- Return a JSON array for tool calls, even for one tool.\n- Include all required parameters.\n- Execute tools in the specified order, using previous tool outputs to inform parameters.\n- After tool execution, return the final result as a string without tool calls unless new tools are needed.\n- Do NOT include explanatory text with tool call JSON.\n\nAvailable tools:\n- test_tool: A test tool\n  - param1: \n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSystemPrompt(tt.jsonResponse, tt.jsonResponseKeys, tt.tools)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRoleAndType(t *testing.T) {
	tests := []struct {
		name         string
		rolePtr      *string
		expectedRole string
		expectedType llms.ChatMessageType
	}{
		{
			name:         "nil role",
			rolePtr:      nil,
			expectedRole: RoleHuman,
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "empty role",
			rolePtr:      stringPtr(""),
			expectedRole: RoleHuman,
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "human role",
			rolePtr:      stringPtr("human"),
			expectedRole: "human",
			expectedType: llms.ChatMessageTypeHuman,
		},
		{
			name:         "system role",
			rolePtr:      stringPtr("system"),
			expectedRole: "system",
			expectedType: llms.ChatMessageTypeSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role, msgType := getRoleAndType(tt.rolePtr)
			assert.Equal(t, tt.expectedRole, role)
			assert.Equal(t, tt.expectedType, msgType)
		})
	}
}

func TestProcessScenarioMessages(t *testing.T) {
	tests := []struct {
		name     string
		scenario *[]*pklLLM.MultiChat
		expected []llms.MessageContent
	}{
		{
			name:     "nil scenario",
			scenario: nil,
			expected: []llms.MessageContent{},
		},
		{
			name:     "empty scenario",
			scenario: &[]*pklLLM.MultiChat{},
			expected: []llms.MessageContent{},
		},
		{
			name: "single message",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("human"),
					Prompt: stringPtr("Hello"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
			},
		},
		{
			name: "multiple messages",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("human"),
					Prompt: stringPtr("Hello"),
				},
				{
					Role:   stringPtr("ai"),
					Prompt: stringPtr("Hi there"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeHuman,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hello"}},
				},
				{
					Role:  llms.ChatMessageTypeAI,
					Parts: []llms.ContentPart{llms.TextContent{Text: "Hi there"}},
				},
			},
		},
		{
			name: "generic role",
			scenario: &[]*pklLLM.MultiChat{
				{
					Role:   stringPtr("custom"),
					Prompt: stringPtr("Custom message"),
				},
			},
			expected: []llms.MessageContent{
				{
					Role:  llms.ChatMessageTypeGeneric,
					Parts: []llms.ContentPart{llms.TextContent{Text: "[custom]: Custom message"}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger()
			result := processScenarioMessages(tt.scenario, logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapRoleToLLMMessageType(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected llms.ChatMessageType
	}{
		{"human role", "human", llms.ChatMessageTypeHuman},
		{"user role", "user", llms.ChatMessageTypeHuman},
		{"person role", "person", llms.ChatMessageTypeHuman},
		{"client role", "client", llms.ChatMessageTypeHuman},
		{"system role", "system", llms.ChatMessageTypeSystem},
		{"ai role", "ai", llms.ChatMessageTypeAI},
		{"assistant role", "assistant", llms.ChatMessageTypeAI},
		{"bot role", "bot", llms.ChatMessageTypeAI},
		{"chatbot role", "chatbot", llms.ChatMessageTypeAI},
		{"llm role", "llm", llms.ChatMessageTypeAI},
		{"function role", "function", llms.ChatMessageTypeFunction},
		{"action role", "action", llms.ChatMessageTypeFunction},
		{"tool role", "tool", llms.ChatMessageTypeTool},
		{"unknown role", "unknown", llms.ChatMessageTypeGeneric},
		{"empty role", "", llms.ChatMessageTypeGeneric},
		{"whitespace role", "   ", llms.ChatMessageTypeGeneric},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapRoleToLLMMessageType(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
