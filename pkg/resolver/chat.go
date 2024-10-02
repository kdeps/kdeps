package resolver

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pklLLM "github.com/kdeps/schema/gen/llm"
	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/llms/ollama"
)

func (dr *DependencyResolver) HandleLLMChat(actionId string, chatBlock *pklLLM.ResourceChat) error {
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
	pklPath := filepath.Join(dr.ActionDir, "llm/llm_output.pkl")

	// Get the current timestamp
	newTimestamp := uint32(time.Now().UnixNano())

	// Load existing PKL data
	pklRes, err := pklLLM.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Ensure pklRes.Resource is of type *map[string]*llm.ResourceChat
	existingResources := *pklRes.Resource // Dereference the pointer to get the map

	// Create or update the ResourceChat entry
	existingResources[resourceId] = &pklLLM.ResourceChat{
		Model:     newChat.Model,
		Prompt:    newChat.Prompt,
		Response:  newChat.Response,
		Timestamp: &newTimestamp,
	}

	// Build the new content for the PKL file in the specified format
	var pklContent strings.Builder
	pklContent.WriteString("amends \"package://schema.kdeps.com/core@0.0.50#/LLM.pkl\"\n\n")
	pklContent.WriteString("resource {\n")

	for id, resource := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    model = \"%s\"\n", resource.Model))
		pklContent.WriteString(fmt.Sprintf("    prompt = \"\"\"\n%s\n\"\"\"\n", resource.Prompt))
		pklContent.WriteString(fmt.Sprintf("    timestamp = %d\n", *resource.Timestamp))
		// Dereference response to pass it correctly
		if resource.Response != nil {
			pklContent.WriteString(fmt.Sprintf("    response = \"\"\"\n%s\n\"\"\"\n", *resource.Response))
		} else {
			pklContent.WriteString("    response = \"\"\n") // Handle nil case
		}

		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Write the new PKL content to the file using afero
	err = afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write to PKL file: %w", err)
	}

	return nil
}
