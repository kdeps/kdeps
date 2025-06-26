package resolver

import (
	pklPython "github.com/kdeps/schema/gen/python"
)

// HandlePython manages the execution of a Python block.
func (dr *DependencyResolver) HandlePython(actionID string, pythonBlock *pklPython.ResourcePython, hasItems bool) error {
	// Run processPythonBlock asynchronously in a goroutine
	go func(aID string, block *pklPython.ResourcePython) {
		if err := dr.processPythonBlock(aID, block, hasItems); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process python block", "actionID", aID, "error", err)
		}
	}(actionID, pythonBlock)

	return nil
}

// processPythonBlock processes a Python block and stores it in SQLite.
func (dr *DependencyResolver) processPythonBlock(actionID string, pythonBlock *pklPython.ResourcePython, hasItems bool) error {
	// Store the Python resource in SQLite using the SQLite resource storage method
	return dr.StorePythonResource(actionID, pythonBlock, hasItems)
}
