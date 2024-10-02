package resource

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(ctx context.Context, resourceFile string, logger *log.Logger) (*pklRes.Resource, error) {
	// Log additional info before reading the resource
	logger.Info("Reading resource file", "resource-file", resourceFile)

	// Attempt to load the resource from the file path
	res, err := pklRes.LoadFromPath(ctx, resourceFile)
	if err != nil {
		// Log the error with debug info if something goes wrong
		logger.Error("Error reading resource file", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
	}

	// Log successful completion of resource loading
	logger.Info("Successfully loaded resource", "resource-file", resourceFile)

	return res, nil
}
