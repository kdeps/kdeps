package resolver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gabriel-vasile/mimetype"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

// Constants for role strings.
const (
	RoleHuman     = "human"
	RoleUser      = "user"
	RolePerson    = "person"
	RoleClient    = "client"
	RoleSystem    = "system"
	RoleAI        = "ai"
	RoleAssistant = "assistant"
	RoleBot       = "bot"
	RoleChatbot   = "chatbot"
	RoleLLM       = "llm"
	RoleFunction  = "function"
	RoleAction    = "action"
	RoleTool      = "tool"
)

// HandleLLMChat initiates asynchronous processing of an LLM chat interaction.
func (dr *DependencyResolver) HandleLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	if err := dr.decodeChatBlock(chatBlock); err != nil {
		dr.Logger.Error("failed to decode chat block", "actionID", actionID, "error", err)
		return err
	}

	go func(aID string, block *pklLLM.ResourceChat) {
		if err := dr.processLLMChat(aID, block); err != nil {
			dr.Logger.Error("failed to process LLM chat", "actionID", aID, "error", err)
		}
	}(actionID, chatBlock)

	return nil
}

// decodeChatBlock decodes fields in the chat block, handling Base64 decoding where necessary.
func (dr *DependencyResolver) decodeChatBlock(chatBlock *pklLLM.ResourceChat) error {
	// Decode Prompt
	if err := decodeField(&chatBlock.Prompt, "Prompt", utils.SafeDerefString, ""); err != nil {
		return err
	}

	// Decode Role
	if err := decodeField(&chatBlock.Role, "Role", utils.SafeDerefString, RoleHuman); err != nil {
		return err
	}

	// Decode JSONResponseKeys
	if chatBlock.JSONResponseKeys != nil {
		decodedKeys, err := utils.DecodeStringSlice(chatBlock.JSONResponseKeys, "JSONResponseKeys")
		if err != nil {
			return fmt.Errorf("failed to decode JSONResponseKeys: %w", err)
		}
		chatBlock.JSONResponseKeys = decodedKeys
	}

	// Decode Scenario
	if err := decodeScenario(chatBlock, dr.Logger); err != nil {
		return err
	}

	// Decode Files
	if err := decodeFiles(chatBlock); err != nil {
		return err
	}

	// Decode Tools
	if err := decodeTools(chatBlock, dr.Logger); err != nil {
		return err
	}

	return nil
}

// decodeField decodes a single field, handling Base64 if needed, and uses a default value if the field is nil.
func decodeField(field **string, fieldName string, deref func(*string) string, defaultValue string) error {
	if field == nil || *field == nil {
		*field = &defaultValue
	}
	original := deref(*field)
	logger := logging.GetLogger()
	logger.Debug("Decoding field", "fieldName", fieldName, "original", original)
	decoded, err := utils.DecodeBase64IfNeeded(original)
	if err != nil {
		logger.Warn("Base64 decoding failed, using original value", "fieldName", fieldName, "error", err)
		decoded = original
	}
	if decoded == "" && original != "" {
		logger.Warn("Decoded value is empty, preserving original", "fieldName", fieldName, "original", original)
		decoded = original
	}
	*field = &decoded
	logger.Debug("Decoded field", "fieldName", fieldName, "decoded", decoded)
	return nil
}

