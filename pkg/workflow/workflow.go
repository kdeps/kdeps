package workflow

import (
	"context"
	"fmt"
	"os"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/assets"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaAssets "github.com/kdeps/schema/assets"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
func LoadWorkflow(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	logger.Debug("reading workflow file", "workflow-file", workflowFile)

	// Check if we should use embedded assets
	if assets.ShouldUseEmbeddedAssets() {
		return loadWorkflowFromEmbeddedAssets(ctx, workflowFile, logger)
	}

	return loadWorkflowFromFile(ctx, workflowFile, logger)
}

// loadWorkflowFromEmbeddedAssets loads workflow using embedded PKL assets
func loadWorkflowFromEmbeddedAssets(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	logger.Debug("loading workflow from embedded assets", "workflow-file", workflowFile)

	// Copy assets to temp directory with URL conversion
	tempDir, err := schemaAssets.CopyAssetsToTempDirWithConversion()
	if err != nil {
		logger.Error("error copying assets to temp directory", "error", err)
		return nil, fmt.Errorf("error copying assets to temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for workflow file '%s': %w", workflowFile, err)
	}
	defer evaluator.Close()

	// Use the user's workflow file but with embedded asset support
	source := pkl.FileSource(workflowFile)
	var module interface{}
	err = evaluator.EvaluateModule(ctx, source, &module)
	if err != nil {
		logger.Error("error reading workflow file", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
	}

	if workflowPtr, ok := module.(pklWf.Workflow); ok {
		logger.Debug("successfully read and parsed workflow file from embedded assets", "workflow-file", workflowFile)
		return workflowPtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for workflow file '%s': %T", workflowFile, module)
}

// loadWorkflowFromFile loads workflow using direct file evaluation (original method)
func loadWorkflowFromFile(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	evaluator, err := pkl.NewEvaluator(ctx, pkl.PreconfiguredOptions)
	if err != nil {
		logger.Error("error creating pkl evaluator", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error creating pkl evaluator for workflow file '%s': %w", workflowFile, err)
	}
	defer evaluator.Close()

	source := pkl.FileSource(workflowFile)
	var module interface{}
	err = evaluator.EvaluateModule(ctx, source, &module)
	if err != nil {
		logger.Error("error reading workflow file", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
	}

	if workflowPtr, ok := module.(pklWf.Workflow); ok {
		logger.Debug("successfully read and parsed workflow file", "workflow-file", workflowFile)
		return workflowPtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for workflow file '%s': %T", workflowFile, module)
}
