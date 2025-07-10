package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklData "github.com/kdeps/schema/gen/data"
)

// AppendDataEntry appends a data entry to the existing files map.
func (dr *DependencyResolver) AppendDataEntry(resourceID string, newData *pklData.DataImpl) error {
	// Ensure dr.Context is not nil
	if dr.Context == nil {
		return errors.New("context is nil")
	}

	// Use pklres path instead of file path
	pklPath := dr.PklresHelper.getResourcePath("data")

	// Load existing PKL data from pklres
	var pklRes *pklData.DataImpl
	res, err := dr.LoadResource(dr.Context, pklPath, ResourceType("data"))
	if err != nil {
		// If loading fails, create a new empty data structure
		emptyFiles := make(map[string]map[string]string)
		pklRes = &pklData.DataImpl{
			Files: &emptyFiles,
		}
	} else {
		var ok bool
		pklRes, ok = res.(*pklData.DataImpl)
		if !ok {
			// Fallback to empty structure if casting fails
			emptyFiles := make(map[string]map[string]string)
			pklRes = &pklData.DataImpl{
				Files: &emptyFiles,
			}
		}
	}

	// Safeguard against nil pointers - create empty structure if needed
	var existingFiles *map[string]map[string]string
	if pklRes.GetFiles() == nil {
		emptyFiles := make(map[string]map[string]string)
		existingFiles = &emptyFiles
	} else {
		existingFiles = pklRes.GetFiles()
	}

	// Ensure newData is not nil
	if newData == nil || newData.Files == nil {
		return errors.New("new data or its files map is nil")
	}

	// Merge new data into the existing files map
	for agentName, baseFileMap := range *newData.Files {
		// Ensure the agent name exists in the existing files map
		if _, exists := (*existingFiles)[agentName]; !exists {
			(*existingFiles)[agentName] = make(map[string]string)
		}

		// Merge and encode base filenames and file paths
		for baseFilename, filePath := range baseFileMap {
			if !utils.IsBase64Encoded(filePath) {
				filePath = utils.EncodeBase64String(filePath)
			}
			(*existingFiles)[agentName][baseFilename] = filePath
		}
	}

	// Store the PKL content using pklres (no JSON, no custom serialization)
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Data.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	// Inject the requestID as a variable accessible to PKL functions
	pklContent.WriteString(fmt.Sprintf("/// Current request ID for pklres operations\nrequestID = \"%s\"\n\n", dr.RequestID))
	pklContent.WriteString("Files {\n")

	for agentName, baseFileMap := range *existingFiles {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", agentName))
		for baseFilename, filePath := range baseFileMap {
			pklContent.WriteString(fmt.Sprintf("    [\"%s\"] = \"%s\"\n", baseFilename, filePath))
		}
		pklContent.WriteString("  }\n")
	}

	pklContent.WriteString("}\n")

	// Store the PKL content using pklres instead of writing to file
	if err := dr.PklresHelper.storePklContent("data", resourceID, pklContent.String()); err != nil {
		return fmt.Errorf("failed to store PKL content in pklres: %w", err)
	}

	return nil
}
