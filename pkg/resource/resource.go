package resource

import (
	"context"
	"fmt"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(ctx context.Context, resourceFile string, logger *logging.Logger) (*pklRes.Resource, error) {
	logger.Debug("reading resource file", "resource-file", resourceFile)

	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for resource file '%s': %w", resourceFile, err)
	}
	defer evaluator.Close()

	source := pkl.FileSource(resourceFile)
	var module interface{}
	err = evaluator.EvaluateModule(ctx, source, &module)
	if err != nil {
		logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
	}

	if resourcePtr, ok := module.(*pklRes.Resource); ok {
		logger.Debug("successfully loaded resource", "resource-file", resourceFile)
		return resourcePtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for resource file '%s': %T", resourceFile, module)
}
