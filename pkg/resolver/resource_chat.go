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
	RoleHuman           = "human"
	RoleUser            = "user"
	RolePerson          = "person"
	RoleClient          = "client"
	RoleSystem          = "system"
	RoleAI              = "ai"
	RoleAssistant       = "assistant"
	RoleBot             = "bot"
	RoleChatbot         = "chatbot"
	RoleLLM             = "llm"
	RoleFunction        = "function"
	RoleAction          = "action"
	RoleTool            = "tool"
	maxLogContentLength = 100
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

func logChatBlockInfo(chatBlock *pklLLM.ResourceChat, logger *logging.Logger) {
	logger.Info("Processing chatBlock",
		"model", chatBlock.Model,
		"prompt", utils.SafeDerefString(chatBlock.Prompt),
		"role", utils.SafeDerefString(chatBlock.Role),
		"json_response", utils.SafeDerefBool(chatBlock.JSONResponse),
		"json_response_keys", utils.SafeDerefSlice(chatBlock.JSONResponseKeys),
		"tool_count", len(utils.SafeDerefSlice(chatBlock.Tools)),
		"scenario_count", len(utils.SafeDerefSlice(chatBlock.Scenario)),
		"file_count", len(utils.SafeDerefSlice(chatBlock.Files)))
}

func logGeneratedTools(availableTools []llms.Tool, logger *logging.Logger) {
	logger.Info("Generated tools",
		"tool_count", len(availableTools),
		"tool_names", extractToolNamesFromTools(availableTools))
}

func buildMessageHistory(ctx context.Context, fs afero.Fs, chatBlock *pklLLM.ResourceChat, availableTools []llms.Tool, logger *logging.Logger) ([]llms.MessageContent, llms.ChatMessageType, error) {
	messageHistory := make([]llms.MessageContent, 0)
	role, roleType := getRoleAndType(chatBlock.Role)

	if err := addSystemPrompt(&messageHistory, chatBlock, availableTools, logger); err != nil {
		return nil, "", err
	}

	if err := addMainPrompt(&messageHistory, chatBlock, role, roleType); err != nil {
		return nil, "", err
	}

	if err := addScenarioMessages(&messageHistory, chatBlock, logger); err != nil {
		return nil, "", err
	}

	if err := addFileContents(&messageHistory, fs, chatBlock, roleType, logger); err != nil {
		return nil, "", err
	}

	return messageHistory, roleType, nil
}

func addSystemPrompt(messageHistory *[]llms.MessageContent, chatBlock *pklLLM.ResourceChat, availableTools []llms.Tool, logger *logging.Logger) error {
	systemPrompt := buildSystemPrompt(chatBlock.JSONResponse, chatBlock.JSONResponseKeys, availableTools)
	logger.Info("Generated system prompt", "content", utils.TruncateString(systemPrompt, 200))

	*messageHistory = append(*messageHistory, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})
	return nil
}

func addMainPrompt(messageHistory *[]llms.MessageContent, chatBlock *pklLLM.ResourceChat, role string, roleType llms.ChatMessageType) error {
	prompt := utils.SafeDerefString(chatBlock.Prompt)
	if strings.TrimSpace(prompt) == "" {
		return nil
	}

	if roleType == llms.ChatMessageTypeGeneric {
		prompt = "[" + role + "]: " + prompt
	}

	*messageHistory = append(*messageHistory, llms.MessageContent{
		Role:  roleType,
		Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
	})
	return nil
}

func addScenarioMessages(messageHistory *[]llms.MessageContent, chatBlock *pklLLM.ResourceChat, logger *logging.Logger) error {
	*messageHistory = append(*messageHistory, processScenarioMessages(chatBlock.Scenario, logger)...)
	return nil
}

func addFileContents(messageHistory *[]llms.MessageContent, fs afero.Fs, chatBlock *pklLLM.ResourceChat, roleType llms.ChatMessageType, logger *logging.Logger) error {
	if chatBlock.Files == nil || len(*chatBlock.Files) == 0 {
		return nil
	}

	for i, filePath := range *chatBlock.Files {
		if err := addSingleFile(messageHistory, fs, filePath, i, roleType, logger); err != nil {
			return err
		}
	}
	return nil
}

