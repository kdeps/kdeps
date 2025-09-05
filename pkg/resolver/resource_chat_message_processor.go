package resolver

import (
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/tmc/langchaingo/llms"
)

// summarizeMessageHistory creates a concise summary of message history for logging.
func summarizeMessageHistory(history []llms.MessageContent) string {
	var summary strings.Builder
	for i, msg := range history {
		if i > 0 {
			summary.WriteString("; ")
		}
		summary.WriteString("Role:" + string(msg.Role) + " Parts:")
		for j, part := range msg.Parts {
			if j > 0 {
				summary.WriteString("|")
			}
			if textPart, ok := part.(llms.TextContent); ok {
				summary.WriteString(utils.TruncateString(textPart.Text, 50))
			}
		}
	}
	return summary.String()
}

// buildSystemPrompt constructs the system prompt with strict JSON tool usage instructions.
func buildSystemPrompt(jsonResponse *bool, jsonResponseKeys *[]string, tools []llms.Tool) string {
	var sb strings.Builder

	if jsonResponse != nil && *jsonResponse {
		if jsonResponseKeys != nil && len(*jsonResponseKeys) > 0 {
			sb.WriteString("Respond in JSON format, include `" + strings.Join(*jsonResponseKeys, "`, `") + "` in response keys. ")
		} else {
			sb.WriteString("Respond in JSON format. ")
		}
	}

	if len(tools) == 0 {
		sb.WriteString("No tools are available. Respond with the final result as a string.\n")
		return sb.String()
	}

	sb.WriteString("\n\nYou have access to the following tools. Use tools only when necessary to fulfill the request. Consider all previous tool outputs when deciding which tools to use next. After tool execution, you will receive the results in the conversation history. Do NOT suggest the same tool with identical parameters unless explicitly required by new user input. Once all necessary tools are executed, return the final result as a string (e.g., '12345', 'joel').\n\n")
	sb.WriteString("When using tools, respond with a JSON array of tool call objects, each containing 'name' and 'arguments' fields, even for a single tool:\n")
	sb.WriteString("[\n  {\n    \"name\": \"tool1\",\n    \"arguments\": {\n      \"param1\": \"value1\"\n    }\n  }\n]\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Return a JSON array for tool calls, even for one tool.\n")
	sb.WriteString("- Include all required parameters.\n")
	sb.WriteString("- Execute tools in the specified order, using previous tool outputs to inform parameters.\n")
	sb.WriteString("- After tool execution, return the final result as a string without tool calls unless new tools are needed.\n")
	sb.WriteString("- Do NOT include explanatory text with tool call JSON.\n")
	sb.WriteString("\nAvailable tools:\n")
	for _, tool := range tools {
		if tool.Function != nil {
			sb.WriteString("- " + tool.Function.Name + ": " + tool.Function.Description + "\n")
			formatToolParameters(tool, &sb)
		}
	}

	return sb.String()
}

// getRoleAndType retrieves the role and its corresponding message type.
func getRoleAndType(rolePtr *string) (string, llms.ChatMessageType) {
	role := utils.SafeDerefString(rolePtr)
	if strings.TrimSpace(role) == "" {
		role = RoleHuman
	}
	return role, mapRoleToLLMMessageType(role)
}

// processScenarioMessages processes scenario entries into LLM messages.
func processScenarioMessages(scenario *[]pklLLM.MultiChat, logger *logging.Logger) []llms.MessageContent {
	if scenario == nil {
		logger.Info("No scenario messages to process")
		return make([]llms.MessageContent, 0)
	}

	logger.Info("Processing scenario messages", "count", len(*scenario))
	content := make([]llms.MessageContent, 0, len(*scenario))

	for i, entry := range *scenario {
		// MultiChat is a struct, not a pointer, so we can always access it
		prompt := utils.SafeDerefString(entry.Prompt)
		if strings.TrimSpace(prompt) == "" {
			logger.Info("Processing empty scenario prompt", "index", i, "role", utils.SafeDerefString(entry.Role))
		}
		entryRole, entryType := getRoleAndType(entry.Role)
		entryPrompt := prompt
		if entryType == llms.ChatMessageTypeGeneric {
			entryPrompt = "[" + entryRole + "]: " + prompt
		}
		logger.Info("Adding scenario message", "index", i, "role", entryRole, "prompt", entryPrompt)
		content = append(content, llms.MessageContent{
			Role:  entryType,
			Parts: []llms.ContentPart{llms.TextContent{Text: entryPrompt}},
		})
	}
	return content
}

// mapRoleToLLMMessageType maps user-defined roles to llms.ChatMessageType.
func mapRoleToLLMMessageType(role string) llms.ChatMessageType {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleHuman, RoleUser, RolePerson, RoleClient:
		return llms.ChatMessageTypeHuman
	case RoleSystem:
		return llms.ChatMessageTypeSystem
	case RoleAI, RoleAssistant, RoleBot, RoleChatbot, RoleLLM:
		return llms.ChatMessageTypeAI
	case RoleFunction, RoleAction:
		return llms.ChatMessageTypeFunction
	case RoleTool:
		return llms.ChatMessageTypeTool
	default:
		return llms.ChatMessageTypeGeneric
	}
}