// isBase64 checks if a string is likely Base64-encoded.
func isBase64(s string) bool {
	if len(s) == 0 {
		return false
	}
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

// decodeScenario decodes the Scenario field, handling nil and empty cases.
func decodeScenario(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	if chatBlock.Scenario == nil {
		logger.Info("Scenario is nil, initializing empty slice")
		emptyScenario := make([]*pklLLM.MultiChat, 0)
		chatBlock.Scenario = &emptyScenario
		return nil
	}

	logger.Info("Decoding Scenario", "length", len(*chatBlock.Scenario))
	decodedScenario := make([]*pklLLM.MultiChat, 0, len(*chatBlock.Scenario))
	for i, entry := range *chatBlock.Scenario {
		if entry == nil {
			logger.Warn("Scenario entry is nil", "index", i)
			continue
		}
		decodedEntry := &pklLLM.MultiChat{}
		if entry.Role != nil {
			decodedRole, err := utils.DecodeBase64IfNeeded(utils.SafeDerefString(entry.Role))
			if err != nil {
				logger.Error("Failed to decode scenario role", "index", i, "error", err)
				return err
			}
			decodedEntry.Role = &decodedRole
		} else {
			logger.Warn("Scenario role is nil", "index", i)
			defaultRole := RoleHuman
			decodedEntry.Role = &defaultRole
		}
		if entry.Prompt != nil {
			decodedPrompt, err := utils.DecodeBase64IfNeeded(utils.SafeDerefString(entry.Prompt))
			if err != nil {
				logger.Error("Failed to decode scenario prompt", "index", i, "error", err)
				return err
			}
			decodedEntry.Prompt = &decodedPrompt
		} else {
			logger.Warn("Scenario prompt is nil", "index", i)
			emptyPrompt := ""
			decodedEntry.Prompt = &emptyPrompt
		}
		logger.Info("Decoded Scenario entry", "index", i, "role", *decodedEntry.Role, "prompt", *decodedEntry.Prompt)
		decodedScenario = append(decodedScenario, decodedEntry)
	}
	chatBlock.Scenario = &decodedScenario
	return nil
}

// decodeFiles decodes the Files field, handling Base64 if needed.
func decodeFiles(chatBlock *pklLLM.ResourceChat) error {
	if chatBlock.Files == nil {
		return nil
	}
	decodedFiles := make([]string, len(*chatBlock.Files))
	for i, file := range *chatBlock.Files {
		decodedFile, err := utils.DecodeBase64IfNeeded(file)
		if err != nil {
			return fmt.Errorf("failed to decode Files[%d]: %w", i, err)
		}
		decodedFiles[i] = decodedFile
	}
	chatBlock.Files = &decodedFiles
	return nil
}

// decodeTools decodes the Tools field, handling nested parameters and nil cases.
func decodeTools(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	if chatBlock == nil {
		logger.Error("chatBlock is nil in decodeTools")
		return errors.New("chatBlock cannot be nil")
	}

	if chatBlock.Tools == nil {
		logger.Info("Tools is nil, initializing empty slice")
		emptyTools := make([]*pklLLM.Tool, 0)
		chatBlock.Tools = &emptyTools
		return nil
	}

	logger.Info("Decoding Tools", "length", len(*chatBlock.Tools))
	decodedTools := make([]*pklLLM.Tool, 0, len(*chatBlock.Tools))
	for i, entry := range *chatBlock.Tools {
		if entry == nil {
			logger.Warn("Tools entry is nil", "index", i)
			continue
		}
		logger.Debug("Processing tool entry", "index", i, "name", utils.SafeDerefString(entry.Name), "script", utils.SafeDerefString(entry.Script))
		decodedTool, err := decodeToolEntry(entry, i, logger)
		if err != nil {
			logger.Error("Failed to decode tool entry", "index", i, "error", err)
			return err
		}
		logger.Info("Decoded Tools entry", "index", i, "name", utils.SafeDerefString(decodedTool.Name))
		decodedTools = append(decodedTools, decodedTool)
	}
	chatBlock.Tools = &decodedTools
	return nil
}

// decodeToolEntry decodes a single Tool entry.
func decodeToolEntry(entry *pklLLM.Tool, index int, logger *logging.Logger) (*pklLLM.Tool, error) {
	if entry == nil {
		logger.Error("Tool entry is nil", "index", index)
		return nil, fmt.Errorf("tool entry at index %d is nil", index)
	}

	decodedTool := &pklLLM.Tool{}
	logger.Debug("Decoding tool", "index", index, "raw_name", entry.Name, "raw_script", entry.Script)

	// Decode Name
	if entry.Name != nil {
		nameStr := utils.SafeDerefString(entry.Name)
		logger.Debug("Checking if name is Base64", "index", index, "name", nameStr, "isBase64", isBase64(nameStr))
		if isBase64(nameStr) {
			if err := decodeField(&decodedTool.Name, fmt.Sprintf("Tools[%d].Name", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Name = entry.Name
			logger.Debug("Preserving non-Base64 tool name", "index", index, "name", nameStr)
		}
	} else {
		logger.Warn("Tool name is nil", "index", index)
		emptyName := ""
		decodedTool.Name = &emptyName
	}

	// Decode Script
	if entry.Script != nil {
		scriptStr := utils.SafeDerefString(entry.Script)
		logger.Debug("Checking if script is Base64", "index", index, "script_length", len(scriptStr), "isBase64", isBase64(scriptStr))
		if isBase64(scriptStr) {
			if err := decodeField(&decodedTool.Script, fmt.Sprintf("Tools[%d].Script", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Script = entry.Script
			logger.Debug("Preserving non-Base64 tool script", "index", index, "script_length", len(scriptStr))
		}
	} else {
		logger.Warn("Tool script is nil", "index", index)
		emptyScript := ""
		decodedTool.Script = &emptyScript
	}

	// Decode Description
	if entry.Description != nil {
		descStr := utils.SafeDerefString(entry.Description)
		logger.Debug("Checking if description is Base64", "index", index, "description", descStr, "isBase64", isBase64(descStr))
		if isBase64(descStr) {
			if err := decodeField(&decodedTool.Description, fmt.Sprintf("Tools[%d].Description", index), utils.SafeDerefString, ""); err != nil {
				return nil, err
			}
		} else {
			decodedTool.Description = entry.Description
			logger.Debug("Preserving non-Base64 tool description", "index", index, "description", descStr)
		}
	} else {
		logger.Warn("Tool description is nil", "index", index)
		emptyDesc := ""
		decodedTool.Description = &emptyDesc
	}

	// Decode Parameters
	if entry.Parameters != nil {
		params, err := decodeToolParameters(entry.Parameters, index, logger)
		if err != nil {
			return nil, err
		}
		decodedTool.Parameters = params
		logger.Debug("Decoded tool parameters", "index", index, "param_count", len(*params))
	} else {
		logger.Warn("Tool parameters are nil", "index", index)
		emptyParams := make(map[string]*pklLLM.ToolProperties)
		decodedTool.Parameters = &emptyParams
	}

	return decodedTool, nil
}

// decodeToolParameters decodes tool parameters.
func decodeToolParameters(params *map[string]*pklLLM.ToolProperties, index int, logger *logging.Logger) (*map[string]*pklLLM.ToolProperties, error) {
	decodedParams := make(map[string]*pklLLM.ToolProperties, len(*params))
	for paramName, param := range *params {
		if param == nil {
			logger.Info("Tools parameter is nil", "index", index, "paramName", paramName)
			continue
		}
		decodedParam := &pklLLM.ToolProperties{Required: param.Required}

		// Decode Type
		if param.Type != nil {
			typeStr := utils.SafeDerefString(param.Type)
			logger.Debug("Checking if parameter type is Base64", "index", index, "paramName", paramName, "type", typeStr, "isBase64", isBase64(typeStr))
			if isBase64(typeStr) {
				if err := decodeField(&decodedParam.Type, fmt.Sprintf("Tools[%d].Parameters[%s].Type", index, paramName), utils.SafeDerefString, ""); err != nil {
					return nil, err
				}
			} else {
				decodedParam.Type = param.Type
				logger.Debug("Preserving non-Base64 parameter type", "index", index, "paramName", paramName, "type", typeStr)
			}
		} else {
			logger.Warn("Parameter type is nil", "index", index, "paramName", paramName)
			emptyType := ""
			decodedParam.Type = &emptyType
		}

		// Decode Description
		if param.Description != nil {
			descStr := utils.SafeDerefString(param.Description)
			logger.Debug("Checking if parameter description is Base64", "index", index, "paramName", paramName, "description", descStr, "isBase64", isBase64(descStr))
			if isBase64(descStr) {
				if err := decodeField(&decodedParam.Description, fmt.Sprintf("Tools[%d].Parameters[%s].Description", index, paramName), utils.SafeDerefString, ""); err != nil {
					return nil, err
				}
			} else {
				decodedParam.Description = param.Description
				logger.Debug("Preserving non-Base64 parameter description", "index", index, "paramName", paramName, "description", descStr)
			}
		} else {
			logger.Warn("Parameter description is nil", "index", index, "paramName", paramName)
			emptyDesc := ""
			decodedParam.Description = &emptyDesc
		}

		decodedParams[paramName] = decodedParam
	}
	return &decodedParams, nil
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

// processLLMChat processes the LLM chat and saves the response.
func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	if chatBlock == nil {
		return errors.New("chatBlock cannot be nil")
	}

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	completion, err := generateChatResponse(dr.Context, dr.Fs, llm, chatBlock, dr.ToolReader, dr.Logger)
	if err != nil {
		return err
	}

	chatBlock.Response = &completion
	return dr.AppendChatEntry(actionID, chatBlock)
}

// generateAvailableTools creates a dynamic list of llms.Tool from chatBlock.Tools, designed for execution via dr.ToolReader.Read.
func generateAvailableTools(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) []llms.Tool {
	if chatBlock == nil || chatBlock.Tools == nil || len(*chatBlock.Tools) == 0 {
		logger.Info("No tools defined in chatBlock, returning empty availableTools")
		return nil
	}

	logger.Debug("Generating available tools", "tool_count", len(*chatBlock.Tools))
	tools := make([]llms.Tool, 0, len(*chatBlock.Tools))
	for i, tool := range *chatBlock.Tools {
		if tool == nil || tool.Name == nil || *tool.Name == "" {
			logger.Warn("Skipping invalid tool entry", "index", i)
			continue
		}

		name := *tool.Name
		logger.Debug("Processing tool", "index", i, "name", name)
		description := "Execute a script using PklResourceReader's run operation, storing output with the specified ID"
		if tool.Description != nil && *tool.Description != "" {
			description = *tool.Description
		}

		// Define parameters: id and script are required, params is optional
		properties := map[string]any{
			"id": map[string]any{
				"type":        "string",
				"description": "The unique identifier for the script output",
			},
			"script": map[string]any{
				"type":        "string",
				"description": "The inline script content or path to the script file",
			},
			"params": map[string]any{
				"type":        "string",
				"description": "Comma-separated parameters to pass to the script (optional)",
			},
		}
		required := []string{"id", "script"}

		// Add tool-specific parameters from pklLLM.Tool.Parameters
		if tool.Parameters != nil {
			for paramName, param := range *tool.Parameters {
				if param == nil {
					logger.Warn("Skipping nil parameter", "tool", name, "paramName", paramName)
					continue
				}
				// Skip reserved parameter names
				if paramName == "id" || paramName == "script" || paramName == "params" {
					logger.Warn("Skipping parameter with reserved name", "tool", name, "paramName", paramName)
					continue
				}
				paramType := "string"
				if param.Type != nil && *param.Type != "" {
					paramType = *param.Type
				}
				paramDesc := ""
				if param.Description != nil {
					paramDesc = *param.Description
				}
				properties[paramName] = map[string]any{
					"type":        paramType,
					"description": paramDesc,
				}
				if param.Required != nil && *param.Required {
					required = append(required, paramName)
				}
			}
		}

		tools = append(tools, llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": properties,
					"required":   required,
				},
			},
		})
		logger.Info("Added tool to availableTools", "name", name, "required_params", required)
	}

	return tools
}

// generateChatResponse generates a response from the LLM based on the chat block, executing tools via toolreader.
func generateChatResponse(ctx context.Context, fs afero.Fs, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, toolreader *tool.PklResourceReader, logger *logging.Logger) (string, error) {
	if chatBlock.JSONResponse == nil || !*chatBlock.JSONResponse {
		prompt := utils.SafeDerefString(chatBlock.Prompt)
		if strings.TrimSpace(prompt) == "" {
			return "", errors.New("prompt cannot be empty for non-JSON response")
		}
		logger.Info("Sending non-JSON prompt to LLM", "prompt", prompt)
		return llm.Call(ctx, prompt)
	}

	// Estimate capacity for content slice
	capacity := 1 // System prompt
	if strings.TrimSpace(utils.SafeDerefString(chatBlock.Prompt)) != "" {
		capacity++ // Main prompt
	}
	if chatBlock.Scenario != nil {
		capacity += len(*chatBlock.Scenario)
	}
	if chatBlock.Files != nil {
		capacity += len(*chatBlock.Files)
	}

	// Pre-allocate content slice
	content := make([]llms.MessageContent, 0, capacity)

	systemPrompt := buildSystemPrompt(chatBlock.JSONResponseKeys)
	role, roleType := getRoleAndType(chatBlock.Role)
	prompt := utils.SafeDerefString(chatBlock.Prompt)

	// Add system prompt
	content = append(content, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})

	// Add main prompt if present
	if strings.TrimSpace(prompt) != "" {
		if roleType == llms.ChatMessageTypeGeneric {
			prompt = fmt.Sprintf("[%s]: %s", role, prompt)
		}
		content = append(content, llms.MessageContent{
			Role:  roleType,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		})
	}

	// Add scenario messages
	content = append(content, processScenarioMessages(chatBlock.Scenario, logger)...)

	// Process files if present
	if chatBlock.Files != nil && len(*chatBlock.Files) > 0 {
		for i, filePath := range *chatBlock.Files {
			fileBytes, err := afero.ReadFile(fs, filePath)
			if err != nil {
				logger.Error("Failed to read file", "index", i, "path", filePath, "error", err)
				return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			fileType := mimetype.Detect(fileBytes).String()
			logger.Info("Detected MIME type for file", "index", i, "path", filePath, "mimeType", fileType)
			content = append(content, llms.MessageContent{
				Role: roleType,
				Parts: []llms.ContentPart{
					llms.BinaryPart(fileType, fileBytes),
				},
			})
		}
	}

	// Generate dynamic tools from chatBlock.Tools
	availableTools := generateAvailableTools(chatBlock, logger)

	// Log content being sent
	for i, msg := range content {
		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llms.TextContent:
				logger.Info("Sending message to LLM", "index", i, "role", msg.Role, "content", p.Text)
			case llms.BinaryContent:
				logger.Info("Sending binary content to LLM", "index", i, "role", msg.Role, "mimeType", p.MIMEType)
			}
		}
	}

	// Pass tools to LLM if available
	opts := []llms.CallOption{llms.WithJSONMode()}
	if len(availableTools) > 0 {
		opts = append(opts, llms.WithTools(availableTools))
	}

	response, err := llm.GenerateContent(ctx, content, opts...)
	if err != nil {
		logger.Error("Failed to generate JSON content", "error", err)
		return "", fmt.Errorf("failed to generate JSON content: %w", err)
	}
	if len(response.Choices) == 0 {
		logger.Error("Empty response from LLM")
		return "", errors.New("empty response from model")
	}

	// Process tool calls if present
	if result := processToolCalls(response.Choices[0].ToolCalls, toolreader, logger); result != "" {
		return result, nil
	}

	logger.Info("Received LLM response", "content", response.Choices[0].Content)
	return response.Choices[0].Content, nil
}

