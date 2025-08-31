package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/tmc/langchaingo/llms"
)

// generateAvailableTools creates a dynamic list of llms.Tool from chatBlock.Tools.
func generateAvailableTools(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) []llms.Tool {
	if chatBlock == nil || chatBlock.Tools == nil || len(*chatBlock.Tools) == 0 {
		logger.Info("No tools defined in chatBlock, returning empty availableTools")
		return nil
	}

	logger.Debug("Generating available tools", "tool_count", len(*chatBlock.Tools))
	tools := make([]llms.Tool, 0, len(*chatBlock.Tools))
	seenNames := make(map[string]struct{})

	for i, toolDef := range *chatBlock.Tools {
		// Tool is a struct, not a pointer, so we can always access it
		if toolDef.Name == nil || *toolDef.Name == "" {
			logger.Warn("Skipping invalid tool entry", "index", i)
			continue
		}

		name := *toolDef.Name
		if _, exists := seenNames[name]; exists {
			logger.Warn("Duplicate tool name detected", "name", name, "index", i)
			continue
		}
		seenNames[name] = struct{}{}
		logger.Debug("Processing tool", "index", i, "name", name)

		description := "Execute the '" + name + "' tool when you need to perform this specific action. "
		if toolDef.Description != nil && *toolDef.Description != "" {
			description += *toolDef.Description
		} else if toolDef.Script != nil && *toolDef.Script != "" {
			description += "This tool executes the following script: " + utils.TruncateString(*toolDef.Script, 100)
		}

		properties := map[string]any{}
		required := []string{"name"}

		if toolDef.Parameters != nil {
			for paramName, param := range *toolDef.Parameters {
				// ToolProperties is a struct, not a pointer, so we can always access it

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
		logger.Info("Added tool to availableTools",
			"name", name,
			"description", description,
			"required_params", required,
			"all_params", properties)
	}

	return tools
}

// constructToolCallsFromJSON parses a JSON string into a slice of llms.ToolCall.
func constructToolCallsFromJSON(jsonContent string, logger *logging.Logger) []llms.ToolCall {
	if jsonContent == "" {
		logger.Info("JSON content is empty, returning empty ToolCalls")
		return nil
	}

	type jsonToolCall struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	var toolCalls []jsonToolCall
	var singleCall jsonToolCall

	err := json.Unmarshal([]byte(jsonContent), &toolCalls)
	if err != nil {
		if err := json.Unmarshal([]byte(jsonContent), &singleCall); err != nil {
			logger.Warn("Failed to unmarshal JSON content as array or single object", "content", utils.TruncateString(jsonContent, 100), "error", err)
			return nil
		}
		toolCalls = []jsonToolCall{singleCall}
	}

	if len(toolCalls) == 0 {
		logger.Info("No tool calls found in JSON content")
		return nil
	}

	result := make([]llms.ToolCall, 0, len(toolCalls))
	seen := make(map[string]struct{})
	var errors []string

	for i, tc := range toolCalls {
		if tc.Name == "" || tc.Arguments == nil {
			logger.Warn("Skipping invalid tool call", "index", i, "name", tc.Name)
			errors = append(errors, "tool call at index "+strconv.Itoa(i)+" has empty name or nil arguments")
			continue
		}

		argsJSON, err := json.Marshal(tc.Arguments)
		if err != nil {
			logger.Warn("Failed to marshal arguments", "index", i, "name", tc.Name, "error", err)
			errors = append(errors, "failed to marshal arguments for "+tc.Name+" at index "+strconv.Itoa(i)+": "+err.Error())
			continue
		}

		key := tc.Name + ":" + string(argsJSON)
		if _, exists := seen[key]; exists {
			logger.Info("Skipping duplicate tool call in JSON", "name", tc.Name, "arguments", string(argsJSON))
			continue
		}
		seen[key] = struct{}{}

		toolCallID := uuid.New().String()
		result = append(result, llms.ToolCall{
			ID:   toolCallID,
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      tc.Name,
				Arguments: string(argsJSON),
			},
		})

		logger.Info("Constructed tool call",
			"index", i,
			"id", toolCallID,
			"name", tc.Name,
			"arguments", utils.TruncateString(string(argsJSON), 100))
	}

	if len(result) == 0 && len(errors) > 0 {
		logger.Warn("No valid tool calls constructed", "errors", errors)
		return nil
	}

	logger.Info("Constructed tool calls", "count", len(result))
	return result
}

// extractToolParams extracts and validates tool call parameters.
func extractToolParams(args map[string]interface{}, chatBlock *pklLLM.ResourceChat, toolName string, logger *logging.Logger) (string, string, string, error) {
	if chatBlock.Tools == nil {
		logger.Error("chatBlock.Tools is nil in extractToolParams")
		return "", "", "", errors.New("tools field is nil")
	}

	var name, script string
	var toolParams *map[string]pklLLM.ToolProperties

	for i, toolDef := range *chatBlock.Tools {
		// Tool is a struct, not a pointer, so we can always access it
		if toolDef.Name == nil || *toolDef.Name == "" {
			logger.Warn("Skipping invalid tool entry", "index", i)
			continue
		}
		if toolDef.Script == nil || *toolDef.Script == "" {
			logger.Warn("Skipping invalid tool entry", "index", i)
			continue
		}
		if *toolDef.Name == toolName {
			name = *toolDef.Name
			script = *toolDef.Script
			toolParams = toolDef.Parameters
			break
		}
	}

	if name == "" || script == "" {
		logger.Error("Tool not found or invalid", "toolName", toolName)
		return "", "", "", fmt.Errorf("tool %s not found or has invalid definition", toolName)
	}

	var paramValues []string
	var missingRequired []string
	paramOrder := make([]string, 0)

	if toolParams != nil {
		// Collect parameter names in definition order
		for paramName := range *toolParams {
			paramOrder = append(paramOrder, paramName)
		}
		// Process parameters in order
		for _, paramName := range paramOrder {
			param := (*toolParams)[paramName]
			// ToolProperties is a struct, not a pointer, so we can always access it

			if value, exists := args[paramName]; exists {
				strVal := convertToolParamsToString(value, paramName, toolName, logger)
				if strVal != "" {
					paramValues = append(paramValues, strVal)
				}
			} else if param.Required != nil && *param.Required {
				missingRequired = append(missingRequired, paramName)
			}
		}
	}

	// Handle any extra parameters not in tool definition
	for paramName, value := range args {
		if toolParams != nil {
			if _, exists := (*toolParams)[paramName]; exists {
				continue
			}
		}
		strVal := convertToolParamsToString(value, paramName, toolName, logger)
		if strVal != "" {
			paramValues = append(paramValues, strVal)
		}
	}

	if len(missingRequired) > 0 {
		logger.Warn("Missing required parameters", "tool", toolName, "parameters", missingRequired)
	}

	paramsStr := strings.Join(paramValues, " ")
	if paramsStr == "" {
		logger.Warn("No parameters extracted for tool", "tool", toolName, "args", args)
	}

	logger.Debug("Extracted tool parameters",
		"name", toolName,
		"script", script,
		"params", paramsStr)

	return name, script, paramsStr, nil
}

// buildToolURI constructs the URI for tool execution.
func buildToolURI(id, script, paramsStr string) (*url.URL, error) {
	queryParams := url.Values{
		"op":     []string{"run"},
		"script": []string{script},
	}
	if paramsStr != "" {
		queryParams.Add("params", url.QueryEscape(paramsStr))
	}

	uriStr := "tool:/" + url.PathEscape(id) + "?" + queryParams.Encode()
	uri, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tool URI: %w", err)
	}
	return uri, nil
}

