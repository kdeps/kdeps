package resolver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

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

func (dr *DependencyResolver) decodeChatBlock(chatBlock *pklLLM.ResourceChat) error {
	decodedPrompt, err := utils.DecodeBase64IfNeeded(utils.DerefString(chatBlock.Prompt))
	if err != nil {
		return fmt.Errorf("failed to decode Prompt: %w", err)
	}
	chatBlock.Prompt = &decodedPrompt

	decodedRole, err := utils.DecodeBase64IfNeeded(utils.DerefString(chatBlock.Role))
	if err != nil {
		return fmt.Errorf("failed to decode Role: %w", err)
	}
	chatBlock.Role = &decodedRole

	if chatBlock.JSONResponseKeys != nil {
		decodedKeys, err := utils.DecodeStringSlice(chatBlock.JSONResponseKeys, "JSONResponseKeys")
		if err != nil {
			return fmt.Errorf("failed to decode JSONResponseKeys: %w", err)
		}
		chatBlock.JSONResponseKeys = decodedKeys
	}

	if chatBlock.Scenario != nil {
		dr.Logger.Info("Decoding Scenario", "length", len(*chatBlock.Scenario))
		decodedScenario := make([]*pklLLM.MultiChat, len(*chatBlock.Scenario))
		for i, entry := range *chatBlock.Scenario {
			if entry == nil {
				dr.Logger.Info("Scenario entry is nil", "index", i)
				decodedScenario[i] = nil
				continue
			}
			decodedEntryRole, err := utils.DecodeBase64IfNeeded(utils.DerefString(entry.Role))
			if err != nil {
				return fmt.Errorf("failed to decode Scenario[%d].Role: %w", i, err)
			}
			decodedEntryPrompt, err := utils.DecodeBase64IfNeeded(utils.DerefString(entry.Prompt))
			if err != nil {
				return fmt.Errorf("failed to decode Scenario[%d].Prompt: %w", i, err)
			}
			dr.Logger.Info("Decoded Scenario entry", "index", i, "role", decodedEntryRole, "prompt", decodedEntryPrompt)
			decodedScenario[i] = &pklLLM.MultiChat{
				Role:   &decodedEntryRole,
				Prompt: &decodedEntryPrompt,
			}
		}
		chatBlock.Scenario = &decodedScenario
	} else {
		dr.Logger.Info("Scenario is nil")
	}

	return nil
}

// mapRoleToLLMMessageType maps user-defined roles to llms.ChatMessageType.
func mapRoleToLLMMessageType(role string) llms.ChatMessageType {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "human", "user", "person", "client":
		return llms.ChatMessageTypeHuman
	case "system":
		return llms.ChatMessageTypeSystem
	case "ai", "assistant", "bot", "chatbot", "llm":
		return llms.ChatMessageTypeAI
	case "function", "action":
		return llms.ChatMessageTypeFunction
	case "tool":
		return llms.ChatMessageTypeTool
	default:
		// fallback to generic
		return llms.ChatMessageTypeGeneric
	}
}

func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	if chatBlock == nil {
		return errors.New("chatBlock cannot be nil")
	}

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	completion, err := generateChatResponse(dr.Context, llm, chatBlock, dr.Logger)
	if err != nil {
		return err
	}

	chatBlock.Response = &completion
	return dr.AppendChatEntry(actionID, chatBlock)
}