// processToolCalls processes tool calls and returns combined results or an empty string if none.
func processToolCalls(toolCalls []llms.ToolCall, toolreader *tool.PklResourceReader, logger *logging.Logger) string {
	if len(toolCalls) == 0 {
		return ""
	}

	toolResults := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc.FunctionCall == nil || tc.FunctionCall.Name == "" {
			logger.Warn("Skipping tool call with empty function name or nil FunctionCall")
			continue
		}
		logger.Info("Processing tool call", "name", tc.FunctionCall.Name, "arguments", tc.FunctionCall.Arguments)

		args, err := parseToolCallArgs(tc.FunctionCall.Arguments, logger)
		if err != nil {
			toolResults = append(toolResults, "Error: failed to parse arguments: "+err.Error())
			continue
		}

		id, script, paramsStr, err := extractToolParams(args, tc.FunctionCall.Name, logger)
		if err != nil {
			toolResults = append(toolResults, err.Error())
			continue
		}

		uri, err := buildToolURI(id, script, paramsStr)
		if err != nil {
			logger.Error("Failed to parse URI", "uri", uri, "error", err)
			toolResults = append(toolResults, "Error: failed to parse URI: "+err.Error())
			continue
		}

		result, err := toolreader.Read(*uri)
		if err != nil {
			logger.Error("Tool execution failed", "name", tc.FunctionCall.Name, "uri", uri.String(), "error", err)
			toolResults = append(toolResults, "Error: "+err.Error())
			continue
		}

		resultStr := string(result)
		logger.Info("Tool execution succeeded", "name", tc.FunctionCall.Name, "result", resultStr)
		toolResults = append(toolResults, resultStr)
	}

	if len(toolResults) == 0 {
		return ""
	}
	return strings.Join(toolResults, "\n")
}

