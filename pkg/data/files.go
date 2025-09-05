package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// PopulateDataFileRegistry populates a registry of files categorized by agentName/version.
// It returns a map where the key is "agentName/version", and the value is another map
// mapping relative paths within the version directory to their full paths.
func PopulateDataFileRegistry(fs afero.Fs, baseDir string) (*map[string]map[string]string, error) {
	// Initialize the registry
	files := make(map[string]map[string]string)
	separator := string(filepath.Separator) // Use constant for clarity

	// Check if the base directory exists
	baseDirExists, err := afero.DirExists(fs, baseDir)
	if err != nil {
		return &files, fmt.Errorf("error checking existence of base directory %s: %w", baseDir, err)
	}
	if !baseDirExists {
		// If the directory does not exist, return an empty registry
		return &files, nil
	}

	// Walk through the base directory
	err = afero.Walk(fs, baseDir, func(path string, info os.FileInfo, walkErr error) error {
		// If there was an error accessing this file, skip it and continue walking
		if walkErr != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get the relative path from the base directory
		relPath, walkRelErr := filepath.Rel(baseDir, path)
		if walkRelErr != nil {
			return fmt.Errorf("error computing relative path for %s: %w", path, walkRelErr)
		}

		// Split the relative path into components
		parts := strings.Split(relPath, separator)
		const minPartsRequired = 2
		if len(parts) < minPartsRequired {
			// Skip entries without at least agentName and version
			return nil
		}

		// Extract agent name and version
		agentName := filepath.Join(parts[0], parts[1])

		// Construct the key (relative path within the version directory)
		key := strings.Join(parts[2:], separator)

		// Ensure the map for this agent exists
		if _, agentExists := files[agentName]; !agentExists {
			files[agentName] = make(map[string]string)
		}

		// Map the key to the full path
		files[agentName][key] = path

		return nil
	})
	// If walking fails entirely (e.g., directory read error), return the error
	if err != nil {
		return &files, fmt.Errorf("error walking directory %s: %w", baseDir, err)
	}

	return &files, nil
}
