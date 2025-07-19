package resolver

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/utils"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleExec(actionID string, execBlock *pklExec.ResourceExec) error {
	// Canonicalize the actionID if it's a short ActionID
	canonicalActionID := actionID
	if dr.PklresHelper != nil {
		canonicalActionID = dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Process the exec block synchronously
	if err := dr.DecodeExecBlock(execBlock); err != nil {
		dr.Logger.Error("failed to process exec block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Run processExecBlock synchronously (patch: removed goroutine)
	if err := dr.processExecBlock(canonicalActionID, execBlock); err != nil {
		dr.Logger.Error("failed to process exec block", "actionID", canonicalActionID, "error", err)
		return err
	}

	return nil
}

// ProcessExecBlock processes an exec block - no decoding needed
func (dr *DependencyResolver) DecodeExecBlock(execBlock *pklExec.ResourceExec) error {
	// No processing needed - data is already in proper format
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

	// Write the Exec output to file for pklres access
	if execBlock.Stdout != nil {
		dr.Logger.Debug("processExecBlock: writing stdout to file", "actionID", actionID)
		filePath, err := dr.WriteStdoutToFile(actionID, execBlock.Stdout)
		if err != nil {
			dr.Logger.Error("processExecBlock: failed to write stdout to file", "actionID", actionID, "error", err)
			return fmt.Errorf("failed to write stdout to file: %w", err)
		}
		execBlock.File = &filePath
		dr.Logger.Debug("processExecBlock: wrote stdout to file", "actionID", actionID, "filePath", filePath)
	}

	dr.Logger.Info("processExecBlock: skipping AppendExecEntry - using real-time pklres", "actionID", actionID)
	// Note: AppendExecEntry is no longer needed as we use real-time pklres access
	// The Exec output files are directly accessible through pklres.getResourceOutput()

	// Store the complete exec resource record in the PKL mapping
	if dr.PklresHelper != nil {
		// Create a ResourceExec object for storage
		resourceExec := &pklExec.ResourceExec{
			Command:         execBlock.Command,
			Env:             execBlock.Env,
			Stdout:          execBlock.Stdout,
			Stderr:          execBlock.Stderr,
			ExitCode:        execBlock.ExitCode,
			File:            execBlock.File,
			Timestamp:       execBlock.Timestamp,
			TimeoutDuration: execBlock.TimeoutDuration,
		}

		// Store the resource object using the new method
		// Store exec resource attributes using the new generic approach
		if err := dr.PklresHelper.Set(actionID, "command", resourceExec.Command); err != nil {
			dr.Logger.Error("processExecBlock: failed to store exec resource in pklres", "actionID", actionID, "error", err)
		} else {
			dr.Logger.Info("processExecBlock: stored exec resource in pklres", "actionID", actionID)
		}
	}

	// Mark the resource as finished processing
	// Processing status tracking removed - simplified to pure key-value store approach

	return nil
}

func (dr *DependencyResolver) WriteStdoutToFile(resourceID string, stdout *string) (string, error) {
	if stdout == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	// Write the stdout content directly without base64 decoding
	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(*stdout), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}

// AppendExecEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.

