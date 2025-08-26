package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(ctx context.Context, resourceFile string, logger *logging.Logger) (*pklRes.Resource, error) {
	// Log additional info before reading the resource
	logger.Debug("reading resource file", "resource-file", resourceFile)

	// Try using the generated LoadFromPath first
	res, err := pklRes.LoadFromPath(ctx, resourceFile)
	if err == nil {
		logger.Debug("successfully loaded resource", "resource-file", resourceFile)
		return res, nil
	}

	// Check if it's the specific reflection error we need to work around
	if strings.Contains(err.Error(), "reflect.Set: value of type *resource.Resource is not assignable to type resource.Resource") {
		logger.Debug("working around pkl-go reflection issue", "resource-file", resourceFile)
		
		// Use direct evaluation as a workaround
		evaluator, evalErr := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
		if evalErr != nil {
			logger.Error("error creating pkl evaluator", "resource-file", resourceFile, "error", evalErr)
			return nil, fmt.Errorf("error creating pkl evaluator for resource file '%s': %w", resourceFile, evalErr)
		}
		defer evaluator.Close()

		source := pkl.FileSource(resourceFile)
		var module interface{}
		evalErr = evaluator.EvaluateModule(ctx, source, &module)
		if evalErr != nil {
			logger.Error("error reading resource file", "resource-file", resourceFile, "error", evalErr)
			return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, evalErr)
		}

		// The module should be a *Resource
		if resourcePtr, ok := module.(*pklRes.Resource); ok {
			logger.Debug("successfully loaded resource via workaround", "resource-file", resourceFile)
			return resourcePtr, nil
		}

		return nil, fmt.Errorf("unexpected module type for resource file '%s': %T", resourceFile, module)
	}

	// If it's a different error, propagate it
	logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
	return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
}
