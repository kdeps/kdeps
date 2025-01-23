package resolver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func (dr *DependencyResolver) HandleLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	// Decode Prompt if it is Base64-encoded
	if utf8.ValidString(chatBlock.Prompt) && utils.IsBase64Encoded(chatBlock.Prompt) {
		decodedPrompt, err := utils.DecodeBase64String(chatBlock.Prompt)
		if err == nil {
			chatBlock.Prompt = decodedPrompt
		}
	}

	// Decode the JSONResponseKeys field if it exists
	if chatBlock.JSONResponseKeys != nil {
		decodedJSONResponseKeys := make([]string, len(*chatBlock.JSONResponseKeys))
		for i, v := range *chatBlock.JSONResponseKeys {
			// Check if the key value is Base64 encoded
			if utils.IsBase64Encoded(v) {
				decodedValue, err := utils.DecodeBase64String(v)
				if err != nil {
					return fmt.Errorf("failed to decode response key at index %d: %w", i, err)
				}
				decodedJSONResponseKeys[i] = decodedValue
			} else {
				// If not Base64 encoded, leave the value as it is
				decodedJSONResponseKeys[i] = v
			}
		}
		chatBlock.JSONResponseKeys = &decodedJSONResponseKeys
	}

	err := dr.processLLMChat(actionID, chatBlock)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat) error {
	var completion string

	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return err
	}

	if chatBlock.JSONResponse != nil && *chatBlock.JSONResponse {
		// Base system prompt asking for JSON format response
		systemPrompt := "Respond in JSON format."

		// Check if there are JSONResponseKeys to include in the prompt
		if chatBlock.JSONResponseKeys != nil && len(*chatBlock.JSONResponseKeys) > 0 {
			// Join the keys and append to the prompt
			additionalKeys := strings.Join(*chatBlock.JSONResponseKeys, "`, `")
			systemPrompt = fmt.Sprintf("Respond in JSON format, include `%s` in response keys.", additionalKeys)
		}

		files := make(map[string][]byte)

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

		response, err := llm.GenerateContent(dr.Context, content, llms.WithJSONMode())
		if err != nil {
			return err
		}

		choices := response.Choices
		if len(choices) < 1 {
			return errors.New("Empty response from model")
		}

		completion = choices[0].Content
	} else {
		completion, err = llm.Call(dr.Context, chatBlock.Prompt)
		if err != nil {
			return err
		}
	}

	chatBlock.Response = &completion

	if err := dr.AppendChatEntry(actionID, chatBlock); err != nil {
		return err
	}

	return nil
}

func (dr *DependencyResolver) WriteResponseToFile(resourceID string, responseEncoded *string) (string, error) {
	// Convert resourceID to be filename friendly
	resourceIDFile := utils.ConvertToFilenameFriendly(resourceID)
	// Define the file path using the FilesDir and resource ID
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Ensure the Response is not nil
	if responseEncoded != nil {
		// Prepare the content to write
		var content string
		if utils.IsBase64Encoded(*responseEncoded) {
			// Decode the Base64-encoded Response string
			decodedResponse, err := utils.DecodeBase64String(*responseEncoded)
			if err != nil {
				return "", fmt.Errorf("failed to decode Base64 string for resource ID: %s: %w", resourceID, err)
			}
			content = decodedResponse
		} else {
			// Use the Response content as-is if not Base64-encoded
			content = *responseEncoded
		}

		// Write the content to the file
		err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644)
		if err != nil {
			return "", fmt.Errorf("failed to write Response to file for resource ID: %s: %w", resourceID, err)
		}
	} else {
		return "", nil
	}

	return outputFilePath, nil
}

func (dr *DependencyResolver) AppendChatEntry(resourceID string, newChat *pklLLM.ResourceChat) error {
	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "llm/"+dr.RequestID+"__llm_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklLLM.LoadFromPath(dr.Context, pklPath)
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

	var filePath, encodedResponse string
	if newChat.Response != nil {
		filePath, err = dr.WriteResponseToFile(resourceID, newChat.Response)
		if err != nil {
			return fmt.Errorf("failed to write Response to file: %w", err)
		}
		newChat.File = &filePath

		if !utils.IsBase64Encoded(*newChat.Response) {
			encodedResponse = utils.EncodeBase64String(*newChat.Response)
		} else {
			encodedResponse = *newChat.Response
		}
	}

	// Create or update the ResourceChat entry
	existingResources[resourceID] = &pklLLM.ResourceChat{
		Model:     encodedModel,
		Prompt:    encodedPrompt,
		Response:  &encodedResponse,
		File:      &filePath,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("resources {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = \"%s\"\n", resource.Model))
		pklContent.WriteString(fmt.Sprintf("    prompt = \"%s\"\n", resource.Prompt))

		if resource.JSONResponse != nil {
			pklContent.WriteString(fmt.Sprintf("    JSONResponse = %t\n", *resource.JSONResponse))
		}

		if resource.JSONResponseKeys != nil {
			pklContent.WriteString("    JSONResponseKeys {\n")
			for _, value := range *resource.JSONResponseKeys {
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
			pklContent.WriteString("    JSONResponseKeys {\"\"}\n")
		}

		pklContent.WriteString(fmt.Sprintf("    timeoutSeconds = %d\n", resource.TimeoutSeconds))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))

		// Dereference response to pass it correctly
		if resource.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = #\"\"\"\n%s\n\"\"\"#\n", *resource.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n")
		}

		pklContent.WriteString(fmt.Sprintf("    file = \"%s\"\n", filePath))

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	// Evaluate the PKL file using EvalPkl
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
