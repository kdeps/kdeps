package resolver

import (
	"fmt"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/schema"
	"kdeps/pkg/utils"
	"path/filepath"
	"strings"

	pklData "github.com/kdeps/schema/gen/data"
	"github.com/spf13/afero"
)

// AppendDataEntry appends a data entry to the existing files map
// AppendDataEntry appends a data entry to the existing files map
func (dr *DependencyResolver) AppendDataEntry(resourceId string, newData *pklData.DataImpl) error {
	// Ensure dr.Context is not nil
	if dr.Context == nil {
		return fmt.Errorf("context is nil")
	}

	// Define the path to the PKL file
	pklPath := filepath.Join(dr.ActionDir, "data/"+dr.RequestId+"__data_output.pkl")

	// Load existing PKL data
	pklRes, err := pklData.LoadFromPath(*dr.Context, pklPath)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	// Safeguard against nil pointers
	if pklRes == nil || pklRes.GetFiles() == nil {
		return fmt.Errorf("PKL data or files map is nil")
	}

	// Get the existing files map
	existingFiles := pklRes.GetFiles() // Pointer to the map

	// Ensure newData is not nil
	if newData == nil || newData.Files == nil {
		return fmt.Errorf("new data or its files map is nil")
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

	// Build the new PKL content
	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Data.pkl\"\n\n", schema.SchemaVersion))
	pklContent.WriteString("files {\n")

	for agentName, baseFileMap := range *existingFiles {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", agentName))
		for baseFilename, filePath := range baseFileMap {
			pklContent.WriteString(fmt.Sprintf("    [\"%s\"] = \"%s\"\n", baseFilename, filePath))
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
	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, pklPath, fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Data.pkl\"", schema.SchemaVersion), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL file: %w", err)
	}

	// Rebuild the PKL content with the evaluated content
	err = afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write evaluated content to PKL file: %w", err)
	}

	return nil
}
