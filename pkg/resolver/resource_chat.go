package resolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/gabriel-vasile/mimetype"
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

// HandleLLMChat processes an LLM chat interaction synchronously.
func (dr *DependencyResolver) HandleLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	dr.Logger.Info("HandleLLMChat: ENTRY", "actionID", actionID, "chatBlock_nil", chatBlock == nil)
	if chatBlock != nil {
		dr.Logger.Info("HandleLLMChat: chatBlock fields", "actionID", actionID, "model", chatBlock.Model, "prompt_nil", chatBlock.Prompt == nil)
	}
	dr.Logger.Debug("HandleLLMChat: called", "actionID", actionID, "PklresHelper_nil", dr.PklresHelper == nil, "PklresReader_nil", dr.PklresHelper == nil || dr.PklresHelper.resolver == nil || dr.PklresHelper.resolver.PklresReader == nil)
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Decode the chat block synchronously
	if err := dr.decodeChatBlock(chatBlock); err != nil {
		dr.Logger.Error("failed to decode chat block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Process the chat block synchronously
	if err := dr.processLLMChat(canonicalActionID, chatBlock); err != nil {
		dr.Logger.Error("failed to process LLM chat block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// generateChatResponse generates a response from the LLM based on the chat block, executing tools via toolreader.
func generateChatResponse(ctx context.Context, fs afero.Fs, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, toolreader *tool.PklResourceReader, logger *logging.Logger) (string, error) {
	logger.Info("Processing chatBlock",
		"model", chatBlock.Model,
		"prompt", utils.SafeDerefString(chatBlock.Prompt),
		"role", utils.SafeDerefString(chatBlock.Role),
		"json_response", utils.SafeDerefBool(chatBlock.JSONResponse),
		"json_response_keys", utils.SafeDerefSlice(chatBlock.JSONResponseKeys),
		"tool_count", len(utils.SafeDerefSlice(chatBlock.Tools)),
		"scenario_count", len(utils.SafeDerefSlice(chatBlock.Scenario)),
		"file_count", len(utils.SafeDerefSlice(chatBlock.Files)))

	// Generate dynamic tools with enhanced logging
	availableTools := generateAvailableTools(chatBlock, logger)
	logger.Info("Generated tools",
		"tool_count", len(availableTools),
		"tool_names", extractToolNamesFromTools(availableTools))

	// Build message history
	messageHistory := make([]llms.MessageContent, 0)

	// Store tool outputs to influence subsequent calls
	toolOutputs := make(map[string]string) // Key: tool_call_id, Value: output

	// Build system prompt that encourages tool usage and considers previous outputs
	systemPrompt := buildSystemPrompt(chatBlock.JSONResponse, chatBlock.JSONResponseKeys, availableTools)
	logger.Info("Generated system prompt", "content", utils.TruncateString(systemPrompt, 200))

	messageHistory = append(messageHistory, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})

	// Add main prompt if present
	role, roleType := getRoleAndType(chatBlock.Role)
	prompt := utils.SafeDerefString(chatBlock.Prompt)
	if strings.TrimSpace(prompt) != "" {
		if roleType == llms.ChatMessageTypeGeneric {
			prompt = "[" + role + "]: " + prompt
		}
		messageHistory = append(messageHistory, llms.MessageContent{
			Role:  roleType,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		})
	}

	// Add scenario messages
	messageHistory = append(messageHistory, processScenarioMessages(chatBlock.Scenario, logger)...)

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

			// Add binary content directly instead of base64-encoded text
			messageHistory = append(messageHistory, llms.MessageContent{
				Role: roleType,
				Parts: []llms.ContentPart{
					llms.BinaryPart(fileType, fileBytes),
				},
			})
		}
	}

	// Call options
	opts := []llms.CallOption{}
	if chatBlock.JSONResponse != nil && *chatBlock.JSONResponse {
		opts = append(opts, llms.WithJSONMode())
	}
	if len(availableTools) > 0 {
		opts = append(opts,
			llms.WithTools(availableTools),
			llms.WithJSONMode(),
			llms.WithToolChoice("auto"))
	}

	logger.Info("Calling LLM with options",
		"json_mode", utils.SafeDerefBool(chatBlock.JSONResponse),
		"tool_count", len(availableTools))

	// First GenerateContent call
	response, err := llm.GenerateContent(ctx, messageHistory, opts...)
	if err != nil {
		logger.Error("Failed to generate content in first call", "error", err)
		return "", fmt.Errorf("failed to generate content in first call: %w", err)
	}

	if len(response.Choices) == 0 {
		logger.Error("No choices in LLM response")
		return "", errors.New("no choices in LLM response")
	}

	// Select choice with tool calls, if any
	var respChoice *llms.ContentChoice
	if len(availableTools) > 0 {
		for _, choice := range response.Choices {
			if len(choice.ToolCalls) > 0 {
				respChoice = choice
				break
			}
		}
	}
	if respChoice == nil && len(response.Choices) > 0 {
		respChoice = response.Choices[0]
	}

	logger.Info("First LLM response",
		"content", utils.TruncateString(respChoice.Content, 100),
		"tool_calls", len(respChoice.ToolCalls),
		"stop_reason", respChoice.StopReason,
		"tool_names", extractToolNames(respChoice.ToolCalls))

	// Process first response
	toolCalls := respChoice.ToolCalls
	if len(toolCalls) == 0 && len(availableTools) > 0 {
		logger.Info("No direct ToolCalls, attempting to construct from JSON")
		constructedToolCalls := constructToolCallsFromJSON(respChoice.Content, logger)
		toolCalls = constructedToolCalls
	}

	// Deduplicate tool calls
	toolCalls = deduplicateToolCalls(toolCalls, logger)

	// Add response to history
	assistantParts := []string{}
	if respChoice.Content != "" {
		assistantParts = append(assistantParts, respChoice.Content)
	}
	for _, tc := range toolCalls {
		toolCallJSON, err := json.Marshal(map[string]interface{}{
			"id":   tc.ID,
			"type": tc.Type,
			"function": map[string]interface{}{
				"name":      tc.FunctionCall.Name,
				"arguments": tc.FunctionCall.Arguments,
			},
		})
		if err != nil {
			logger.Error("Failed to serialize ToolCall to JSON", "tool_call_id", tc.ID, "error", err)
			continue
		}
		assistantParts = append(assistantParts, "ToolCall: "+string(toolCallJSON))
	}

	if len(toolCalls) > 0 {
		toolNames := extractToolNames(toolCalls)
		assistantParts = append(assistantParts, "Suggested tools: "+strings.Join(toolNames, ", "))
	}

	assistantContent := strings.Join(assistantParts, "\n")
	if assistantContent != "" {
		messageHistory = append(messageHistory, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextContent{Text: assistantContent}},
		})
	}

	// Track tool calls to prevent duplicates and looping
	toolCallHistory := make(map[string]int)
	const maxIterations = 5 // Allow more iterations to process chained tool calls

	// Process tool calls iteratively
	for iteration := 0; len(toolCalls) > 0 && iteration < maxIterations; iteration++ {
		logger.Info("Processing tool calls",
			"iteration", iteration+1,
			"count", len(toolCalls),
			"tool_names", extractToolNames(toolCalls))

		err = processToolCalls(toolCalls, toolreader, chatBlock, logger, &messageHistory, prompt, toolOutputs)
		if err != nil {
			logger.Error("Failed to process tool calls", "iteration", iteration+1, "error", err)
			return "", fmt.Errorf("failed to process tool calls in iteration %d: %w", iteration+1, err)
		}

		// Include tool outputs in the system prompt for the next call
		systemPrompt = buildSystemPrompt(chatBlock.JSONResponse, chatBlock.JSONResponseKeys, availableTools)
		if len(toolOutputs) > 0 {
			var toolOutputSummary strings.Builder
			toolOutputSummary.WriteString("\nPrevious Tool Outputs:\n")
			for toolID, output := range toolOutputs {
				toolOutputSummary.WriteString("- ToolCall ID " + toolID + ": " + utils.TruncateString(output, 100) + "\n")
			}
			systemPrompt += toolOutputSummary.String()
		}

		// Update system message in history
		messageHistory[0] = llms.MessageContent{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
		}

		// Generate content with updated history
		logger.Debug("Message history before LLM call", "iteration", iteration+1, "history", summarizeMessageHistory(messageHistory))
		response, err = llm.GenerateContent(ctx, messageHistory, opts...)
		if err != nil {
			logger.Error("Failed to generate content", "iteration", iteration+1, "error", err)
			return "", fmt.Errorf("failed to generate content in iteration %d: %w", iteration+1, err)
		}

		if len(response.Choices) == 0 {
			logger.Error("No choices in LLM response", "iteration", iteration+1)
			return "", errors.New("no choices in LLM response")
		}

		// Select choice with tool calls, if any
		respChoice = nil
		for _, choice := range response.Choices {
			if len(choice.ToolCalls) > 0 {
				respChoice = choice
				break
			}
		}
		if respChoice == nil && len(response.Choices) > 0 {
			respChoice = response.Choices[0]
		}

		logger.Info("LLM response",
			"iteration", iteration+1,
			"content", utils.TruncateString(respChoice.Content, 100),
			"tool_calls", len(respChoice.ToolCalls),
			"stop_reason", respChoice.StopReason,
			"tool_names", extractToolNames(respChoice.ToolCalls))

		// Check for tool calls
		toolCalls = respChoice.ToolCalls
		if len(toolCalls) == 0 && len(availableTools) > 0 {
			logger.Info("No direct ToolCalls, attempting to construct from JSON", "iteration", iteration+1)
			constructedToolCalls := constructToolCallsFromJSON(respChoice.Content, logger)
			toolCalls = constructedToolCalls
		}

		// Deduplicate tool calls
		toolCalls = deduplicateToolCalls(toolCalls, logger)

		// Exit if no new tool calls or LLM stopped
		if len(toolCalls) == 0 || respChoice.StopReason == "stop" {
			logger.Info("No valid tool calls or LLM stopped, returning response", "iteration", iteration+1, "content", utils.TruncateString(respChoice.Content, 100))
			// If response is empty, use the last tool output
			if respChoice.Content == "{}" || respChoice.Content == "" {
				logger.Warn("Empty response detected, falling back to last tool output")
				for _, output := range toolOutputs {
					respChoice.Content = output
				}
				if respChoice.Content == "" {
					logger.Error("No tool outputs available, returning default response")
					respChoice.Content = "No result available"
				}
			}
			logger.Info("Final response", "content", utils.TruncateString(respChoice.Content, 100))
			return respChoice.Content, nil
		}

		// Check for repeated tool calls
		for _, tc := range toolCalls {
			if tc.FunctionCall != nil {
				// Normalize arguments
				argsMap := make(map[string]interface{})
				if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &argsMap); err != nil {
					logger.Warn("Failed to normalize tool arguments", "tool", tc.FunctionCall.Name, "error", err)
					continue
				}
				normalizedArgs, err := json.Marshal(argsMap)
				if err != nil {
					logger.Warn("Failed to normalize tool arguments", "tool", tc.FunctionCall.Name, "error", err)
					continue
				}
				toolKey := tc.FunctionCall.Name + ":" + string(normalizedArgs)
				toolCallHistory[toolKey]++
				if toolCallHistory[toolKey] > 1 {
					logger.Info("Detected repeated tool call, returning response",
						"tool", tc.FunctionCall.Name,
						"arguments", tc.FunctionCall.Arguments,
						"count", toolCallHistory[toolKey])
					// Use last tool output if available
					for _, output := range toolOutputs {
						logger.Info("Final response from repeated tool call", "content", utils.TruncateString(output, 100))
						return output, nil
					}
					return respChoice.Content, nil
				}
			}
		}

		// Add response to history
		assistantParts = []string{}
		if respChoice.Content != "" {
			assistantParts = append(assistantParts, respChoice.Content)
		}
		for _, tc := range toolCalls {
			toolCallJSON, err := json.Marshal(map[string]interface{}{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]interface{}{
					"name":      tc.FunctionCall.Name,
					"arguments": tc.FunctionCall.Arguments,
				},
			})
			if err != nil {
				logger.Error("Failed to serialize ToolCall to JSON", "tool_call_id", tc.ID, "error", err)
				continue
			}
			assistantParts = append(assistantParts, "ToolCall: "+string(toolCallJSON))
		}

		if len(toolCalls) > 0 {
			toolNames := extractToolNames(toolCalls)
			assistantParts = append(assistantParts, "Suggested tools: "+strings.Join(toolNames, ", "))
		}

		assistantContent = strings.Join(assistantParts, "\n")
		if assistantContent != "" {
			messageHistory = append(messageHistory, llms.MessageContent{
				Role:  llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{llms.TextContent{Text: assistantContent}},
			})
		}

		if iteration == maxIterations-1 && len(toolCalls) > 0 {
			logger.Error("Reached maximum tool call iterations", "max_iterations", maxIterations)
			// Return last tool output if available
			for _, output := range toolOutputs {
				logger.Info("Final response from max iterations", "content", utils.TruncateString(output, 100))
				return output, nil
			}
			return respChoice.Content, fmt.Errorf("reached maximum tool call iterations (%d)", maxIterations)
		}
	}

	logger.Info("Received final LLM response", "content", utils.TruncateString(respChoice.Content, 100))
	// Ensure non-empty response
	if respChoice.Content == "{}" || respChoice.Content == "" {
		logger.Warn("Empty response detected, falling back to last tool output")
		for _, output := range toolOutputs {
			respChoice.Content = output
		}
		if respChoice.Content == "" {
			logger.Error("No tool outputs available, returning default response")
			respChoice.Content = "No result available"
		}
	}
	logger.Info("Final response", "content", utils.TruncateString(respChoice.Content, 100))
	return respChoice.Content, nil
}