// parseToolCallArgs parses JSON arguments from a tool call.
func parseToolCallArgs(arguments string, logger *logging.Logger) (map[string]interface{}, error) {
	args := make(map[string]interface{})
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		logger.Error("Failed to parse tool call arguments", "error", err)
		return nil, err
	}
	return args, nil
}

// extractToolParams extracts and validates tool call parameters.
func extractToolParams(args map[string]interface{}, toolName string, logger *logging.Logger) (string, string, string, error) {
	id, ok := args["id"].(string)
	if !ok || id == "" {
		logger.Error("Tool call missing required 'id' parameter", "name", toolName)
		return "", "", "", errors.New("Error: missing or invalid 'id' for " + toolName)
	}

	script, ok := args["script"].(string)
	if !ok || script == "" {
		logger.Error("Tool call missing required 'script' parameter", "name", toolName)
		return "", "", "", errors.New("Error: missing or invalid 'script' for " + toolName)
	}

	var extraParams []string
	if params, ok := args["params"].(string); ok && params != "" {
		extraParams = append(extraParams, params)
	}
	for key, value := range args {
		if key != "id" && key != "script" && key != "params" {
			if strVal, ok := value.(string); ok {
				extraParams = append(extraParams, strVal)
			}
		}
	}
	paramsStr := strings.Join(extraParams, ",")
	return id, script, paramsStr, nil
}

