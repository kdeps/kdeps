package resource

import (
	"context"
	"fmt"
	"os"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/assets"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaAssets "github.com/kdeps/schema/assets"
	pklResource "github.com/kdeps/schema/gen/resource"
)

// LoadResource reads a resource file and returns the parsed resource object or an error.
func LoadResource(ctx context.Context, resourceFile string, logger *logging.Logger) (*pklResource.Resource, error) {
	logger.Debug("reading resource file", "resource-file", resourceFile)

	// Check if we should use embedded assets
	if assets.ShouldUseEmbeddedAssets() {
		return loadResourceFromEmbeddedAssets(ctx, resourceFile, logger)
	}

	return loadResourceFromFile(ctx, resourceFile, logger)
}

// loadResourceFromEmbeddedAssets loads resource using embedded PKL assets
func loadResourceFromEmbeddedAssets(ctx context.Context, resourceFile string, logger *logging.Logger) (*pklResource.Resource, error) {
	logger.Debug("loading resource from embedded assets", "resource-file", resourceFile)

	// Copy assets to temp directory with URL conversion
	tempDir, err := schemaAssets.CopyAssetsToTempDirWithConversion()
	if err != nil {
		logger.Error("error copying assets to temp directory", "error", err)
		return nil, fmt.Errorf("error copying assets to temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for resource file '%s': %w", resourceFile, err)
	}
	defer evaluator.Close()

	// Use the user's resource file but with embedded asset support
	source := pkl.FileSource(resourceFile)
	var module interface{}
	err = evaluator.EvaluateModule(ctx, source, &module)
	if err != nil {
		logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
		return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
	}

	if resourcePtr, ok := module.(*pklResource.Resource); ok {
		logger.Debug("successfully loaded resource from embedded assets", "resource-file", resourceFile)
		return resourcePtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for resource file '%s': %T", resourceFile, module)
}

// loadResourceFromFile loads resource using direct file evaluation (original method)
func loadResourceFromFile(ctx context.Context, resourceFile string, logger *logging.Logger) (*pklResource.Resource, error) {
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

	if resourcePtr, ok := module.(*pklResource.Resource); ok {
		logger.Debug("successfully loaded resource", "resource-file", resourceFile)
		return resourcePtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for resource file '%s': %T", resourceFile, module)
}
