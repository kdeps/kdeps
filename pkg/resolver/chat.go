package resolver

import (
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
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

	go func() error {
		err := dr.processLLMChat(actionId, chatBlock)
		if err != nil {
			return err
		}

		return nil
	}()

	return nil
}

func (dr *DependencyResolver) processLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
	llm, err := ollama.New(ollama.WithModel(chatBlock.Model))
	if err != nil {
		return err
	}
	// Prompt - Base64 decode here
	completion, err := llm.Call(*dr.Context, chatBlock.Prompt)
	if err != nil {
		return err
	}

	llmResponse := pklLLM.ResourceChat{
		Model:    chatBlock.Model,
		Prompt:   chatBlock.Prompt,
		Response: &completion,
	}

	if err := dr.AppendChatEntry(actionId, &llmResponse); err != nil {
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
		if resource.Schema != nil {
			pklContent.WriteString(fmt.Sprintf("    schema = \"%s\"\n", *resource.Schema))
		} else {
			pklContent.WriteString("    schema = \"\"\n")
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
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, pklPath, dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the "extends" header and evaluated content
	var finalContent strings.Builder
	finalContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/LLM.pkl\"\n\n", schema.SchemaVersion))
	finalContent.WriteString(evaluatedContent)

	// Write the final evaluated content back to the PKL file
	err = afero.WriteFile(dr.Fs, pklPath, []byte(finalContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