func addSingleFile(messageHistory *[]llms.MessageContent, fs afero.Fs, filePath string, index int, roleType llms.ChatMessageType, logger *logging.Logger) error {
	fileBytes, err := afero.ReadFile(fs, filePath)
	if err != nil {
		logger.Error("Failed to read file", "index", index, "path", filePath, "error", err)
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	fileType := mimetype.Detect(fileBytes).String()
	logger.Info("Detected MIME type for file", "index", index, "path", filePath, "mimeType", fileType)

	*messageHistory = append(*messageHistory, llms.MessageContent{
		Role: roleType,
		Parts: []llms.ContentPart{
			llms.BinaryPart(fileType, fileBytes),
		},
	})
	return nil
}

// generateChatResponse generates a response from the LLM based on the chat block, executing tools via toolreader.
func generateChatResponse(ctx context.Context, fs afero.Fs, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, toolreader *tool.PklResourceReader, logger *logging.Logger) (string, error) {
	logChatBlockInfo(chatBlock, logger)

	availableTools := generateAvailableTools(chatBlock, logger)
	logGeneratedTools(availableTools, logger)

	messageHistory, _, err := buildMessageHistory(ctx, fs, chatBlock, availableTools, logger)
	if err != nil {
		return "", err
	}

	opts := prepareCallOptions(chatBlock, availableTools, logger)

	return generateChatResponseWithTools(ctx, llm, chatBlock, toolreader, messageHistory, availableTools, opts, logger)
}

func prepareCallOptions(chatBlock *pklLLM.ResourceChat, availableTools []llms.Tool, logger *logging.Logger) []llms.CallOption {
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

	return opts
}

func generateChatResponseWithTools(ctx context.Context, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, toolreader *tool.PklResourceReader, messageHistory []llms.MessageContent, availableTools []llms.Tool, opts []llms.CallOption, logger *logging.Logger) (string, error) {
	toolOutputs := make(map[string]string)

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

	return handleLLMResponseWithTools(ctx, llm, toolreader, response, &messageHistory, availableTools, opts, toolOutputs, logger)
}

func handleLLMResponseWithTools(ctx context.Context, llm *ollama.LLM, toolreader *tool.PklResourceReader, response *llms.ContentResponse, messageHistory *[]llms.MessageContent, availableTools []llms.Tool, opts []llms.CallOption, toolOutputs map[string]string, logger *logging.Logger) (string, error) {
	respChoice := selectBestChoice(response, availableTools)

	return processLLMResponseLoop(ctx, llm, toolreader, respChoice, messageHistory, availableTools, opts, toolOutputs, logger)
}

func selectBestChoice(response *llms.ContentResponse, availableTools []llms.Tool) *llms.ContentChoice {
	if len(availableTools) > 0 {
		for _, choice := range response.Choices {
			if len(choice.ToolCalls) > 0 {
				return choice
			}
		}
	}
	return response.Choices[0]
}

func processLLMResponseLoop(ctx context.Context, llm *ollama.LLM, toolreader *tool.PklResourceReader, respChoice *llms.ContentChoice, messageHistory *[]llms.MessageContent, availableTools []llms.Tool, opts []llms.CallOption, toolOutputs map[string]string, logger *logging.Logger) (string, error) {
	const maxIterations = 5
	const maxLogContentLength = 500

	for iteration := 0; iteration < maxIterations; iteration++ {
		logger.Info("Processing iteration", "iteration", iteration, "max_iterations", maxIterations)

		toolCalls := processResponseToolCalls(respChoice, availableTools, logger)

		if err := addResponseToHistory(respChoice, toolCalls, messageHistory, logger); err != nil {
			return "", err
		}

		if len(toolCalls) == 0 {
			logger.Info("No tool calls in response, returning content")
			return respChoice.Content, nil
		}

		if err := executeToolCalls(ctx, toolCalls, toolreader, toolOutputs, messageHistory, logger); err != nil {
			return "", err
		}

		if iteration == maxIterations-1 {
			logger.Warn("Reached maximum iterations", "max_iterations", maxIterations)
			return getFinalResponse(toolOutputs, respChoice, maxLogContentLength, logger)
		}

		// Generate next response with tool results
		nextResponse, err := llm.GenerateContent(ctx, *messageHistory, opts...)
		if err != nil {
			logger.Error("Failed to generate content in iteration", "iteration", iteration, "error", err)
			return "", fmt.Errorf("failed to generate content in iteration %d: %w", iteration, err)
		}

		if len(nextResponse.Choices) == 0 {
			logger.Error("No choices in LLM response for iteration", "iteration", iteration)
			return "", fmt.Errorf("no choices in LLM response for iteration %d", iteration)
		}

		respChoice = nextResponse.Choices[0]
		logger.Info("Next LLM response", "iteration", iteration, "content", utils.TruncateString(respChoice.Content, maxLogContentLength))
	}

	logger.Info("Received final LLM response", "content", utils.TruncateString(respChoice.Content, maxLogContentLength))
	return respChoice.Content, nil
}

func processResponseToolCalls(respChoice *llms.ContentChoice, availableTools []llms.Tool, logger *logging.Logger) []llms.ToolCall {
	toolCalls := respChoice.ToolCalls
	if len(toolCalls) == 0 && len(availableTools) > 0 {
		logger.Info("No direct ToolCalls, attempting to construct from JSON")
		constructedToolCalls := constructToolCallsFromJSON(respChoice.Content, logger)
		toolCalls = constructedToolCalls
	}

	// Deduplicate tool calls
	toolCalls = deduplicateToolCalls(toolCalls, logger)
	return toolCalls
}

func addResponseToHistory(respChoice *llms.ContentChoice, toolCalls []llms.ToolCall, messageHistory *[]llms.MessageContent, logger *logging.Logger) error {
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
		*messageHistory = append(*messageHistory, llms.MessageContent{
			Role:  llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{llms.TextContent{Text: assistantContent}},
		})
	}

	return nil
}