func generateChatResponse(ctx context.Context, llm *ollama.LLM, chatBlock *pklLLM.ResourceChat, logger *logging.Logger) (string, error) {
	if chatBlock.JSONResponse == nil || !*chatBlock.JSONResponse {
		prompt := utils.DerefString(chatBlock.Prompt)
		if strings.TrimSpace(prompt) == "" {
			return "", errors.New("prompt cannot be empty for non-JSON response")
		}
		logger.Info("Sending non-JSON prompt to LLM", "prompt", prompt)
		return llm.Call(ctx, prompt)
	}

	// Estimate capacity for content slice
	capacity := 1 // System prompt
	if strings.TrimSpace(utils.DerefString(chatBlock.Prompt)) != "" {
		capacity++ // Main prompt
	}
	if chatBlock.Scenario != nil {
		capacity += len(*chatBlock.Scenario) // Safe: checked scenario != nil
	}

	// Pre-allocate content slice with estimated capacity
	content := make([]llms.MessageContent, 0, capacity)

	systemPrompt := buildSystemPrompt(chatBlock.JSONResponseKeys)
	role, roleType := getRoleAndType(chatBlock.Role)
	prompt := utils.DerefString(chatBlock.Prompt)

	// Add system prompt
	content = append(content, llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextContent{Text: systemPrompt}},
	})

	if strings.TrimSpace(prompt) != "" {
		if roleType == llms.ChatMessageTypeGeneric {
			prompt = fmt.Sprintf("[%s]: %s", role, prompt)
		}
		content = append(content, llms.MessageContent{
			Role:  roleType,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		})
	}

	content = append(content, processScenarioMessages(chatBlock.Scenario, logger)...)

	// Log the content being sent to ollama
	for i, msg := range content {
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				logger.Info("Sending message to LLM", "index", i, "role", msg.Role, "content", textPart.Text)
			}
		}
	}

	response, err := llm.GenerateContent(ctx, content, llms.WithJSONMode())
	if err != nil {
		logger.Error("Failed to generate JSON content", "error", err)
		return "", fmt.Errorf("failed to generate JSON content: %w", err)
	}
	if len(response.Choices) == 0 {
		logger.Error("Empty response from LLM")
		return "", errors.New("empty response from model")
	}
	logger.Info("Received LLM response", "content", response.Choices[0].Content)
	return response.Choices[0].Content, nil
}

func buildSystemPrompt(jsonResponseKeys *[]string) string {
	if jsonResponseKeys != nil && len(*jsonResponseKeys) > 0 {
		return fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", strings.Join(*jsonResponseKeys, "`, `"))
	}
	return "Respond in JSON format."
}

func getRoleAndType(rolePtr *string) (string, llms.ChatMessageType) {
	role := utils.DerefString(rolePtr)
	if strings.TrimSpace(role) == "" {
		role = "human"
	}
	return role, mapRoleToLLMMessageType(role)
}

