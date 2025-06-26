package resolver

import (
	pklHTTP "github.com/kdeps/schema/gen/http"
)

// HandleHTTPClient manages the execution of an HTTP block.
func (dr *DependencyResolver) HandleHTTPClient(actionID string, httpBlock *pklHTTP.ResourceHTTPClient, hasItems bool) error {
	// Run processHTTPBlock asynchronously in a goroutine
	go func(aID string, block *pklHTTP.ResourceHTTPClient) {
		if err := dr.processHTTPBlock(aID, block, hasItems); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process http block", "actionID", aID, "error", err)
		}
	}(actionID, httpBlock)

	return nil
}

// processHTTPBlock processes an HTTP block and stores it in SQLite.
func (dr *DependencyResolver) processHTTPBlock(actionID string, httpBlock *pklHTTP.ResourceHTTPClient, hasItems bool) error {
	// Store the HTTP resource in SQLite using the SQLite resource storage method
	return dr.StoreHTTPResource(actionID, httpBlock, hasItems)
}
