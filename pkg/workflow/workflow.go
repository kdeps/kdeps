package workflow

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
func LoadWorkflow(workflowFile string) (*pklWf.Workflow, error) {
	log.Info("Reading workflow file", "workflow-file", workflowFile)

	wf, err := pklWf.LoadFromPath(context.Background(), workflowFile)
	if err != nil {
		return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
	}

	return wf, nil
}
