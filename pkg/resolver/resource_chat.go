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
	pklResource "github.com/kdeps/schema/gen/resource"
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

// substituteChatBlockTemplates substitutes template placeholders in chat block with actual values at runtime
func (dr *DependencyResolver) substituteChatBlockTemplates(actionID string, chatBlock *pklLLM.ResourceChat) error {
	// Create a mapping of template variables to their actual values
	templateVars := make(map[string]string)

	// Get request parameter 'q' if available
	if dr.Request != nil {
		if q := dr.Request.Query("q"); q != "" {
			templateVars["REQUEST_PARAM_Q"] = q
		}
	}

	// Substitute templates in the prompt
	if chatBlock.Prompt != nil {
		newPrompt := *chatBlock.Prompt
		for placeholder, value := range templateVars {
			newPrompt = strings.ReplaceAll(newPrompt, "{"+placeholder+"}", value)
		}
		chatBlock.Prompt = &newPrompt
	}

	// Substitute templates in scenario prompts
	if chatBlock.Scenario != nil {
		for _, scenario := range *chatBlock.Scenario {
			if scenario != nil && scenario.Prompt != nil {
				newPrompt := *scenario.Prompt
				for placeholder, value := range templateVars {
					newPrompt = strings.ReplaceAll(newPrompt, "{"+placeholder+"}", value)
				}
				scenario.Prompt = &newPrompt
			}
		}
	}

	return nil
}

// getResourceOutputSafely safely retrieves resource output without causing circular dependencies
func (dr *DependencyResolver) getResourceOutputSafely(resourceID, resourceType, field string) string {
	if dr.PklresHelper == nil {
		return ""
	}

	// Resolve the resource ID to canonical form
	canonicalID := dr.PklresHelper.resolveActionID(resourceID)

	// Try to get the resource data from pklres
	// This should only work if the resource has already been processed
	switch field {
	case "Body":
		if response, err := dr.PklresHelper.Get(canonicalID, "response"); err == nil && response != "" {
			return response
		}
	case "Stdout":
		if stdout, err := dr.PklresHelper.Get(canonicalID, "stdout"); err == nil && stdout != "" {
			return stdout
		}
	}

	return ""
}

