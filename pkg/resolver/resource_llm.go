package resolver

import (
	pklLLM "github.com/kdeps/schema/gen/llm"
)

func (dr *DependencyResolver) HandleLLM(actionID string, llmBlock *pklLLM.ResourceChat) error {
	// Synchronously decode the LLM block.
	if err := dr.DecodeLLMBlockFunc(llmBlock); err != nil {
		dr.Logger.Error("failed to decode LLM block", "actionID", actionID, "error", err)
		return err
	}

	// Process the LLM block asynchronously in a goroutine.
	go func(aID string, block *pklLLM.ResourceChat) {
		if err := dr.ProcessLLMBlockFunc(aID, block); err != nil {
			// Log the error; you can adjust error handling as needed.
			dr.Logger.Error("failed to process LLM block", "actionID", aID, "error", err)
		}
	}(actionID, llmBlock)

	// Return immediately; the LLM block is processed in the background.
	return nil
}