// buildToolURI constructs the URI for tool execution.
func buildToolURI(id, script, paramsStr string) (*url.URL, error) {
	queryParams := url.Values{
		"op":     []string{"run"},
		"script": []string{script},
	}
	if paramsStr != "" {
		queryParams.Add("params", paramsStr)
	}
	uriStr := "tool:/" + url.PathEscape(id) + "?" + queryParams.Encode()
	return url.Parse(uriStr)
}

// buildSystemPrompt constructs the system prompt for JSON responses.
func buildSystemPrompt(jsonResponseKeys *[]string) string {
	if jsonResponseKeys != nil && len(*jsonResponseKeys) > 0 {
		return fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", strings.Join(*jsonResponseKeys, "`, `"))
	}
	return "Respond in JSON format."
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
func processScenarioMessages(scenario *[]*pklLLM.MultiChat, logger *logging.Logger) []llms.MessageContent {
	if scenario == nil {
		logger.Info("No scenario messages to process")
		return make([]llms.MessageContent, 0)
	}

	logger.Info("Processing scenario messages", "count", len(*scenario))
	content := make([]llms.MessageContent, 0, len(*scenario))

	for i, entry := range *scenario {
		if entry == nil {
			logger.Info("Skipping nil scenario entry", "index", i)
			continue
		}
		prompt := utils.SafeDerefString(entry.Prompt)
		if strings.TrimSpace(prompt) == "" {
			logger.Info("Processing empty scenario prompt", "index", i, "role", utils.SafeDerefString(entry.Role))
		}
		entryRole, entryType := getRoleAndType(entry.Role)
		entryPrompt := prompt
		if entryType == llms.ChatMessageTypeGeneric {
			entryPrompt = fmt.Sprintf("[%s]: %s", entryRole, prompt)
		}
		logger.Info("Adding scenario message", "index", i, "role", entryRole, "prompt", entryPrompt)
		content = append(content, llms.MessageContent{
			Role:  entryType,
			Parts: []llms.ContentPart{llms.TextContent{Text: entryPrompt}},
		})
	}
	return content
}

// AppendChatEntry appends a chat entry to the Pkl file.
func (dr *DependencyResolver) AppendChatEntry(resourceID string, newChat *pklLLM.ResourceChat) error {
	pklPath := filepath.Join(dr.ActionDir, "llm/"+dr.RequestID+"__llm_output.pkl")

	llmRes, err := dr.LoadResource(dr.Context, pklPath, LLMResource)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	pklRes, ok := llmRes.(*pklLLM.LLMImpl)
	if !ok {
		return errors.New("failed to cast pklRes to *pklLLM.Resource")
	}

	resources := pklRes.GetResources()
	if resources == nil {
		emptyMap := make(map[string]*pklLLM.ResourceChat)
		resources = &emptyMap
	}
	existingResources := *resources

	var filePath string
	if newChat.Response != nil {
		filePath, err = dr.WriteResponseToFile(resourceID, newChat.Response)
		if err != nil {
			return fmt.Errorf("failed to write response to file: %w", err)
		}
		newChat.File = &filePath
	}

	// Encode newChat
	encodedChat := encodeChat(newChat, dr.Logger)
	existingResources[resourceID] = encodedChat

	// Generate PKL content
	pklContent := generatePklContent(existingResources, dr.Context, dr.Logger)

	// Write and evaluate PKL file
	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL file: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

// encodeChat encodes a ResourceChat for Pkl storage.
func encodeChat(chat *pklLLM.ResourceChat, logger *logging.Logger) *pklLLM.ResourceChat {
	// Encode Scenario
	var encodedScenario *[]*pklLLM.MultiChat
	if chat.Scenario != nil && len(*chat.Scenario) > 0 {
		encodedEntries := make([]*pklLLM.MultiChat, 0, len(*chat.Scenario))
		for i, entry := range *chat.Scenario {
			if entry == nil {
				logger.Warn("Skipping nil scenario entry in encodeChat", "index", i)
				continue
			}
			role := utils.SafeDerefString(entry.Role)
			if role == "" {
				role = RoleHuman
				logger.Info("Setting default role for scenario entry", "index", i, "role", role)
			}
			prompt := utils.SafeDerefString(entry.Prompt)
			logger.Info("Encoding scenario entry", "index", i, "role", role, "prompt", prompt)
			encodedRole := utils.EncodeValue(role)
			encodedPrompt := utils.EncodeValue(prompt)
			encodedEntries = append(encodedEntries, &pklLLM.MultiChat{
				Role:   &encodedRole,
				Prompt: &encodedPrompt,
			})
		}
		if len(encodedEntries) > 0 {
			encodedScenario = &encodedEntries
		} else {
			logger.Warn("No valid scenario entries after encoding", "original_length", len(*chat.Scenario))
		}
	} else {
		logger.Info("Scenario is nil or empty in encodeChat")
	}

	// Encode Tools
	var encodedTools *[]*pklLLM.Tool
	if chat.Tools != nil {
		encodedEntries := encodeTools(chat.Tools)
		encodedTools = &encodedEntries
	}

	// Encode Files
	var encodedFiles *[]string
	if chat.Files != nil {
		encodedEntries := make([]string, len(*chat.Files))
		for i, file := range *chat.Files {
			encodedEntries[i] = utils.EncodeValue(file)
		}
		encodedFiles = &encodedEntries
	}

	encodedModel := utils.EncodeValue(chat.Model)
	encodedRole := utils.EncodeValue(utils.SafeDerefString(chat.Role))
	encodedPrompt := utils.EncodeValue(utils.SafeDerefString(chat.Prompt))
	encodedResponse := utils.EncodeValuePtr(chat.Response)
	encodedJSONResponseKeys := encodeJSONResponseKeys(chat.JSONResponseKeys)

	timeoutDuration := chat.TimeoutDuration
	if timeoutDuration == nil {
		timeoutDuration = &pkl.Duration{Value: 60, Unit: pkl.Second}
	}

	timestamp := chat.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond}
	}

	return &pklLLM.ResourceChat{
		Model:            encodedModel,
		Prompt:           &encodedPrompt,
		Role:             &encodedRole,
		Scenario:         encodedScenario,
		Tools:            encodedTools,
		JSONResponse:     chat.JSONResponse,
		JSONResponseKeys: encodedJSONResponseKeys,
		Response:         encodedResponse,
		Files:            encodedFiles,
		File:             chat.File,
		Timestamp:        timestamp,
		TimeoutDuration:  timeoutDuration,
	}
}

