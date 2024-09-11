package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

func LoadWorkflow(workflowFile string) (*pklWf.Workflow, error) {
	log.Info("Reading workflow file:", "workflow-file", workflowFile)

	wf, err := pklWf.LoadFromPath(context.Background(), workflowFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading workflow-file '%s': %s", workflowFile, err))
	}

	return wf, nil
}