// processLLMChat processes the LLM chat and saves the response.
func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	dr.Logger.Info("processLLMChat: called", "actionID", actionID, "PklresHelper_nil", dr.PklresHelper == nil, "PklresReader_nil", dr.PklresHelper == nil || dr.PklresHelper.resolver == nil || dr.PklresHelper.resolver.PklresReader == nil)
	dr.Logger.Info("processLLMChat: starting", "actionID", actionID)

	if dr.NewLLMFn == nil {
		dr.Logger.Error("processLLMChat: NewLLMFn is nil!", "actionID", actionID)
		return errors.New("NewLLMFn is nil")
	}
	if dr.GenerateChatResponseFn == nil {
		dr.Logger.Error("processLLMChat: GenerateChatResponseFn is nil!", "actionID", actionID)
		return errors.New("GenerateChatResponseFn is nil")
	}

	if chatBlock == nil {
		dr.Logger.Error("processLLMChat: chatBlock is nil", "actionID", actionID)
		return errors.New("chatBlock cannot be nil")
	}

	dr.Logger.Debug("processLLMChat: initializing LLM", "actionID", actionID, "model", chatBlock.Model)
	llm, err := dr.NewLLMFn(chatBlock.Model)
	if err != nil {
		dr.Logger.Error("processLLMChat: failed to initialize LLM", "actionID", actionID, "error", err)
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	dr.Logger.Debug("processLLMChat: generating chat response", "actionID", actionID)
	completion, err := dr.GenerateChatResponseFn(dr.Context, dr.Fs, llm, chatBlock, dr.ToolReader, dr.Logger)
	if err != nil {
		dr.Logger.Error("processLLMChat: failed to generate chat response", "actionID", actionID, "error", err)
		return err
	}

	dr.Logger.Info("processLLMChat: setting response", "actionID", actionID, "responseLength", len(completion))
	chatBlock.Response = &completion

	// Write the LLM response to output file for pklres access
	if chatBlock.Response != nil {
		dr.Logger.Debug("processLLMChat: writing response to file", "actionID", actionID)
		filePath, err := dr.WriteResponseToFile(actionID, chatBlock.Response)
		if err != nil {
			dr.Logger.Error("processLLMChat: failed to write response to file", "actionID", actionID, "error", err)
			return fmt.Errorf("failed to write response to file: %w", err)
		}
		chatBlock.File = &filePath
		dr.Logger.Debug("processLLMChat: wrote response to file", "actionID", actionID, "filePath", filePath)
	}

	// Set timestamp after processing is complete
	ts := pkl.Duration{
		Value: float64(time.Now().UnixNano()),
		Unit:  pkl.Nanosecond,
	}
	chatBlock.Timestamp = &ts

	// Store the LLM resource data in pklres for real-time access
	if dr.PklresHelper != nil {
		// Create a ResourceChat object for storage
		resourceChat := &pklLLM.ResourceChat{
			Model:        chatBlock.Model,
			Role:         chatBlock.Role,
			Prompt:       chatBlock.Prompt,
			Response:     chatBlock.Response,
			File:         chatBlock.File,
			JSONResponse: chatBlock.JSONResponse,
			Timestamp:    chatBlock.Timestamp,
		}

		// Store the resource object using the new method
		if err := dr.PklresHelper.StoreResourceObject("llm", actionID, resourceChat); err != nil {
			dr.Logger.Error("processLLMChat: failed to store LLM resource in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("processLLMChat: stored LLM resource in pklres", "actionID", actionID)
		}
	}

	// Mark the resource as finished processing
	if err := dr.MarkResourceFinished(actionID); err != nil {
		dr.Logger.Warn("processLLMChat: failed to mark resource as finished", "actionID", actionID, "error", err)
	}

	dr.Logger.Info("processLLMChat: completed successfully", "actionID", actionID)
	return nil
}

// generatePklContent generates Pkl content from resources.
func generatePklContent(resources map[string]*pklLLM.ResourceChat, ctx context.Context, logger *logging.Logger, requestID string) string {
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"%s\"\n\n", schema.ImportPath(ctx, "LLM.pkl")))
	pklContent.WriteString("Resources {\n")

	for id, res := range resources {
		logger.Info("Generating PKL for resource", "id", id)
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    Model = %q\n", res.Model))

		prompt := ""
		if res.Prompt != nil {
			prompt = *res.Prompt
		}
		pklContent.WriteString(fmt.Sprintf("    Prompt = %q\n", prompt))

		role := RoleHuman
		if res.Role != nil && *res.Role != "" {
			role = *res.Role
		}
		pklContent.WriteString(fmt.Sprintf("    Role = %q\n", role))

		pklContent.WriteString("    Scenario ")
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
				pklContent.WriteString(fmt.Sprintf("        Role = %q\n", entryRole))
				entryPrompt := ""
				if entry.Prompt != nil {
					entryPrompt = *entry.Prompt
				}
				pklContent.WriteString(fmt.Sprintf("        Prompt = %q\n", entryPrompt))
				logger.Info("Serialized scenario entry", "index", i, "role", entryRole, "prompt", entryPrompt)
				pklContent.WriteString("      }\n")
			}
			pklContent.WriteString("    }\n")
		} else {
			logger.Info("Scenario is nil or empty in generatePklContent")
			pklContent.WriteString("{}\n")
		}

		serializeTools(&pklContent, res.Tools)

		jsonResponse := false
		if res.JSONResponse != nil {
			jsonResponse = *res.JSONResponse
		}
		pklContent.WriteString(fmt.Sprintf("    JSONResponse = %t\n", jsonResponse))

		pklContent.WriteString("    JSONResponseKeys ")
		if res.JSONResponseKeys != nil && len(*res.JSONResponseKeys) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.JSONResponseKeys))
		} else {
			pklContent.WriteString("{}\n")
		}

		pklContent.WriteString("    Files ")
		if res.Files != nil && len(*res.Files) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.Files))
		} else {
			pklContent.WriteString("{}\n")
		}

		timeoutValue := 60.0
		timeoutUnit := pkl.Second
		if res.TimeoutDuration != nil {
			timeoutValue = res.TimeoutDuration.Value
			timeoutUnit = res.TimeoutDuration.Unit
		}
		pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %g.%s\n", timeoutValue, timeoutUnit.String()))

		timestampValue := float64(time.Now().Unix())
		timestampUnit := pkl.Nanosecond
		if res.Timestamp != nil {
			timestampValue = res.Timestamp.Value
			timestampUnit = res.Timestamp.Unit
		}
		pklContent.WriteString(fmt.Sprintf("    Timestamp = %g.%s\n", timestampValue, timestampUnit.String()))

		if res.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    Response = #\"\"\"\n%s\n\"\"\"#\n", *res.Response))
		} else {
			pklContent.WriteString("    Response = \"\"\n")
		}

		if res.File != nil {
			pklContent.WriteString(fmt.Sprintf("    File = %q\n", *res.File))
		} else {
			pklContent.WriteString("    File = \"\"\n")
		}

		// Add ItemValues
		pklContent.WriteString("    ItemValues ")
		if res.ItemValues != nil && len(*res.ItemValues) > 0 {
			pklContent.WriteString(utils.EncodePklSlice(res.ItemValues))
		} else {
			pklContent.WriteString("{}\n")
		}

		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	return pklContent.String()
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

