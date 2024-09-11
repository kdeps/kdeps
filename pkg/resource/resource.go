package resource

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(resourceFile string) (*pklRes.Resource, error) {
	log.Info("Reading resource file", "resource-file", resourceFile)

	res, err := pklRes.LoadFromPath(context.Background(), resourceFile)
	if err != nil {
		return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
	}

	return res, nil
}
