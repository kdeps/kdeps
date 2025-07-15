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

	// Decode the exec block synchronously
	if err := dr.DecodeExecBlock(execBlock); err != nil {
		dr.Logger.Error("failed to decode exec block", "actionID", canonicalActionID, "error", err)
		return err
	}

	// Run processExecBlock asynchronously in a goroutine
	go func(aID string, block *pklExec.ResourceExec) {
		if err := dr.processExecBlock(aID, block); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process exec block", "actionID", aID, "error", err)
		}
	}(canonicalActionID, execBlock)

	// Return immediately; the exec block is being processed in the background.
	return nil
}

// DecodeExecBlock decodes an exec block by processing base64 encoded fields
func (dr *DependencyResolver) DecodeExecBlock(execBlock *pklExec.ResourceExec) error {
	// Decode Command
	decodedCommand, err := utils.DecodeBase64IfNeeded(execBlock.Command)
	if err != nil {
		return fmt.Errorf("failed to decode command: %w", err)
	}
	execBlock.Command = decodedCommand

	// Decode Stderr
	if execBlock.Stderr != nil {
		decodedStderr, err := utils.DecodeBase64IfNeeded(*execBlock.Stderr)
		if err != nil {
			return fmt.Errorf("failed to decode stderr: %w", err)
		}
		execBlock.Stderr = &decodedStderr
	}

	// Decode Stdout
	if execBlock.Stdout != nil {
		decodedStdout, err := utils.DecodeBase64IfNeeded(*execBlock.Stdout)
		if err != nil {
			return fmt.Errorf("failed to decode stdout: %w", err)
		}
		execBlock.Stdout = &decodedStdout
	}

	// Decode Env values
	decodedEnv, err := utils.DecodeStringMap(execBlock.Env, "env")
	if err != nil {
		return fmt.Errorf("failed to decode env: %w", err)
	}
	execBlock.Env = decodedEnv

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
		Value: float64(time.Now().Unix()),
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
	return nil
}

func (dr *DependencyResolver) WriteStdoutToFile(resourceID string, stdoutEncoded *string) (string, error) {
	if stdoutEncoded == nil {
		return "", nil
	}

	resourceIDFile := utils.GenerateResourceIDFilename(resourceID, dr.RequestID)
	outputFilePath := filepath.Join(dr.FilesDir, resourceIDFile)

	content, err := utils.DecodeBase64IfNeeded(*stdoutEncoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode stdout: %w", err)
	}

	if err := afero.WriteFile(dr.Fs, outputFilePath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFilePath, nil
}

// AppendExecEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.

func (dr *DependencyResolver) encodeExecEnv(env *map[string]string) *map[string]string {
	if env == nil {
		return nil
	}
	encoded := make(map[string]string)
	for k, v := range *env {
		encoded[k] = utils.EncodeValue(v)
	}
	return &encoded
}

func (dr *DependencyResolver) encodeExecOutputs(stderr, stdout *string) (*string, *string) {
	return utils.EncodeValuePtr(stderr), utils.EncodeValuePtr(stdout)
}

func (dr *DependencyResolver) encodeExecStderr(stderr *string) string {
	if stderr == nil {
		return "    Stderr = \"\"\n"
	}
	return fmt.Sprintf("    Stderr = #\"\"\"\n%s\n\"\"\"#\n", *stderr)
}

func (dr *DependencyResolver) encodeExecStdout(stdout *string) string {
	if stdout == nil {
		return "    Stdout = \"\"\n"
	}
	return fmt.Sprintf("    Stdout = #\"\"\"\n%s\n\"\"\"#\n", *stdout)
}

// Exported for testing
func (dr *DependencyResolver) EncodeExecEnv(env *map[string]string) *map[string]string {
	return dr.encodeExecEnv(env)
}

func (dr *DependencyResolver) EncodeExecOutputs(stderr, stdout *string) (*string, *string) {
	return dr.encodeExecOutputs(stderr, stdout)
}

func (dr *DependencyResolver) EncodeExecStderr(stderr *string) string {
	return dr.encodeExecStderr(stderr)
}

func (dr *DependencyResolver) EncodeExecStdout(stdout *string) string {
	return dr.encodeExecStdout(stdout)
}