// formatToolParameters formats tool parameters for the system prompt.
func formatToolParameters(tool llms.Tool, sb *strings.Builder) {
	if tool.Function == nil || tool.Function.Parameters == nil {
		return
	}
	params, ok := tool.Function.Parameters.(map[string]interface{})
	if !ok {
		return
	}
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		return
	}
	for paramName, param := range props {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}
		desc, _ := paramMap["description"].(string)
		required := ""
		if reqs, ok := params["required"].([]interface{}); ok {
			for _, req := range reqs {
				if req == paramName {
					required = " (required)"
					break
				}
			}
		}
		sb.WriteString("  - " + paramName + ": " + desc + required + "\n")
	}
	sb.WriteString("\n")
}

// processToolCalls processes tool calls, appends results to messageHistory, and stores outputs.
func processToolCalls(toolCalls []llms.ToolCall, toolreader *tool.PklResourceReader, chatBlock *pklLLM.ResourceChat, logger *logging.Logger, messageHistory *[]llms.MessageContent, originalPrompt string, toolOutputs map[string]string) error {
	if len(toolCalls) == 0 {
		logger.Info("No tool calls to process")
		return nil
	}

	var errorMessages []error
	successfulCalls := 0

	// Add original prompt to message history
	*messageHistory = append(*messageHistory, llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: "Original Prompt: " + originalPrompt}},
	})

	// Add AI response with suggested tools
	var toolNames []string
	for _, tc := range toolCalls {
		if tc.FunctionCall != nil {
			toolNames = append(toolNames, tc.FunctionCall.Name)
		}
	}
	if len(toolNames) > 0 {
		*messageHistory = append(*messageHistory, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextContent{Text: "AI Suggested Tools: " + strings.Join(toolNames, ", ")}},
		})
	}

	for _, tc := range toolCalls {
		if tc.FunctionCall == nil || tc.FunctionCall.Name == "" {
			logger.Warn("Skipping tool call with empty function name or nil FunctionCall")
			errorMessages = append(errorMessages, errors.New("invalid tool call: empty function name or nil FunctionCall"))
			continue
		}

		logger.Info("Processing tool call",
			"name", tc.FunctionCall.Name,
			"arguments", tc.FunctionCall.Arguments,
			"tool_call_id", tc.ID)

		args, err := parseToolCallArgs(tc.FunctionCall.Arguments, logger)
		if err != nil {
			logger.Error("Failed to parse tool call arguments", "name", tc.FunctionCall.Name, "error", err)
			errorMessages = append(errorMessages, fmt.Errorf("failed to parse arguments for tool %s: %w", tc.FunctionCall.Name, err))
			continue
		}

		id, script, paramsStr, err := extractToolParams(args, chatBlock, tc.FunctionCall.Name, logger)
		if err != nil {
			logger.Error("Failed to extract tool parameters", "name", tc.FunctionCall.Name, "error", err)
			errorMessages = append(errorMessages, fmt.Errorf("failed to extract parameters for tool %s: %w", tc.FunctionCall.Name, err))
			continue
		}

		uri, err := buildToolURI(id, script, paramsStr)
		if err != nil {
			logger.Error("Failed to build tool URI", "name", tc.FunctionCall.Name, "error", err)
			errorMessages = append(errorMessages, fmt.Errorf("failed to build URI for tool %s: %w", tc.FunctionCall.Name, err))
			continue
		}

		logger.Info("Executing tool",
			"name", tc.FunctionCall.Name,
			"uri", uri.String())

		result, err := toolreader.Read(*uri)
		if err != nil {
			logger.Error("Tool execution failed", "name", tc.FunctionCall.Name, "uri", uri.String(), "error", err)
			errorMessages = append(errorMessages, fmt.Errorf("tool execution failed for %s: %w", tc.FunctionCall.Name, err))
			continue
		}

		resultStr := string(result)
		logger.Info("Tool execution succeeded",
			"name", tc.FunctionCall.Name,
			"result_length", len(resultStr),
			"result_preview", utils.TruncateString(resultStr, 100))

		// Store tool output
		toolOutputs[tc.ID] = resultStr

		// Add tool execution message to history
		toolExecutionMessage := "Tool '" + tc.FunctionCall.Name + "' executed with arguments: " + tc.FunctionCall.Arguments + "\nOutput: " + resultStr
		*messageHistory = append(*messageHistory, llms.MessageContent{
			Role:  llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{llms.TextContent{Text: toolExecutionMessage}},
		})

		// Add tool response to history
		toolResponseJSON, err := json.Marshal(map[string]interface{}{
			"tool_call_id": tc.ID,
			"name":         tc.FunctionCall.Name,
			"content":      resultStr,
			"status":       "completed",
		})
		if err != nil {
			logger.Error("Failed to serialize ToolCallResponse to JSON", "tool_call_id", tc.ID, "error", err)
			errorMessages = append(errorMessages, fmt.Errorf("failed to serialize ToolCallResponse for %s: %w", tc.FunctionCall.Name, err))
			continue
		}

		toolResponse := llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.TextContent{
					Text: "ToolCallResponse: " + string(toolResponseJSON),
				},
			},
		}
		*messageHistory = append(*messageHistory, toolResponse)
		successfulCalls++
	}

	if len(errorMessages) > 0 {
		logger.Warn("Some tool calls failed",
			"error_count", len(errorMessages),
			"successful_calls", successfulCalls)
		return errors.Join(errorMessages...)
	}

	logger.Info("Processed tool calls",
		"total_calls", len(toolCalls),
		"successful_calls", successfulCalls,
		"failed_calls", len(errorMessages))
	return nil
}