// encodeTools encodes the Tools field.
func encodeTools(tools *[]*pklLLM.Tool) []*pklLLM.Tool {
	encodedEntries := make([]*pklLLM.Tool, len(*tools))
	for i, entry := range *tools {
		if entry == nil {
			continue
		}
		encodedName := utils.EncodeValue(utils.SafeDerefString(entry.Name))
		encodedScript := utils.EncodeValue(utils.SafeDerefString(entry.Script))
		encodedDescription := utils.EncodeValue(utils.SafeDerefString(entry.Description))

		var encodedParameters *map[string]*pklLLM.ToolProperties
		if entry.Parameters != nil {
			params := encodeToolParameters(entry.Parameters)
			encodedParameters = params
		}

		encodedEntries[i] = &pklLLM.Tool{
			Name:        &encodedName,
			Script:      &encodedScript,
			Description: &encodedDescription,
			Parameters:  encodedParameters,
		}
	}
	return encodedEntries
}

// encodeToolParameters encodes tool parameters.
func encodeToolParameters(params *map[string]*pklLLM.ToolProperties) *map[string]*pklLLM.ToolProperties {
	encodedParams := make(map[string]*pklLLM.ToolProperties, len(*params))
	for paramName, param := range *params {
		if param == nil {
			continue
		}
		encodedType := utils.EncodeValue(utils.SafeDerefString(param.Type))
		encodedDescription := utils.EncodeValue(utils.SafeDerefString(param.Description))
		encodedParams[paramName] = &pklLLM.ToolProperties{
			Required:    param.Required,
			Type:        &encodedType,
			Description: &encodedDescription,
		}
	}
	return &encodedParams
}

