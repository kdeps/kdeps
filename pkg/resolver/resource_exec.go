package resolver

import (
	pklExec "github.com/kdeps/schema/gen/exec"
)

// HandleExec manages the execution of an exec block.
func (dr *DependencyResolver) HandleExec(actionID string, execBlock *pklExec.ResourceExec, hasItems bool) error {
	// Run processExecBlock asynchronously in a goroutine
	go func(aID string, block *pklExec.ResourceExec) {
		if err := dr.processExecBlock(aID, block, hasItems); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process exec block", "actionID", aID, "error", err)
		}
	}(actionID, execBlock)

	return nil
}

// processExecBlock processes an execution block and stores it in SQLite.
func (dr *DependencyResolver) processExecBlock(actionID string, execBlock *pklExec.ResourceExec, hasItems bool) error {
	// Store the exec resource in SQLite using the SQLite resource storage method
	return dr.StoreExecResource(actionID, execBlock, hasItems)
}
