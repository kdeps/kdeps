package workflow

import (
	"context"
	"fmt"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
func LoadWorkflow(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	logger.Debug("reading workflow file", "workflow-file", workflowFile)

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

	if workflowPtr, ok := module.(*pklWf.WorkflowImpl); ok {
		logger.Debug("successfully read and parsed workflow file", "workflow-file", workflowFile)
		return workflowPtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for workflow file '%s': %T", workflowFile, module)
}