// encodeJSONResponseKeys encodes JSON response keys.
func encodeJSONResponseKeys(keys *[]string) *[]string {
	if keys == nil {
		return nil
	}
	encoded := make([]string, len(*keys))
	for i, v := range *keys {
		encoded[i] = utils.EncodeValue(v)
	}
	return &encoded
}

// generatePklContent generates Pkl content from resources.
func generatePklContent(resources map[string]*pklLLM.ResourceChat, ctx context.Context, logger *logging.Logger) string {
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion(ctx)))
	pklContent.WriteString("resources {\n")

	for id, res := range resources {
		logger.Info("Generating PKL for resource", "id", id)
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = %q\n", res.Model))

		// Prompt with default
		prompt := ""
		if res.Prompt != nil {
			prompt = *res.Prompt
		}
		pklContent.WriteString(fmt.Sprintf("    prompt = %q\n", prompt))

		// Role with default
		role := RoleHuman
		if res.Role != nil && *res.Role != "" {
			role = *res.Role
		}
		pklContent.WriteString(fmt.Sprintf("    role = %q\n", role))

		// Scenario
		pklContent.WriteString("    scenario ")
		if res.Scenario != nil && len(*res.Scenario) > 0 {
			logger.Info("Serializing scenario", "entry_count", len(*res.Scenario))
			pklContent.WriteString("{\n")
			for i, entry := range *res.Scenario {
				if entry == nil {
					logger.Warn("Skipping nil scenario entry in generatePklContent", "index", i)
					continue
				}
				pklContent.WriteString("      new {\n")
				entryRole := RoleHuman
				if entry.Role != nil && *entry.Role != "" {
					entryRole = *entry.Role
				}
				pklContent.WriteString(fmt.Sprintf("        role = %q\n", entryRole))
				entryPrompt := ""
				if entry.Prompt != nil {
					entryPrompt = *entry.Prompt
				}
				pklContent.WriteString(fmt.Sprintf("        prompt = %q\n", entryPrompt))
				logger.Info("Serialized scenario entry", "index", i, "role", entryRole, "prompt", entryPrompt)
				pklContent.WriteString("      }\n")
			}
			pklContent.WriteString("    }\n")
		} else {
			logger.Info("Scenario is nil or empty in generatePklContent")
			pklContent.WriteString("{}\n")
		}

		// Tools
		serializeTools(&pklContent, res.Tools)

		// JSONResponse with default
		jsonResponse := false
		if res.JSONResponse != nil {
			jsonResponse = *res.JSONResponse
		}
		pklContent.WriteString(fmt.Sprintf("    JSONResponse = %t\n", jsonResponse))

		// JSONResponseKeys
		pklContent.WriteString("    JSONResponseKeys ")
		if res.JSONResponseKeys != nil && len(*res.JSONResponseKeys) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.JSONResponseKeys))
		} else {
			pklContent.WriteString("{}\n")
		}

		// Files
		pklContent.WriteString("    files ")
		if res.Files != nil && len(*res.Files) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.Files))
		} else {
			pklContent.WriteString("{}\n")
		}

		// TimeoutDuration with default
		timeoutValue := 60.0
		timeoutUnit := pkl.Second
		if res.TimeoutDuration != nil {
			timeoutValue = res.TimeoutDuration.Value
			timeoutUnit = res.TimeoutDuration.Unit
		}
		pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %g.%s\n", timeoutValue, timeoutUnit.String()))

		// Timestamp with default
		timestampValue := float64(time.Now().Unix())
		timestampUnit := pkl.Nanosecond
		if res.Timestamp != nil {
			timestampValue = res.Timestamp.Value
			timestampUnit = res.Timestamp.Unit
		}
		pklContent.WriteString(fmt.Sprintf("    timestamp = %g.%s\n", timestampValue, timestampUnit.String()))

		// Response
		if res.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = #\"\"\"\n%s\n\"\"\"#\n", *res.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n")
		}

		// File
		if res.File != nil {
			pklContent.WriteString(fmt.Sprintf("    file = %q\n", *res.File))
		} else {
			pklContent.WriteString("    file = \"\"\n")
		}

		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	return pklContent.String()
}