func processScenarioMessages(scenario *[]*pklLLM.MultiChat, logger *logging.Logger) []llms.MessageContent {
	// Return empty slice if scenario is nil
	if scenario == nil {
		logger.Info("No scenario messages to process")
		return make([]llms.MessageContent, 0)
	}

	logger.Info("Processing scenario messages", "count", len(*scenario))
	// Pre-allocate content with max possible size
	content := make([]llms.MessageContent, 0, len(*scenario))

	for i, entry := range *scenario {
		if entry == nil {
			logger.Info("Skipping nil scenario entry", "index", i)
			continue
		}
		prompt := utils.DerefString(entry.Prompt)
		if strings.TrimSpace(prompt) == "" {
			logger.Info("Skipping empty scenario prompt", "index", i, "role", utils.DerefString(entry.Role))
			continue
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

func (dr *DependencyResolver) AppendChatEntry(resourceID string, newChat *pklLLM.ResourceChat) error {
	pklPath := filepath.Join(dr.ActionDir, "llm/"+dr.RequestID+"__llm_output.pkl")

	pklRes, err := pklLLM.LoadFromPath(dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
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

	// Encode scenario entries
	var encodedScenario *[]*pklLLM.MultiChat
	if newChat.Scenario != nil {
		encodedEntries := make([]*pklLLM.MultiChat, len(*newChat.Scenario))
		for i, entry := range *newChat.Scenario {
			if entry == nil {
				continue
			}
			encodedRole := utils.EncodeValue(utils.DerefString(entry.Role))
			encodedPrompt := utils.EncodeValue(utils.DerefString(entry.Prompt))
			encodedEntries[i] = &pklLLM.MultiChat{
				Role:   &encodedRole,
				Prompt: &encodedPrompt,
			}
		}
		encodedScenario = &encodedEntries
	}

	encodedModel := utils.EncodeValue(newChat.Model)
	encodedRole := utils.EncodeValue(utils.DerefString(newChat.Role))
	encodedPrompt := utils.EncodeValue(utils.DerefString(newChat.Prompt))
	encodedResponse := utils.EncodeValuePtr(newChat.Response)
	encodedJSONResponseKeys := dr.encodeChatJSONResponseKeys(newChat.JSONResponseKeys)

	timeoutDuration := newChat.TimeoutDuration
	if timeoutDuration == nil {
		timeoutDuration = &pkl.Duration{
			Value: 60,
			Unit:  pkl.Second,
		}
	}

	timestamp := newChat.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{
			Value: float64(time.Now().Unix()),
			Unit:  pkl.Nanosecond,
		}
	}

	existingResources[resourceID] = &pklLLM.ResourceChat{
		Model:            encodedModel,
		Prompt:           &encodedPrompt,
		Role:             &encodedRole,
		Scenario:         encodedScenario,
		JSONResponse:     newChat.JSONResponse,
		JSONResponseKeys: encodedJSONResponseKeys,
		Response:         encodedResponse,
		File:             &filePath,
		Timestamp:        timestamp,
		TimeoutDuration:  timeoutDuration,
	}

	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, res := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = \"%s\"\n", res.Model))

		if res.Prompt != nil {
			pklContent.WriteString(fmt.Sprintf("    prompt = \"%s\"\n", *res.Prompt))
		} else {
			pklContent.WriteString("    prompt = \"\"\n")
		}

		if res.Role != nil {
			pklContent.WriteString(fmt.Sprintf("    role = \"%s\"\n", *res.Role))
		} else {
			pklContent.WriteString("    role = \"\"\n")
		}

		// Add scenario
		pklContent.WriteString("    scenario ")
		if res.Scenario != nil && len(*res.Scenario) > 0 {
			pklContent.WriteString("{\n")
			for _, entry := range *res.Scenario {
				if entry == nil {
					continue
				}
				pklContent.WriteString("      new {\n")
				if entry.Role != nil {
					pklContent.WriteString(fmt.Sprintf("        role = \"%s\"\n", *entry.Role))
				} else {
					pklContent.WriteString("        role = \"\"\n")
				}
				if entry.Prompt != nil {
					pklContent.WriteString(fmt.Sprintf("        prompt = \"%s\"\n", *entry.Prompt))
				} else {
					pklContent.WriteString("        prompt = \"\"\n")
				}
				pklContent.WriteString("      }\n")
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("{}\n")
		}

		if res.JSONResponse != nil {
			pklContent.WriteString(fmt.Sprintf("    JSONResponse = %t\n", *res.JSONResponse))
		}

		pklContent.WriteString("    JSONResponseKeys ")
		if res.JSONResponseKeys != nil {
			pklContent.WriteString(utils.EncodePklSlice(res.JSONResponseKeys))
		} else {
			pklContent.WriteString("{}\n")
		}

		if res.TimeoutDuration != nil {
			pklContent.WriteString(fmt.Sprintf("    timeoutDuration = %g.%s\n", res.TimeoutDuration.Value, res.TimeoutDuration.Unit.String()))
		} else {
			pklContent.WriteString("    timeoutDuration = 60.s\n")
		}

		if res.Timestamp != nil {
			pklContent.WriteString(fmt.Sprintf("    timestamp = %g.%s\n", res.Timestamp.Value, res.Timestamp.Unit.String()))
		}

		if res.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = #\"\"\"\n%s\n\"\"\"#\n", *res.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n")
		}

		pklContent.WriteString(fmt.Sprintf("    file = \"%s\"\n", *res.File))
		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL file: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

func (dr *DependencyResolver) encodeChatJSONResponseKeys(keys *[]string) *[]string {
	if keys == nil {
		return nil
	}
	encoded := make([]string, len(*keys))
	for i, v := range *keys {
		encoded[i] = utils.EncodeValue(v)
	}
	return &encoded
}

func (dr *DependencyResolver) WriteResponseToFile(resourceID string, responseEncoded *string) (string, error) {
	if responseEncoded == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	content, err := utils.DecodeBase64IfNeeded(utils.DerefString(responseEncoded))
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}