// parseToolCallArgs parses JSON arguments from a tool call.
func parseToolCallArgs(arguments string, logger *logging.Logger) (map[string]interface{}, error) {
	args := make(map[string]interface{})
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		logger.Error("Failed to parse tool call arguments",
			"arguments", arguments,
			"error", err)
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	logger.Debug("Parsed tool arguments", "args", args)
	return args, nil
}

// encodeTools encodes the Tools field.
func encodeTools(tools *[]pklLLM.Tool) []pklLLM.Tool {
	encodedEntries := make([]pklLLM.Tool, len(*tools))
	for i, entry := range *tools {
		// Tool is a struct, not a pointer, so we can always access it
		encodedName := utils.EncodeValue(utils.SafeDerefString(entry.Name))
		encodedScript := utils.EncodeValue(utils.SafeDerefString(entry.Script))
		encodedDescription := utils.EncodeValue(utils.SafeDerefString(entry.Description))

		var encodedParameters *map[string]pklLLM.ToolProperties
		if entry.Parameters != nil {
			params := encodeToolParameters(entry.Parameters)
			encodedParameters = params
		}

		encodedEntries[i] = pklLLM.Tool{
			Name:        &encodedName,
			Script:      &encodedScript,
			Description: &encodedDescription,
			Parameters:  encodedParameters,
		}
	}
	return encodedEntries
}

