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
	decodedPrompt, err := utils.DecodeBase64IfNeeded(chatBlock.Prompt)
	if err != nil {
		return fmt.Errorf("failed to decode Prompt: %w", err)
	}
	chatBlock.Prompt = decodedPrompt

	if chatBlock.JSONResponseKeys != nil {
		decodedKeys, err := utils.DecodeStringSlice(chatBlock.JSONResponseKeys, "JSONResponseKeys")
		if err != nil {
			return fmt.Errorf("failed to decode JSONResponseKeys: %w", err)
		}
		chatBlock.JSONResponseKeys = decodedKeys
	}

	return nil
}

func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	var completion string

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		// Signal failure via bus service
		if dr.BusManager != nil {
			busErr := dr.BusManager.SignalResourceCompletion(actionID, "llm", "failed", map[string]interface{}{
				"error": err.Error(),
				"model": chatBlock.Model,
				"stage": "llm_initialization",
			})
			if busErr != nil {
				dr.Logger.Warn("Failed to signal LLM initialization failure via bus", "actionID", actionID, "error", busErr)
			}
		}
		return err
	}

	if chatBlock.JSONResponse != nil && *chatBlock.JSONResponse {
		systemPrompt := "Respond in JSON format."
		if chatBlock.JSONResponseKeys != nil && len(*chatBlock.JSONResponseKeys) > 0 {
			systemPrompt = fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", strings.Join(*chatBlock.JSONResponseKeys, "`, `"))
		}

		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, chatBlock.Prompt),
		}

		response, err := llm.GenerateContent(dr.Context, content, llms.WithJSONMode())
		if err != nil {
			// Signal failure via bus service
			if dr.BusManager != nil {
				busErr := dr.BusManager.SignalResourceCompletion(actionID, "llm", "failed", map[string]interface{}{
					"error":    err.Error(),
					"model":    chatBlock.Model,
					"prompt":   chatBlock.Prompt,
					"jsonMode": true,
				})
				if busErr != nil {
					dr.Logger.Warn("Failed to signal LLM JSON generation failure via bus", "actionID", actionID, "error", busErr)
				}
			}
			return err
		}

		if len(response.Choices) == 0 {
			err := errors.New("empty response from model")
			// Signal failure via bus service
			if dr.BusManager != nil {
				busErr := dr.BusManager.SignalResourceCompletion(actionID, "llm", "failed", map[string]interface{}{
					"error":    err.Error(),
					"model":    chatBlock.Model,
					"jsonMode": true,
				})
				if busErr != nil {
					dr.Logger.Warn("Failed to signal LLM empty response failure via bus", "actionID", actionID, "error", busErr)
				}
			}
			return err
		}
		completion = response.Choices[0].Content
	} else {
		completion, err = llm.Call(dr.Context, chatBlock.Prompt)
		if err != nil {
			// Signal failure via bus service
			if dr.BusManager != nil {
				busErr := dr.BusManager.SignalResourceCompletion(actionID, "llm", "failed", map[string]interface{}{
					"error":  err.Error(),
					"model":  chatBlock.Model,
					"prompt": chatBlock.Prompt,
				})
				if busErr != nil {
					dr.Logger.Warn("Failed to signal LLM call failure via bus", "actionID", actionID, "error", busErr)
				}
			}
			return err
		}
	}

	chatBlock.Response = &completion
	appendErr := dr.AppendChatEntry(actionID, chatBlock)

	// Signal completion via bus service
	if dr.BusManager != nil {
		status := "completed"
		data := map[string]interface{}{
			"model":  chatBlock.Model,
			"prompt": chatBlock.Prompt,
		}
		if appendErr != nil {
			status = "failed"
			data["error"] = appendErr.Error()
		} else {
			data["response"] = completion
		}

		busErr := dr.BusManager.SignalResourceCompletion(actionID, "llm", status, data)
		if busErr != nil {
			dr.Logger.Warn("Failed to signal LLM completion via bus", "actionID", actionID, "error", busErr)
		}
	}

	return appendErr
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
	encodedPrompt := utils.EncodeValue(newChat.Prompt)
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
		Prompt:           encodedPrompt,
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
		pklContent.WriteString(fmt.Sprintf("    prompt = \"%s\"\n", res.Prompt))

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

	content, err := utils.DecodeBase64IfNeeded(*responseEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}
