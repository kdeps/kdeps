package resolver

import (
	"fmt"

	"github.com/kdeps/kdeps/pkg/data"
	pklData "github.com/kdeps/schema/gen/data"
)

// HandleData processes data resources and populates the Files mapping registry
func (dr *DependencyResolver) HandleData(actionID string, dataBlock *pklData.DataImpl) error {
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Process data block to populate Files mapping
	if err := dr.processDataBlock(canonicalActionID, dataBlock); err != nil {
		dr.Logger.Error("failed to process data block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// processDataBlock processes a data block and populates the Files mapping registry
func (dr *DependencyResolver) processDataBlock(actionID string, dataBlock *pklData.DataImpl) error {
	// Populate the Files mapping registry using the data package
	if dataBlock.Files == nil {
		dataBlock.Files = make(map[string]map[string]string)
	}

	// Use the data package to populate the file registry
	fileRegistry, err := data.PopulateDataFileRegistry(dr.Fs, dr.DataDir)
	if err != nil {
		dr.Logger.Error("failed to populate data file registry", "actionID", actionID, "error", err)
		return fmt.Errorf("failed to populate data file registry: %w", err)
	}

	// Convert the file registry to the expected format
	for agentVersion, files := range *fileRegistry {
		dataBlock.Files[agentVersion] = files
	}

	// Store the complete data resource record in the PKL mapping
	if dr.PklresHelper != nil {
		// Create a DataImpl object for storage
		resourceData := &pklData.DataImpl{
			Files: dataBlock.Files,
		}

		// Store the resource object using the new method
		// Store data resource attributes using the new generic approach
		if err := dr.PklresHelper.Set(actionID, "files", fmt.Sprintf("%v", resourceData.Files)); err != nil {
			dr.Logger.Error("processDataBlock: failed to store data resource in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("processDataBlock: stored data resource in pklres", "actionID", actionID)
		}
	}

	dr.Logger.Info("processDataBlock: completed successfully", "actionID", actionID, "fileCount", len(dataBlock.Files))

	// Mark the resource as finished processing
	// Processing status tracking removed - simplified to pure key-value store approach

	return nil
}

// AppendDataEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.
// Data resources are now handled differently - they don't store PKL content but rather
// manage file references directly through the agent system.

// Exported for testing
var (
	FormatValue     = formatValue
	FormatErrors    = formatErrors
	FormatDataValue = formatDataValue
)
