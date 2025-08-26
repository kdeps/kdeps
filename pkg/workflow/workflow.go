package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
//

func LoadWorkflow(ctx context.Context, workflowFile string, logger *logging.Logger) (wf pklWf.Workflow, err error) {
	logger.Debug("reading workflow file", "workflow-file", workflowFile)

	// Try using the generated LoadFromPath first, but recover from panics
	defer func() {
		if r := recover(); r != nil {
			// If we get a reflection panic, use our workaround
			if errStr, ok := r.(string); ok && strings.Contains(errStr, "reflect.Set: value of type *workflow.WorkflowImpl is not assignable to type workflow.WorkflowImpl") {
				logger.Debug("recovering from pkl-go reflection panic, using workaround", "workflow-file", workflowFile)
				wf, err = loadWorkflowWorkaround(ctx, workflowFile, logger)
				return
			}
			// Re-panic if it's not the specific error we're handling
			panic(r)
		}
	}()

	wf, err = pklWf.LoadFromPath(ctx, workflowFile)
	if err == nil {
		logger.Debug("successfully read and parsed workflow file", "workflow-file", workflowFile)
		return wf, nil
	}

	// Check if it's the specific reflection error we need to work around
	if strings.Contains(err.Error(), "reflect.Set: value of type *workflow.WorkflowImpl is not assignable to type workflow.WorkflowImpl") {
		logger.Debug("working around pkl-go reflection issue", "workflow-file", workflowFile)
		return loadWorkflowWorkaround(ctx, workflowFile, logger)
	}

	// If it's a different error, propagate it
	logger.Error("error reading workflow file", "workflow-file", workflowFile, "error", err)
	return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
}

func loadWorkflowWorkaround(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	// Use direct evaluation as a workaround
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

	// The module should be a *WorkflowImpl which implements the Workflow interface
	if workflowPtr, ok := module.(*pklWf.WorkflowImpl); ok {
		logger.Debug("successfully read and parsed workflow file via workaround", "workflow-file", workflowFile)
		return workflowPtr, nil
	}

	return nil, fmt.Errorf("unexpected module type for workflow file '%s': %T", workflowFile, module)
}
