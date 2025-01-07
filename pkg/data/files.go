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

	// Walk through the base directory
	err := afero.Walk(fs, baseDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("error accessing path %s: %w", path, walkErr)
		}

		// Get the relative path from the base directory
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}

		// Split the relative path into components
		parts := strings.Split(relPath, separator)
		if len(parts) < 2 {
			// Skip entries without at least agentName and version
			return nil
		}

		// Extract agent name and version
		agentName := filepath.Join(parts[0], parts[1])

		// Construct the key (relative path within the version directory)
		key := strings.Join(parts[2:], separator)

		// Ensure the map for this agent exists
		if _, exists := files[agentName]; !exists {
			files[agentName] = make(map[string]string)
		}

		// Map the key to the full path
		files[agentName][key] = path

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk through directory %s: %w", baseDir, err)
	}

	return &files, nil
}