// encodeToolParameters encodes tool parameters.
func encodeToolParameters(params *map[string]pklLLM.ToolProperties) *map[string]pklLLM.ToolProperties {
	encodedParams := make(map[string]pklLLM.ToolProperties, len(*params))
	for paramName, param := range *params {
		// ToolProperties is a struct, not a pointer, so we can always access it
		encodedType := utils.EncodeValue(utils.SafeDerefString(param.Type))
		encodedDescription := utils.EncodeValue(utils.SafeDerefString(param.Description))
		encodedParams[paramName] = pklLLM.ToolProperties{
			Required:    param.Required,
			Type:        &encodedType,
			Description: &encodedDescription,
		}
	}
	return &encodedParams
}

// extractToolNames extracts tool names from a slice of llms.ToolCall for logging.
func extractToolNames(toolCalls []llms.ToolCall) []string {
	names := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc.FunctionCall != nil && tc.FunctionCall.Name != "" {
			names = append(names, tc.FunctionCall.Name)
		}
	}
	return names
}

// extractToolNamesFromTools extracts tool names from a slice of llms.Tool for logging.
func extractToolNamesFromTools(tools []llms.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		if t.Function != nil {
			names = append(names, t.Function.Name)
		}
	}
	return names
}

// deduplicateToolCalls removes duplicate tool calls based on name and arguments.
func deduplicateToolCalls(toolCalls []llms.ToolCall, logger *logging.Logger) []llms.ToolCall {
	seen := make(map[string]struct{})
	result := make([]llms.ToolCall, 0, len(toolCalls))

	for _, tc := range toolCalls {
		if tc.FunctionCall == nil {
			logger.Warn("Skipping tool call with nil FunctionCall")
			continue
		}
		key := tc.FunctionCall.Name + ":" + tc.FunctionCall.Arguments
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, tc)
		} else {
			logger.Info("Removed duplicate tool call", "name", tc.FunctionCall.Name, "arguments", tc.FunctionCall.Arguments)
		}
	}
	return result
}

// convertToolParamsToString converts a value to a string, handling different types.
func convertToolParamsToString(value interface{}, paramName, toolName string, logger *logging.Logger) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		jsonVal, err := json.Marshal(v)
		if err != nil {
			logger.Warn("Failed to serialize parameter", "tool", toolName, "paramName", paramName, "error", err)
			return ""
		}
		return string(jsonVal)
	}
}

// serializeTools serializes the Tools field to Pkl format.
func serializeTools(builder *strings.Builder, tools *[]pklLLM.Tool) {
	builder.WriteString("    Tools ")
	if tools == nil || len(*tools) == 0 {
		builder.WriteString("{}\n")
		return
	}

	builder.WriteString("{\n")
	for _, entry := range *tools {
		// Tool is a struct, not a pointer, so we can always access it
		builder.WriteString("      new {\n")
		name := ""
		if entry.Name != nil {
			name = *entry.Name
		}
		builder.WriteString(fmt.Sprintf("        Name = %q\n", name))
		script := ""
		if entry.Script != nil {
			script = *entry.Script
		}
		builder.WriteString(fmt.Sprintf("        Script = #\"\"\"\n%s\n\"\"\"#\n", script))
		description := ""
		if entry.Description != nil {
			description = *entry.Description
		}
		builder.WriteString(fmt.Sprintf("        Description = %q\n", description))
		builder.WriteString("        Parameters ")
		if entry.Parameters != nil && len(*entry.Parameters) > 0 {
			builder.WriteString("{\n")
			for pname, param := range *entry.Parameters {
				// ToolProperties is a struct, not a pointer, so we can always access it
				builder.WriteString(fmt.Sprintf("          [\"%s\"] {\n", pname))
				required := false
				if param.Required != nil {
					required = *param.Required
				}
				builder.WriteString(fmt.Sprintf("            Required = %t\n", required))
				paramType := ""
				if param.Type != nil {
					paramType = *param.Type
				}
				builder.WriteString(fmt.Sprintf("            Type = %q\n", paramType))
				paramDescription := ""
				if param.Description != nil {
					paramDescription = *param.Description
				}
				builder.WriteString(fmt.Sprintf("            Description = %q\n", paramDescription))
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
