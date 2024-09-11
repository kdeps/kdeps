package resource

import (
	"context"
	"fmt"
	"kdeps/pkg/logging"

	pklRes "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(resourceFile string) (*pklRes.Resource, error) {
	// Log the beginning of the resource loading process
	logging.Debug("Starting to load resource", "resource-file", resourceFile)

	// Log additional info before reading the resource
	logging.Info("Reading resource file", "resource-file", resourceFile)

	// Attempt to load the resource from the file path
	res, err := pklRes.LoadFromPath(context.Background(), resourceFile)
	if err != nil {
		// Log the error with debug info if something goes wrong
		logging.Error("Error reading resource file", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
	}

	// Log successful completion of resource loading
	logging.Debug("Successfully loaded resource", "resource-file", resourceFile)

	return res, nil
}
