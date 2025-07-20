package resolver

import (
	"fmt"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklResource "github.com/kdeps/schema/gen/resource"
)

func (dr *DependencyResolver) HandleExec(actionID string, execBlock *pklExec.ResourceExec) error {
	dr.Logger.Info("HandleExec: ENTRY", "actionID", actionID, "execBlock_nil", execBlock == nil)
	if execBlock != nil {
		dr.Logger.Info("HandleExec: execBlock fields", "actionID", actionID, "command", execBlock.Command)
	}
	dr.Logger.Debug("HandleExec: called", "actionID", actionID, "PklresHelper_nil", dr.PklresHelper == nil)

	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Reload the Exec resource to ensure PKL templates are evaluated after dependencies are processed
	// This ensures that PKL template expressions like \(client.responseBody("clientResource")) have access to dependency data
	if err := dr.reloadExecResourceWithDependencies(canonicalActionID, execBlock); err != nil {
		dr.Logger.Warn("failed to reload Exec resource, continuing with original", "actionID", canonicalActionID, "error", err)
	}

	// Run processExecBlock synchronously
	if err := dr.processExecBlock(canonicalActionID, execBlock); err != nil {
		dr.Logger.Error("failed to process exec block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// reloadExecResourceWithDependencies reloads the Exec resource to ensure PKL templates are evaluated after dependencies
func (dr *DependencyResolver) reloadExecResourceWithDependencies(actionID string, execBlock *pklExec.ResourceExec) error {
	dr.Logger.Debug("reloadExecResourceWithDependencies: reloading Exec resource for fresh template evaluation", "actionID", actionID)

	// Find the resource file path for this actionID
	resourceFile := ""
	for _, res := range dr.Resources {
		if res.ActionID == actionID {
			resourceFile = res.File
			break
		}
	}

	if resourceFile == "" {
		return fmt.Errorf("could not find resource file for actionID: %s", actionID)
	}

	dr.Logger.Debug("reloadExecResourceWithDependencies: found resource file", "actionID", actionID, "file", resourceFile)

	// Reload the Exec resource with fresh PKL template evaluation
	// Load as generic Resource since the Exec resource extends Resource.pkl, not Exec.pkl
	var reloadedResource interface{}
	var err error
	if dr.APIServerMode {
		reloadedResource, err = dr.LoadResourceWithRequestContextFn(dr.Context, resourceFile, Resource)
	} else {
		reloadedResource, err = dr.LoadResourceFn(dr.Context, resourceFile, Resource)
	}

	if err != nil {
		return fmt.Errorf("failed to reload Exec resource: %w", err)
	}

	// Cast to generic Resource first
	reloadedGenericResource, ok := reloadedResource.(*pklResource.Resource)
	if !ok {
		return fmt.Errorf("failed to cast reloaded resource to generic Resource")
	}

	// Extract the Exec block from the reloaded resource
	if reloadedGenericResource.Run != nil && reloadedGenericResource.Run.Exec != nil {
		reloadedExec := reloadedGenericResource.Run.Exec

		// Update the execBlock with the reloaded values that contain fresh template evaluation
		if reloadedExec.Command != "" {
			execBlock.Command = reloadedExec.Command
			dr.Logger.Debug("reloadExecResourceWithDependencies: updated command from reloaded resource", "actionID", actionID)
		}

		if reloadedExec.Env != nil {
			execBlock.Env = reloadedExec.Env
			dr.Logger.Debug("reloadExecResourceWithDependencies: updated env from reloaded resource", "actionID", actionID)
		}
	}

	dr.Logger.Info("reloadExecResourceWithDependencies: successfully reloaded Exec resource with fresh template evaluation", "actionID", actionID)
	return nil
}

func (dr *DependencyResolver) processExecBlock(actionID string, execBlock *pklExec.ResourceExec) error {
	var env []string
	if execBlock.Env != nil {
		for key, value := range *execBlock.Env {
			env = append(env, fmt.Sprintf(`%s=%s`, key, value))
		}
	}

	dr.Logger.Info("executing command", "command", execBlock.Command, "env", env)

	task := execute.ExecTask{
		Command:     execBlock.Command,
		Shell:       true,
		Env:         env,
		StreamStdio: false,
	}

	var stdout, stderr string
	var exitCode int
	var err error
	if dr.ExecTaskRunnerFn != nil {
		stdout, stderr, err = dr.ExecTaskRunnerFn(dr.Context, task)
		// Default exit code for injected runner (assuming success)
		exitCode = 0
	} else {
		// fallback direct execution via kdepsexec
		stdout, stderr, exitCode, err = kdepsexec.RunExecTask(dr.Context, task, dr.Logger, false)
	}
	if err != nil {
		// Even if there's an error, we should still record the exit code
		if exitCode == 0 {
			exitCode = 1 // Default error exit code
		}
	}

	execBlock.Stdout = &stdout
	execBlock.Stderr = &stderr
	execBlock.ExitCode = &exitCode

	ts := pkl.Duration{
		Value: float64(time.Now().UnixNano()),
		Unit:  pkl.Nanosecond,
	}
	execBlock.Timestamp = &ts

	// Store resource data in pklres for cross-resource access
	if dr.PklresHelper != nil {
		// Store command
		if err := dr.PklresHelper.Set(actionID, "command", execBlock.Command); err != nil {
			dr.Logger.Error("failed to store command in pklres", "actionID", actionID, "error", err)
		}

		// Store stdout for cross-resource access
		if execBlock.Stdout != nil {
			if err := dr.PklresHelper.Set(actionID, "stdout", *execBlock.Stdout); err != nil {
				dr.Logger.Error("failed to store stdout in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store stderr for cross-resource access
		if execBlock.Stderr != nil {
			if err := dr.PklresHelper.Set(actionID, "stderr", *execBlock.Stderr); err != nil {
				dr.Logger.Error("failed to store stderr in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store exit code
		if execBlock.ExitCode != nil {
			exitCodeStr := fmt.Sprintf("%d", *execBlock.ExitCode)
			if err := dr.PklresHelper.Set(actionID, "exitCode", exitCodeStr); err != nil {
				dr.Logger.Error("failed to store exitCode in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store timestamp
		if execBlock.Timestamp != nil {
			timestampStr := fmt.Sprintf("%.0f", execBlock.Timestamp.Value)
			if err := dr.PklresHelper.Set(actionID, "timestamp", timestampStr); err != nil {
				dr.Logger.Error("failed to store timestamp in pklres", "actionID", actionID, "error", err)
			}
		}

		dr.Logger.Info("stored exec resource in pklres", "actionID", actionID)
	}

	return nil
}