// EncodeChat encodes a chat block for LLM processing.
func EncodeChat(chat *pklLLM.ResourceChat, logger *logging.Logger) string {
	if chat == nil {
		return ""
	}

	// Use the private encodeChat function to encode the chat
	encodedChat := encodeChat(chat, logger)

	// Convert the encoded chat to a string representation
	var result strings.Builder

	// Add model
	result.WriteString(fmt.Sprintf("Model = %s\n", utils.EncodeValue(encodedChat.Model)))

	// Add prompt
	if encodedChat.Prompt != nil {
		result.WriteString(fmt.Sprintf("Prompt = %s\n", *encodedChat.Prompt))
	}

	// Add role
	if encodedChat.Role != nil {
		result.WriteString(fmt.Sprintf("Role = %s\n", *encodedChat.Role))
	}

	// Add scenario
	if encodedChat.Scenario != nil && len(*encodedChat.Scenario) > 0 {
		result.WriteString("Scenario {\n")
		for _, entry := range *encodedChat.Scenario {
			if entry != nil {
				if entry.Role != nil {
					result.WriteString(fmt.Sprintf("  Role = %s\n", *entry.Role))
				}
				if entry.Prompt != nil {
					result.WriteString(fmt.Sprintf("  Prompt = %s\n", *entry.Prompt))
				}
			}
		}
		result.WriteString("}\n")
	}

	// Add tools
	if encodedChat.Tools != nil && len(*encodedChat.Tools) > 0 {
		result.WriteString("Tools {\n")
		for _, tool := range *encodedChat.Tools {
			if tool != nil {
				if tool.Name != nil {
					result.WriteString(fmt.Sprintf("  Name = %s\n", *tool.Name))
				}
				if tool.Script != nil {
					result.WriteString(fmt.Sprintf("  Script = %s\n", *tool.Script))
				}
				if tool.Parameters != nil {
					result.WriteString("  Parameters {\n")
					for paramName, param := range *tool.Parameters {
						if param != nil {
							result.WriteString(fmt.Sprintf("    [%s] {\n", utils.EncodeValue(paramName)))
							if param.Type != nil {
								result.WriteString(fmt.Sprintf("      Type = %s\n", *param.Type))
							}
							if param.Description != nil {
								result.WriteString(fmt.Sprintf("      Description = %s\n", *param.Description))
							}
							result.WriteString("    }\n")
						}
					}
					result.WriteString("  }\n")
				}
			}
		}
		result.WriteString("}\n")
	}

	// Add files
	if encodedChat.Files != nil && len(*encodedChat.Files) > 0 {
		result.WriteString("Files {\n")
		for _, file := range *encodedChat.Files {
			result.WriteString(fmt.Sprintf("  %s\n", file))
		}
		result.WriteString("}\n")
	}

	// Add timeout
	if encodedChat.TimeoutDuration != nil {
		result.WriteString(fmt.Sprintf("TimeoutDuration = %g.%s\n", encodedChat.TimeoutDuration.Value, encodedChat.TimeoutDuration.Unit.String()))
	}

	// Add timestamp
	if encodedChat.Timestamp != nil {
		result.WriteString(fmt.Sprintf("Timestamp = %g.%s\n", encodedChat.Timestamp.Value, encodedChat.Timestamp.Unit.String()))
	}

	return result.String()
}

// EncodeJSONResponseKeys encodes JSON response keys.
func EncodeJSONResponseKeys(keys *[]string) *[]string {
	if keys == nil {
		return nil
	}
	encoded := make([]string, len(*keys))
	for i, v := range *keys {
		encoded[i] = utils.EncodeValue(v)
	}
	return &encoded
}

// Exported for testing
var GenerateChatResponse = generateChatResponse

// Exported for testing
var GeneratePklContent = generatePklContent
