package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/evaluator"
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

	return nil
}

// mapRoleToLLMMessageType maps user-defined roles to llms.ChatMessageType.
func mapRoleToLLMMessageType(role string) llms.ChatMessageType {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "human":
		return llms.ChatMessageTypeHuman
	case "system":
		return llms.ChatMessageTypeSystem
	case "ai", "assistant":
		return llms.ChatMessageTypeAI
	case "function":
		return llms.ChatMessageTypeFunction
	case "tool":
		return llms.ChatMessageTypeTool
	default:
		// fallback to generic
		return llms.ChatMessageTypeGeneric
	}
}

func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	var completion string

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return err
	}

	if chatBlock.JSONResponse != nil && *chatBlock.JSONResponse {
		systemPrompt := "Respond in JSON format."
		if chatBlock.JSONResponseKeys != nil && len(*chatBlock.JSONResponseKeys) > 0 {
			systemPrompt = fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", strings.Join(*chatBlock.JSONResponseKeys, "`, `"))
		}

		role := utils.DerefString(chatBlock.Role)
		if strings.TrimSpace(role) == "" {
			role = "human"
		}
		roleType := mapRoleToLLMMessageType(role)
		prompt := utils.DerefString(chatBlock.Prompt)

		if roleType == llms.ChatMessageTypeGeneric {
			prompt = fmt.Sprintf("[%s]: %s", role, prompt)
		}

		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(roleType, prompt),
		}

		response, err := llm.GenerateContent(dr.Context, content, llms.WithJSONMode())
		if err != nil {
			return err
		}

		if len(response.Choices) == 0 {
			return errors.New("empty response from model")
		}
		completion = response.Choices[0].Content
	} else {
		completion, err = llm.Call(dr.Context, utils.DerefString(chatBlock.Prompt))
		if err != nil {
			return err
		}
	}

	chatBlock.Response = &completion
	return dr.AppendChatEntry(actionID, chatBlock)
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
