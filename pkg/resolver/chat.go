package resolver

import (
	"errors"
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func (dr *DependencyResolver) HandleLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
	// Decode Prompt if it is Base64-encoded
	if utf8.ValidString(chatBlock.Prompt) && utils.IsBase64Encoded(chatBlock.Prompt) {
		decodedPrompt, err := utils.DecodeBase64String(chatBlock.Prompt)
		if err == nil {
			chatBlock.Prompt = decodedPrompt
		}
	}

	// Decode the jsonResponseKeys field if it exists
	if chatBlock.JsonResponseKeys != nil {
		decodedJsonResponseKeys := make([]string, len(*chatBlock.JsonResponseKeys))
		for i, v := range *chatBlock.JsonResponseKeys {
			// Check if the key value is Base64 encoded
			if utils.IsBase64Encoded(v) {
				decodedValue, err := utils.DecodeBase64String(v)
				if err != nil {
					return fmt.Errorf("failed to decode response key at index %d: %w", i, err)
				}
				decodedJsonResponseKeys[i] = decodedValue
			} else {
				// If not Base64 encoded, leave the value as it is
				decodedJsonResponseKeys[i] = v
			}
		}
		chatBlock.JsonResponseKeys = &decodedJsonResponseKeys
	}

	err := dr.processLLMChat(actionId, chatBlock)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) processLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
	var completion string

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return err
	}

	if chatBlock.JsonResponse != nil && *chatBlock.JsonResponse {
		// Base system prompt asking for JSON format response
		systemPrompt := "Respond in JSON format."

		// Check if there are JsonResponseKeys to include in the prompt
		if chatBlock.JsonResponseKeys != nil && len(*chatBlock.JsonResponseKeys) > 0 {
			// Join the keys and append to the prompt
			additionalKeys := strings.Join(*chatBlock.JsonResponseKeys, "`, `")
			systemPrompt = fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", additionalKeys)
		}

		var files = make(map[string][]byte)

		if chatBlock.Files != nil {
			for _, file := range *chatBlock.Files {
				// Read file content from the file path
				fileBytes, err := os.ReadFile(file)
				if err != nil {
					return err
				}

				// Detect file type (mimetype)
				filetype := mimetype.Detect(fileBytes).String()

				// Store filetype as key and file content as value
				files[filetype] = fileBytes
			}
		}

		// Initialize message content with text parts
		content := []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
			llms.TextParts(llms.ChatMessageTypeHuman, chatBlock.Prompt),
		}

		for filetype, fileBytes := range files {
			binaryContent := llms.MessageContent{
				Role: llms.ChatMessageTypeHuman,
				Parts: []llms.ContentPart{
					llms.BinaryPart(filetype, fileBytes),
				},
			}

			content = append(content, binaryContent)
		}

		response, err := llm.GenerateContent(*dr.Context, content, llms.WithJSONMode())
		if err != nil {
			return err
		}

		choices := response.Choices
		if len(choices) < 1 {
			return errors.New("Empty response from model")
		}

		completion = choices[0].Content
	} else {
		completion, err = llm.Call(*dr.Context, chatBlock.Prompt)
		if err != nil {
			return err
		}
	}

	chatBlock.Response = &completion

	if err := dr.AppendChatEntry(actionId, chatBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) AppendChatEntry(resourceId string, newChat *pklLLM.ResourceChat) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "llm/"+dr.RequestId+"__llm_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.GetResources() // Dereference the pointer to get the map

	// Check and Base64 encode model, prompt, and response if not already encoded
	encodedModel := newChat.Model
	if !utils.IsBase64Encoded(newChat.Model) {
		encodedModel = utils.EncodeBase64String(newChat.Model)
	}

	encodedPrompt := newChat.Prompt
	if !utils.IsBase64Encoded(newChat.Prompt) {
		encodedPrompt = utils.EncodeBase64String(newChat.Prompt)
	}

	var encodedResponse string
	if newChat.Response != nil {
		if !utils.IsBase64Encoded(*newChat.Response) {
			encodedResponse = utils.EncodeBase64String(*newChat.Response)
		} else {
			encodedResponse = *newChat.Response
		}
	}

	// Create or update the ResourceChat entry
	existingResources[resourceId] = &pklLLM.ResourceChat{
		Model:     encodedModel,
		Prompt:    encodedPrompt,
		Response:  &encodedResponse,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = \"%s\"\n", resource.Model))
		pklContent.WriteString(fmt.Sprintf("    prompt = \"%s\"\n", resource.Prompt))

		if resource.JsonResponse != nil {
			pklContent.WriteString(fmt.Sprintf("    jsonResponse = %t\n", *resource.JsonResponse))
		}

		if resource.JsonResponseKeys != nil {
			pklContent.WriteString("    jsonResponseKeys {\n")
			for _, value := range *resource.JsonResponseKeys {
				var encodedData string
				if utils.IsBase64Encoded(value) {
					encodedData = value // Use as it is if already Base64 encoded
				} else {
					encodedData = utils.EncodeBase64String(value) // Otherwise, encode it
				}
				pklContent.WriteString(fmt.Sprintf("      \"%s\"\n", encodedData))
			}
			pklContent.WriteString("    }\n")
		} else {
			pklContent.WriteString("    jsonResponseKeys {\"\"}\n")
		}

		pklContent.WriteString(fmt.Sprintf("    timeoutSeconds = %d\n", resource.TimeoutSeconds))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))
		// Dereference response to pass it correctly
		if resource.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = #\"\"\"\n%s\n\"\"\"#\n", *resource.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n")
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
