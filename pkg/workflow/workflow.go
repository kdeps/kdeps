package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

func LoadConfiguration(fs afero.Fs, workflowFile string) (*workflow.Workflow, error) {
	log.Info("Reading workflow file:", "workflow-file", workflowFile)

	wf, err := workflow.LoadFromPath(context.Background(), workflowFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading workflow-file '%s': %s", workflowFile, err))
	}

	return wf, nil
}