// HandleLLMChat processes an LLM chat interaction synchronously.
func (dr *DependencyResolver) HandleLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	// LLM Chat processing started for actionID
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			// ActionID canonicalized
		}
	}

	// Decode the chat block synchronously
	if err := dr.decodeChatBlock(chatBlock); err != nil {
		dr.Logger.Error("failed to decode chat block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Reload the LLM resource to ensure PKL templates are evaluated after dependencies are processed
	// This ensures that PKL template expressions like \(client.responseBody("clientResource")) have access to dependency data
	if err := dr.reloadLLMResourceWithDependencies(canonicalActionID, chatBlock); err != nil {
		dr.Logger.Warn("failed to reload LLM resource, continuing with original", "actionID", canonicalActionID, "error", err)
	}

	// Process the chat block synchronously
	if err := dr.processLLMChat(canonicalActionID, chatBlock); err != nil {
		dr.Logger.Error("failed to process LLM chat block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// reloadLLMResourceWithDependencies reloads the LLM resource to ensure PKL templates are evaluated after dependencies
func (dr *DependencyResolver) reloadLLMResourceWithDependencies(actionID string, chatBlock *pklLLM.ResourceChat) error {
	// Reloading LLM resource with fresh template evaluation

	// Find the resource file path for this actionID
	resourceFile := ""
	for _, resInterface := range dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
			if res.ActionID == actionID {
				resourceFile = res.File
				break
			}
		}
	}

	if resourceFile == "" {
		return fmt.Errorf("could not find resource file for actionID: %s", actionID)
	}

	// Found resource file for reloading

	// Reload the LLM resource with fresh PKL template evaluation
	// Load as generic Resource since the LLM resource extends Resource.pkl, not LLM.pkl
	var reloadedResource interface{}
	var err error
	if dr.APIServerMode {
		reloadedResource, err = dr.LoadResourceWithRequestContextFn(dr.Context, resourceFile, Resource)
	} else {
		reloadedResource, err = dr.LoadResourceFn(dr.Context, resourceFile, Resource)
	}

	if err != nil {
		return fmt.Errorf("failed to reload LLM resource: %w", err)
	}

	// Cast to generic Resource first
	reloadedGenericResource, ok := reloadedResource.(pklResource.Resource)
	if !ok {
		return fmt.Errorf("failed to cast reloaded resource to generic Resource")
	}

	// Extract the Chat block from the reloaded resource
	if reloadedRun := reloadedGenericResource.GetRun(); reloadedRun != nil && reloadedRun.Chat != nil {
		reloadedChat := reloadedRun.Chat

		// Update the chatBlock with the reloaded values that contain fresh template evaluation
		if reloadedChat.Prompt != nil {
			chatBlock.Prompt = reloadedChat.Prompt
			// Updated prompt from reloaded resource
		}

		if reloadedChat.Scenario != nil {
			chatBlock.Scenario = reloadedChat.Scenario
			// Updated scenario from reloaded resource
		}
	}

	dr.Logger.Info("LLM resource reloaded", "actionID", actionID)
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
	// Processing LLM Chat
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

	// Substitute template placeholders with actual values at runtime
	if err := dr.substituteChatBlockTemplates(actionID, chatBlock); err != nil {
		dr.Logger.Error("processLLMChat: failed to substitute templates", "actionID", actionID, "error", err)
		return fmt.Errorf("failed to substitute templates: %w", err)
	}

	modelStr := ""
	if chatBlock.Model != nil {
		modelStr = *chatBlock.Model
	}
	dr.Logger.Debug("processLLMChat: initializing LLM", "actionID", actionID, "model", modelStr)
	llm, err := dr.NewLLMFn(modelStr)
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
		// Store individual attributes as key-value pairs for direct access
		if chatBlock.Model != nil && *chatBlock.Model != "" {
			if err := dr.PklresHelper.Set(actionID, "model", *chatBlock.Model); err != nil {
				dr.Logger.Error("processLLMChat: failed to store model", "actionID", actionID, "error", err)
			}
		}

		if chatBlock.Role != nil && *chatBlock.Role != "" {
			if err := dr.PklresHelper.Set(actionID, "role", *chatBlock.Role); err != nil {
				dr.Logger.Error("processLLMChat: failed to store role", "actionID", actionID, "error", err)
			}
		}

		if chatBlock.Prompt != nil && *chatBlock.Prompt != "" {
			if err := dr.PklresHelper.Set(actionID, "prompt", *chatBlock.Prompt); err != nil {
				dr.Logger.Error("processLLMChat: failed to store prompt", "actionID", actionID, "error", err)
			}
		}

		if chatBlock.Response != nil && *chatBlock.Response != "" {
			if err := dr.PklresHelper.Set(actionID, "response", *chatBlock.Response); err != nil {
				dr.Logger.Error("processLLMChat: failed to store response", "actionID", actionID, "error", err)
			}
		}

		if chatBlock.File != nil && *chatBlock.File != "" {
			if err := dr.PklresHelper.Set(actionID, "file", *chatBlock.File); err != nil {
				dr.Logger.Error("processLLMChat: failed to store file", "actionID", actionID, "error", err)
			}
		}

		if chatBlock.JSONResponse != nil {
			jsonResponseStr := "false"
			if *chatBlock.JSONResponse {
				jsonResponseStr = "true"
			}
			if err := dr.PklresHelper.Set(actionID, "jsonResponse", jsonResponseStr); err != nil {
				dr.Logger.Error("processLLMChat: failed to store jsonResponse", "actionID", actionID, "error", err)
			}
		}

		// Store JSONResponseKeys in pklres for fallback response generation
		if chatBlock.JSONResponseKeys != nil && len(*chatBlock.JSONResponseKeys) > 0 {
			jsonResponseKeysJSON, err := json.Marshal(*chatBlock.JSONResponseKeys)
			if err == nil {
				if err := dr.PklresHelper.Set(actionID, "jsonResponseKeys", string(jsonResponseKeysJSON)); err != nil {
					dr.Logger.Error("processLLMChat: failed to store jsonResponseKeys", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("processLLMChat: failed to marshal jsonResponseKeys", "actionID", actionID, "error", err)
			}
		}

		// Store Scenario as JSON for complex structure
		if chatBlock.Scenario != nil && len(*chatBlock.Scenario) > 0 {
			if scenarioJSON, err := json.Marshal(*chatBlock.Scenario); err == nil {
				if err := dr.PklresHelper.Set(actionID, "scenario", string(scenarioJSON)); err != nil {
					dr.Logger.Error("processLLMChat: failed to store scenario", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("processLLMChat: failed to marshal scenario", "actionID", actionID, "error", err)
			}
		}

		// Store Tools as JSON for complex structure
		if chatBlock.Tools != nil && len(*chatBlock.Tools) > 0 {
			if toolsJSON, err := json.Marshal(*chatBlock.Tools); err == nil {
				if err := dr.PklresHelper.Set(actionID, "tools", string(toolsJSON)); err != nil {
					dr.Logger.Error("processLLMChat: failed to store tools", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("processLLMChat: failed to marshal tools", "actionID", actionID, "error", err)
			}
		}

		// Store Files as JSON for complex structure
		if chatBlock.Files != nil && len(*chatBlock.Files) > 0 {
			if filesJSON, err := json.Marshal(*chatBlock.Files); err == nil {
				if err := dr.PklresHelper.Set(actionID, "files", string(filesJSON)); err != nil {
					dr.Logger.Error("processLLMChat: failed to store files", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("processLLMChat: failed to marshal files", "actionID", actionID, "error", err)
			}
		}

		// Store ItemValues as JSON for complex structure
		if chatBlock.ItemValues != nil && len(*chatBlock.ItemValues) > 0 {
			if itemValuesJSON, err := json.Marshal(*chatBlock.ItemValues); err == nil {
				if err := dr.PklresHelper.Set(actionID, "itemValues", string(itemValuesJSON)); err != nil {
					dr.Logger.Error("processLLMChat: failed to store itemValues", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("processLLMChat: failed to marshal itemValues", "actionID", actionID, "error", err)
			}
		}

		// Store TimeoutDuration
		if chatBlock.TimeoutDuration != nil {
			timeoutStr := fmt.Sprintf("%g", chatBlock.TimeoutDuration.Value)
			if err := dr.PklresHelper.Set(actionID, "timeoutDuration", timeoutStr); err != nil {
				dr.Logger.Error("processLLMChat: failed to store timeoutDuration", "actionID", actionID, "error", err)
			}
		}

		// Store Description if present (field might not exist in current schema)
		// TODO: Re-enable when Description field is available in ResourceChat struct
		// if chatBlock.Description != nil && *chatBlock.Description != "" {
		//     if err := dr.PklresHelper.Set(actionID, "description", *chatBlock.Description); err != nil {
		//         dr.Logger.Error("processLLMChat: failed to store description", "actionID", actionID, "error", err)
		//     }
		// }

		if chatBlock.Timestamp != nil {
			timestampStr := fmt.Sprintf("%g", chatBlock.Timestamp.Value)
			if err := dr.PklresHelper.Set(actionID, "timestamp", timestampStr); err != nil {
				dr.Logger.Error("processLLMChat: failed to store timestamp", "actionID", actionID, "error", err)
			}
		}

		dr.Logger.Info("processLLMChat: stored comprehensive LLM resource attributes in pklres", "actionID", actionID)
	}

	// Mark the resource as finished processing
	// Processing status tracking removed - simplified to pure key-value store approach

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
		model := ""
		if res.Model != nil {
			model = *res.Model
		}
		pklContent.WriteString(fmt.Sprintf("    Model = %q\n", model))

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

	// Use the response content directly without base64 decoding
	content := utils.SafeDerefString(responseEncoded)

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}

// EncodeChat encodes a chat block for LLM processing.
