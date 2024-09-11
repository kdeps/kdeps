package resource

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	pklRes "github.com/kdeps/schema/gen/resource"
)

func LoadResource(resourceFile string) (*pklRes.Resource, error) {
	log.Info("Reading resource file:", "resource-file", resourceFile)

	res, err := pklRes.LoadFromPath(context.Background(), resourceFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading resource-file '%s': %s", resourceFile, err))
	}

	return res, nil
}