// serializeTools serializes the Tools field to Pkl format.
func serializeTools(builder *strings.Builder, tools *[]*pklLLM.Tool) {
	builder.WriteString("    tools ")
	if tools == nil || len(*tools) == 0 {
		builder.WriteString("{}\n")
		return
	}

	builder.WriteString("{\n")
	for _, entry := range *tools {
		if entry == nil {
			continue
		}
		builder.WriteString("      new {\n")
		name := ""
		if entry.Name != nil {
			name = *entry.Name
		}
		builder.WriteString(fmt.Sprintf("        name = %q\n", name))
		script := ""
		if entry.Script != nil {
			script = *entry.Script
		}
		builder.WriteString(fmt.Sprintf("        script = #\"\"\"\n%s\n\"\"\"#\n", script))
		description := ""
		if entry.Description != nil {
			description = *entry.Description
		}
		builder.WriteString(fmt.Sprintf("        description = %q\n", description))
		builder.WriteString("        parameters ")
		if entry.Parameters != nil && len(*entry.Parameters) > 0 {
			builder.WriteString("{\n")
			for pname, param := range *entry.Parameters {
				if param == nil {
					continue
				}
				builder.WriteString(fmt.Sprintf("          [\"%s\"] {\n", pname))
				required := false
				if param.Required != nil {
					required = *param.Required
				}
				builder.WriteString(fmt.Sprintf("            required = %t\n", required))
				paramType := ""
				if param.Type != nil {
					paramType = *param.Type
				}
				builder.WriteString(fmt.Sprintf("            type = %q\n", paramType))
				paramDescription := ""
				if param.Description != nil {
					paramDescription = *param.Description
				}
				builder.WriteString(fmt.Sprintf("            description = %q\n", paramDescription))
				builder.WriteString("          }\n")
			}
			builder.WriteString("        }\n")
		} else {
			builder.WriteString("{}\n")
		}
		builder.WriteString("      }\n")
	}
	builder.WriteString("    }\n")
}

// WriteResponseToFile writes the LLM response to a file.
func (dr *DependencyResolver) WriteResponseToFile(resourceID string, responseEncoded *string) (string, error) {
	if responseEncoded == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	content, err := utils.DecodeBase64IfNeeded(utils.SafeDerefString(responseEncoded))
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}
