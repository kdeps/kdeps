package resolver

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/utils"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandlePython(actionID string, pythonBlock *pklPython.ResourcePython) error {
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Decode the python block synchronously
	if err := dr.decodePythonBlock(pythonBlock); err != nil {
		dr.Logger.Error("failed to decode python block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Run processPythonBlock asynchronously in a goroutine
	go func(aID string, block *pklPython.ResourcePython) {
		if err := dr.processPythonBlock(aID, block); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process python block", "actionID", aID, "error", err)
		}
	}(canonicalActionID, pythonBlock)

	// Return immediately; the python block is being processed in the background.
	return nil
}

func (dr *DependencyResolver) decodePythonBlock(pythonBlock *pklPython.ResourcePython) error {
	// Decode Script
	decodedScript, err := utils.DecodeBase64IfNeeded(pythonBlock.Script)
	if err != nil {
		return fmt.Errorf("failed to decode script: %w", err)
	}
	pythonBlock.Script = decodedScript

	// Decode Stderr
	if pythonBlock.Stderr != nil {
		decodedStderr, err := utils.DecodeBase64IfNeeded(*pythonBlock.Stderr)
		if err != nil {
			return fmt.Errorf("failed to decode stderr: %w", err)
		}
		pythonBlock.Stderr = &decodedStderr
	}

	// Decode Stdout
	if pythonBlock.Stdout != nil {
		decodedStdout, err := utils.DecodeBase64IfNeeded(*pythonBlock.Stdout)
		if err != nil {
			return fmt.Errorf("failed to decode stdout: %w", err)
		}
		pythonBlock.Stdout = &decodedStdout
	}

	// Decode Env
	decodedEnv, err := utils.DecodeStringMap(pythonBlock.Env, "env")
	if err != nil {
		return fmt.Errorf("failed to decode env: %w", err)
	}
	pythonBlock.Env = decodedEnv

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
		Value: float64(time.Now().Unix()),
		Unit:  pkl.Nanosecond,
	}
	pythonBlock.Timestamp = &ts

	// Write the Python output to file for pklres access
	if pythonBlock.Stdout != nil {
		dr.Logger.Debug("processPythonBlock: writing stdout to file", "actionID", actionID)
		filePath, err := dr.WritePythonOutputToFile(actionID, pythonBlock.Stdout)
		if err != nil {
			dr.Logger.Error("processPythonBlock: failed to write stdout to file", "actionID", actionID, "error", err)
			return fmt.Errorf("failed to write stdout to file: %w", err)
		}
		pythonBlock.File = &filePath
		dr.Logger.Debug("processPythonBlock: wrote stdout to file", "actionID", actionID, "filePath", filePath)
	}

	dr.Logger.Info("processPythonBlock: skipping AppendPythonEntry - using real-time pklres", "actionID", actionID)
	// Note: AppendPythonEntry is no longer needed as we use real-time pklres access
	// The Python output files are directly accessible through pklres.getResourceOutput()

	// Store the complete python resource record in the PKL mapping
	if dr.PklresHelper != nil {
		// Create a ResourcePython object for storage
		resourcePython := &pklPython.ResourcePython{
			Script:            pythonBlock.Script,
			Env:               pythonBlock.Env,
			Stdout:            pythonBlock.Stdout,
			Stderr:            pythonBlock.Stderr,
			ExitCode:          pythonBlock.ExitCode,
			File:              pythonBlock.File,
			Timestamp:         pythonBlock.Timestamp,
			TimeoutDuration:   pythonBlock.TimeoutDuration,
			PythonEnvironment: pythonBlock.PythonEnvironment,
		}

		// Store the resource record using the new method
		if err := dr.PklresHelper.StoreResourceRecord("python", actionID, actionID, fmt.Sprintf("%+v", resourcePython)); err != nil {
			dr.Logger.Error("processPythonBlock: failed to store python resource in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("processPythonBlock: stored python resource in pklres", "actionID", actionID)
		}
	}

	return nil
}

func (dr *DependencyResolver) WritePythonOutputToFile(resourceID string, stdoutEncoded *string) (string, error) {
	if stdoutEncoded == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	content, err := utils.DecodeBase64IfNeeded(*stdoutEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode python stdout: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return outputFilePath, nil
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

func (dr *DependencyResolver) WritePythonStdoutToFile(resourceID string, stdoutEncoded *string) (string, error) {
	if stdoutEncoded == nil {
		return "", nil
	}

	content, err := utils.DecodeBase64IfNeeded(*stdoutEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode stdout: %w", err)
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write stdout to file: %w", err)
	}

	return outputFilePath, nil
}

// AppendPythonEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.

func (dr *DependencyResolver) EncodePythonEnv(env *map[string]string) *map[string]string {
	if env == nil {
		return nil
	}
	encoded := make(map[string]string)
	for k, v := range *env {
		encoded[k] = utils.EncodeValue(v)
	}
	return &encoded
}

func (dr *DependencyResolver) EncodePythonOutputs(stderr, stdout *string) (*string, *string) {
	return utils.EncodeValuePtr(stderr), utils.EncodeValuePtr(stdout)
}

func (dr *DependencyResolver) EncodePythonStderr(stderr *string) string {
	if stderr == nil {
		return "    Stderr = \"\"\n"
	}
	return fmt.Sprintf("    Stderr = #\"\"\"\n%s\n\"\"\"#\n", *stderr)
}

func (dr *DependencyResolver) EncodePythonStdout(stdout *string) string {
	if stdout == nil {
		return "    Stdout = \"\"\n"
	}
	return fmt.Sprintf("    Stdout = #\"\"\"\n%s\n\"\"\"#\n", *stdout)
}