func executeToolCalls(ctx context.Context, toolCalls []llms.ToolCall, toolreader *tool.PklResourceReader, toolOutputs map[string]string, messageHistory *[]llms.MessageContent, logger *logging.Logger) error {
	// Process all tool calls
	err := processToolCalls(toolCalls, toolreader, nil, logger, messageHistory, "", toolOutputs)
	if err != nil {
		logger.Error("Tool calls processing failed", "error", err)
		return err
	}

	// Update toolOutputs with results from the processing
	for _, toolCall := range toolCalls {
		// The processToolCalls function should populate toolOutputs
		if output, exists := toolOutputs[toolCall.ID]; exists {
			// Add tool result to message history
			*messageHistory = append(*messageHistory, llms.MessageContent{
				Role:  llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{llms.TextContent{Text: output}},
			})
		}
	}
	return nil
}

func getFinalResponse(toolOutputs map[string]string, respChoice *llms.ContentChoice, maxLogContentLength int, logger *logging.Logger) (string, error) {
	// Return last tool output if available
	for _, output := range toolOutputs {
		logger.Info("Final response from max iterations", "content", utils.TruncateString(output, maxLogContentLength))
		return output, nil
	}
	return respChoice.Content, fmt.Errorf("reached maximum tool call iterations")
}

// processLLMChat processes the LLM chat and saves the response.
func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	if chatBlock == nil {
		return errors.New("chatBlock cannot be nil")
	}

	llm, err := dr.NewLLMFn(chatBlock.Model)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	completion, err := dr.GenerateChatResponseFn(dr.Context, dr.Fs, llm, chatBlock, dr.ToolReader, dr.Logger)
	if err != nil {
		return err
	}

	chatBlock.Response = &completion
	return dr.AppendChatEntry(actionID, chatBlock)
}

// AppendChatEntry appends a chat entry to the Pkl file.
func (dr *DependencyResolver) AppendChatEntry(resourceID string, newChat *pklLLM.ResourceChat) error {
	pklPath := filepath.Join(dr.ActionDir, "llm/"+dr.RequestID+"__llm_output.pkl")

	llmRes, err := dr.LoadResourceFn(dr.Context, pklPath, LLMResource)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	var pklRes pklLLM.LLMImpl
	if ptr, ok := llmRes.(*pklLLM.LLMImpl); ok {
		pklRes = *ptr
	} else if impl, ok := llmRes.(pklLLM.LLMImpl); ok {
		pklRes = impl
	} else {
		return errors.New("failed to cast pklRes to pklLLM.LLMImpl")
	}

	resources := pklRes.GetResources()
	if resources == nil {
		emptyMap := make(map[string]pklLLM.ResourceChat)
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

	encodedChat := encodeChat(newChat, dr.Logger)
	existingResources[resourceID] = *encodedChat

	pklContent := generatePklContent(existingResources, dr.Context, dr.Logger)

	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL file: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(
		dr.Fs,
		dr.Context,
		pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"", schema.SchemaVersion(dr.Context)),
		nil,
		dr.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

// generatePklContent generates Pkl content from resources.
func generatePklContent(resources map[string]pklLLM.ResourceChat, ctx context.Context, logger *logging.Logger) string {
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion(ctx)))
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
				// MultiChat is a struct, not a pointer, so we can always access it
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
