package resolver

import (
	pklLLM "github.com/kdeps/schema/gen/llm"
)

// HandleLLMChat manages the execution of an LLM chat block.
func (dr *DependencyResolver) HandleLLMChat(actionID string, chatBlock *pklLLM.ResourceChat, hasItems bool) error {
	// Run processLLMChat asynchronously in a goroutine
	go func(aID string, block *pklLLM.ResourceChat) {
		if err := dr.processLLMChat(aID, block, hasItems); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process llm chat block", "actionID", aID, "error", err)
		}
	}(actionID, chatBlock)

	return nil
}

// processLLMChat processes an LLM chat block and stores it in SQLite.
func (dr *DependencyResolver) processLLMChat(actionID string, chatBlock *pklLLM.ResourceChat, hasItems bool) error {
	// Store the LLM resource in SQLite using the SQLite resource storage method
	return dr.StoreLLMResource(actionID, chatBlock, hasItems)
}
