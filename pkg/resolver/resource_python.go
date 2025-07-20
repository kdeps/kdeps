package resolver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	pklPython "github.com/kdeps/schema/gen/python"
	pklResource "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandlePython(actionID string, pythonBlock *pklPython.ResourcePython) error {
	dr.Logger.Info("HandlePython: ENTRY", "actionID", actionID, "pythonBlock_nil", pythonBlock == nil)
	if pythonBlock != nil {
		dr.Logger.Info("HandlePython: pythonBlock fields", "actionID", actionID, "script_length", len(pythonBlock.Script))
	}
	dr.Logger.Debug("HandlePython: called", "actionID", actionID, "PklresHelper_nil", dr.PklresHelper == nil)

	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Reload the Python resource to ensure PKL templates are evaluated after dependencies are processed
	// This ensures that PKL template expressions like \(client.responseBody("clientResource")) have access to dependency data
	if err := dr.reloadPythonResourceWithDependencies(canonicalActionID, pythonBlock); err != nil {
		dr.Logger.Warn("failed to reload Python resource, continuing with original", "actionID", canonicalActionID, "error", err)
	}

	// Run processPythonBlock synchronously
	if err := dr.processPythonBlock(canonicalActionID, pythonBlock); err != nil {
		dr.Logger.Error("failed to process python block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// reloadPythonResourceWithDependencies reloads the Python resource to ensure PKL templates are evaluated after dependencies
func (dr *DependencyResolver) reloadPythonResourceWithDependencies(actionID string, pythonBlock *pklPython.ResourcePython) error {
	dr.Logger.Debug("reloadPythonResourceWithDependencies: reloading Python resource for fresh template evaluation", "actionID", actionID)

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

	dr.Logger.Debug("reloadPythonResourceWithDependencies: found resource file", "actionID", actionID, "file", resourceFile)

	// Reload the Python resource with fresh PKL template evaluation
	// Load as generic Resource since the Python resource extends Resource.pkl, not Python.pkl
	var reloadedResource interface{}
	var err error
	if dr.APIServerMode {
		reloadedResource, err = dr.LoadResourceWithRequestContextFn(dr.Context, resourceFile, Resource)
	} else {
		reloadedResource, err = dr.LoadResourceFn(dr.Context, resourceFile, Resource)
	}

	if err != nil {
		return fmt.Errorf("failed to reload Python resource: %w", err)
	}

	// Cast to generic Resource first
	reloadedGenericResource, ok := reloadedResource.(*pklResource.Resource)
	if !ok {
		return fmt.Errorf("failed to cast reloaded resource to generic Resource")
	}

	// Extract the Python block from the reloaded resource
	if reloadedGenericResource.Run != nil && reloadedGenericResource.Run.Python != nil {
		reloadedPython := reloadedGenericResource.Run.Python

		// Update the pythonBlock with the reloaded values that contain fresh template evaluation
		if reloadedPython.Script != "" {
			pythonBlock.Script = reloadedPython.Script
			dr.Logger.Debug("reloadPythonResourceWithDependencies: updated script from reloaded resource", "actionID", actionID)
		}

		if reloadedPython.Env != nil {
			pythonBlock.Env = reloadedPython.Env
			dr.Logger.Debug("reloadPythonResourceWithDependencies: updated env from reloaded resource", "actionID", actionID)
		}

		if reloadedPython.PythonEnvironment != nil {
			pythonBlock.PythonEnvironment = reloadedPython.PythonEnvironment
			dr.Logger.Debug("reloadPythonResourceWithDependencies: updated python environment from reloaded resource", "actionID", actionID)
		}
	}

	dr.Logger.Info("reloadPythonResourceWithDependencies: successfully reloaded Python resource with fresh template evaluation", "actionID", actionID)
	return nil
}

func (dr *DependencyResolver) processPythonBlock(actionID string, pythonBlock *pklPython.ResourcePython) error {
	if dr.AnacondaInstalled && pythonBlock.PythonEnvironment != nil && *pythonBlock.PythonEnvironment != "" {
		if err := dr.activateCondaEnvironment(*pythonBlock.PythonEnvironment); err != nil {
			return err
		}

		defer func() {
			_ = dr.deactivateCondaEnvironment()
		}()
	}

	env := dr.formatPythonEnv(pythonBlock.Env)

	tmpFile, err := dr.createPythonTempFile(pythonBlock.Script)
	if err != nil {
		return err
	}
	defer dr.cleanupTempFile(tmpFile.Name())

	dr.Logger.Info("running python", "script", tmpFile.Name(), "env", env)

	cmd := execute.ExecTask{
		Command:     "python3",
		Args:        []string{tmpFile.Name()},
		Shell:       false,
		Env:         env,
		StreamStdio: false,
	}

	var execStdout, execStderr string
	var execExitCode int
	var execErr error
	if dr.ExecTaskRunnerFn != nil {
		execStdout, execStderr, execErr = dr.ExecTaskRunnerFn(dr.Context, cmd)
		// Default exit code for injected runner (assuming success)
		execExitCode = 0
	} else {
		execStdout, execStderr, execExitCode, execErr = kdepsexec.RunExecTask(dr.Context, cmd, dr.Logger, false)
	}
	if execErr != nil {
		// Even if there's an error, we should still record the exit code
		if execExitCode == 0 {
			execExitCode = 1 // Default error exit code
		}
	}

	pythonBlock.Stdout = &execStdout
	pythonBlock.Stderr = &execStderr
	pythonBlock.ExitCode = &execExitCode

	ts := pkl.Duration{
		Value: float64(time.Now().UnixNano()),
		Unit:  pkl.Nanosecond,
	}
	pythonBlock.Timestamp = &ts

	// Store comprehensive resource data in pklres for cross-resource access
	if dr.PklresHelper != nil {
		// Store script
		if err := dr.PklresHelper.Set(actionID, "script", pythonBlock.Script); err != nil {
			dr.Logger.Error("failed to store script in pklres", "actionID", actionID, "error", err)
		}

		// Store stdout for cross-resource access
		if pythonBlock.Stdout != nil {
			if err := dr.PklresHelper.Set(actionID, "stdout", *pythonBlock.Stdout); err != nil {
				dr.Logger.Error("failed to store stdout in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store stderr for cross-resource access
		if pythonBlock.Stderr != nil {
			if err := dr.PklresHelper.Set(actionID, "stderr", *pythonBlock.Stderr); err != nil {
				dr.Logger.Error("failed to store stderr in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store exit code
		if pythonBlock.ExitCode != nil {
			exitCodeStr := fmt.Sprintf("%d", *pythonBlock.ExitCode)
			if err := dr.PklresHelper.Set(actionID, "exitCode", exitCodeStr); err != nil {
				dr.Logger.Error("failed to store exitCode in pklres", "actionID", actionID, "error", err)
			}
		}

		// Store environment variables as JSON for complex structure
		if pythonBlock.Env != nil && len(*pythonBlock.Env) > 0 {
			if envJSON, err := json.Marshal(*pythonBlock.Env); err == nil {
				if err := dr.PklresHelper.Set(actionID, "env", string(envJSON)); err != nil {
					dr.Logger.Error("failed to store env", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("failed to marshal env", "actionID", actionID, "error", err)
			}
		}

		// Store Python environment
		if pythonBlock.PythonEnvironment != nil && *pythonBlock.PythonEnvironment != "" {
			if err := dr.PklresHelper.Set(actionID, "pythonEnvironment", *pythonBlock.PythonEnvironment); err != nil {
				dr.Logger.Error("failed to store pythonEnvironment", "actionID", actionID, "error", err)
			}
		}

		// Store file path if present
		if pythonBlock.File != nil && *pythonBlock.File != "" {
			if err := dr.PklresHelper.Set(actionID, "file", *pythonBlock.File); err != nil {
				dr.Logger.Error("failed to store file", "actionID", actionID, "error", err)
			}
		}

		// Store item values as JSON for complex structure
		if pythonBlock.ItemValues != nil && len(*pythonBlock.ItemValues) > 0 {
			if itemValuesJSON, err := json.Marshal(*pythonBlock.ItemValues); err == nil {
				if err := dr.PklresHelper.Set(actionID, "itemValues", string(itemValuesJSON)); err != nil {
					dr.Logger.Error("failed to store itemValues", "actionID", actionID, "error", err)
				}
			} else {
				dr.Logger.Error("failed to marshal itemValues", "actionID", actionID, "error", err)
			}
		}

		// Store timeout duration
		if pythonBlock.TimeoutDuration != nil {
			timeoutStr := fmt.Sprintf("%g", pythonBlock.TimeoutDuration.Value)
			if err := dr.PklresHelper.Set(actionID, "timeoutDuration", timeoutStr); err != nil {
				dr.Logger.Error("failed to store timeoutDuration", "actionID", actionID, "error", err)
			}
		}

		// Store timestamp
		if pythonBlock.Timestamp != nil {
			timestampStr := fmt.Sprintf("%.0f", pythonBlock.Timestamp.Value)
			if err := dr.PklresHelper.Set(actionID, "timestamp", timestampStr); err != nil {
				dr.Logger.Error("failed to store timestamp in pklres", "actionID", actionID, "error", err)
			}
		}

		dr.Logger.Info("stored comprehensive Python resource attributes in pklres", "actionID", actionID)
	}

	return nil
}

func (dr *DependencyResolver) activateCondaEnvironment(envName string) error {
	execTask := execute.ExecTask{
		Command:     "conda",
		Args:        []string{"activate", "--name", envName},
		Shell:       false,
		StreamStdio: false,
	}

	// Use injected runner if provided
	if dr.ExecTaskRunnerFn != nil {
		if _, _, err := dr.ExecTaskRunnerFn(dr.Context, execTask); err != nil {
			return fmt.Errorf("conda activate failed: %w", err)
		}
		return nil
	}

	_, _, _, err := kdepsexec.RunExecTask(dr.Context, execTask, dr.Logger, false)
	if err != nil {
		return fmt.Errorf("conda activate failed: %w", err)
	}
	return nil
}

func (dr *DependencyResolver) deactivateCondaEnvironment() error {
	execTask := execute.ExecTask{
		Command:     "conda",
		Args:        []string{"deactivate"},
		Shell:       false,
		StreamStdio: false,
	}

	// Use injected runner if provided
	if dr.ExecTaskRunnerFn != nil {
		if _, _, err := dr.ExecTaskRunnerFn(context.Background(), execTask); err != nil {
			return fmt.Errorf("conda deactivate failed: %w", err)
		}
		return nil
	}

	_, _, _, err := kdepsexec.RunExecTask(context.Background(), execTask, dr.Logger, false)
	if err != nil {
		return fmt.Errorf("conda deactivate failed: %w", err)
	}
	return nil
}

func (dr *DependencyResolver) formatPythonEnv(env *map[string]string) []string {
	var formatted []string
	if env != nil {
		for k, v := range *env {
			formatted = append(formatted, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return formatted
}

func (dr *DependencyResolver) createPythonTempFile(script string) (afero.File, error) {
	tmpFile, err := afero.TempFile(dr.Fs, "", "script-*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.Write([]byte(script)); err != nil {
		return nil, fmt.Errorf("failed to write script: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpFile, nil
}

func (dr *DependencyResolver) cleanupTempFile(name string) {
	if err := dr.Fs.Remove(name); err != nil {
		dr.Logger.Error("failed to clean up temp file", "path", name, "error", err)
	}
}
