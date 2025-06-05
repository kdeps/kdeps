package workflow

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/logging"
	pklWf "github.com/kdeps/schema/gen/workflow"
)

// LoadWorkflow reads a workflow file and returns the parsed workflow object or an error.
//
//nolint:ireturn
func LoadWorkflow(ctx context.Context, workflowFile string, logger *logging.Logger) (pklWf.Workflow, error) {
	logger.Debug("reading workflow file", "workflow-file", workflowFile)

	wf, err := pklWf.LoadFromPath(ctx, workflowFile)
	if err != nil {
		logger.Error("error reading workflow file", "workflow-file", workflowFile, "error", err)
		return nil, fmt.Errorf("error reading workflow file '%s': %w", workflowFile, err)
	}

	logger.Debug("successfully read and parsed workflow file", "workflow-file", workflowFile)
	return wf, nil
}

// Execute runs the workflow with the given resource handler
func (w *Workflow) Execute(ctx context.Context, handler ResourceHandler) error {
	// Validate the workflow first
	if err := w.Validate(); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// Execute each resource in the workflow
	for _, resource := range w.Resources {
		// Validate the resource
		if err := resource.Validate(); err != nil {
			return fmt.Errorf("resource validation failed: %w", err)
		}

		// Execute the resource based on its type
		if resource.Run.Python != nil {
			if err := handler.HandlePython(ctx, resource); err != nil {
				return fmt.Errorf("failed to handle Python resource: %w", err)
			}
		}
		if resource.Run.Exec != nil {
			if err := handler.HandleExec(ctx, resource); err != nil {
				return fmt.Errorf("failed to handle Exec resource: %w", err)
			}
		}
		if resource.Run.HTTPClient != nil {
			if err := handler.HandleHTTPClient(ctx, resource); err != nil {
				return fmt.Errorf("failed to handle HTTP client resource: %w", err)
			}
		}
		if resource.Run.LLM != nil {
			if err := handler.HandleLLM(ctx, resource); err != nil {
				return fmt.Errorf("failed to handle LLM resource: %w", err)
			}
		}
	}

	return nil
}

// Validate checks if the workflow configuration is valid
func (w *Workflow) Validate() error {
	if w.Version == "" {
		return fmt.Errorf("workflow version is required")
	}

	if w.Settings != nil && w.Settings.APIServer != nil {
		if w.Settings.APIServer.PortNum <= 0 || w.Settings.APIServer.PortNum > 65535 {
			return fmt.Errorf("invalid port number: %d", w.Settings.APIServer.PortNum)
		}
	}

	return nil
}

// Validate checks if the resource configuration is valid
func (r *Resource) Validate() error {
	if r.ActionID == "" {
		return fmt.Errorf("resource action ID is required")
	}

	if r.Run == nil {
		return fmt.Errorf("resource run block is required")
	}

	// Validate the run block
	if err := r.Run.Validate(); err != nil {
		return fmt.Errorf("invalid run block: %w", err)
	}

	return nil
}

// Validate checks if the resource run configuration is valid
func (r *ResourceRun) Validate() error {
	// Check if at least one action is defined
	hasAction := r.Python != nil || r.Exec != nil || r.HTTPClient != nil || r.LLM != nil || r.APIResponse != nil
	if !hasAction {
		return fmt.Errorf("at least one action must be defined in the run block")
	}

	return nil
}
