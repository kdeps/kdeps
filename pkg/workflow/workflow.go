package workflow

import (
	"context"
	"fmt"

	"kdeps/pkg/logging"

	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
func LoadWorkflow(ctx context.Context, workflowFile string) (*pklWf.Workflow, error) {
	logging.Info("Reading workflow file", "workflow-file", workflowFile)

	wf, err := pklWf.LoadFromPath(ctx, workflowFile)
	if err != nil {
		logging.Error("Error reading workflow file", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
	}

	logging.Info("Successfully read and parsed workflow file", "workflow-file", workflowFile)
	return wf, nil
}
