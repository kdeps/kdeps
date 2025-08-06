package resolver

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexellis/go-execute/v2"
	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandleExec(actionID string, execBlock *pklExec.ResourceExec) error {
	// Decode the exec block synchronously
	if err := dr.decodeExecBlock(execBlock); err != nil {
		dr.Logger.Error("failed to decode exec block", "actionID", actionID, "error", err)
		return err
	}

	// Run processExecBlock asynchronously in a goroutine
	go func(aID string, block *pklExec.ResourceExec) {
		if err := dr.processExecBlock(aID, block); err != nil {
			// Log the error; consider additional error handling as needed.
			dr.Logger.Error("failed to process exec block", "actionID", aID, "error", err)
		}
	}(actionID, execBlock)

	// Return immediately; the exec block is being processed in the background.
	return nil
}

func (dr *DependencyResolver) decodeExecBlock(execBlock *pklExec.ResourceExec) error {
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
	var err error
	if dr.ExecTaskRunnerFn != nil {
		stdout, stderr, err = dr.ExecTaskRunnerFn(dr.Context, task)
	} else {
		// fallback direct execution via kdepsexec
		stdout, stderr, _, err = kdepsexec.RunExecTask(dr.Context, task, dr.Logger, false)
	}
	if err != nil {
		return err
	}

	execBlock.Stdout = &stdout
	execBlock.Stderr = &stderr

	ts := pkl.Duration{
		Value: float64(time.Now().Unix()),
		Unit:  pkl.Nanosecond,
	}
	execBlock.Timestamp = &ts

	return dr.AppendExecEntry(actionID, execBlock)
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

func (dr *DependencyResolver) AppendExecEntry(resourceID string, newExec *pklExec.ResourceExec) error {
	pklPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")

	res, err := dr.LoadResource(dr.Context, pklPath, ExecResource)
	if err != nil {
		return fmt.Errorf("failed to load PKL: %w", err)
	}

	pklRes, ok := res.(*pklExec.ExecImpl)
	if !ok {
		return errors.New("failed to cast pklRes to *pklExec.ExecImpl")
	}

	resources := pklRes.GetResources()
	if resources == nil {
		emptyMap := make(map[string]*pklExec.ResourceExec)
		resources = &emptyMap
	}
	existingResources := *resources

	// Prepare file path and write stdout to file
	var filePath string
	if newExec.Stdout != nil {
		filePath, err = dr.WriteStdoutToFile(resourceID, newExec.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write stdout to file: %w", err)
		}
		newExec.File = &filePath
	}

	// Encode fields for PKL storage
	encodedCommand := utils.EncodeValue(newExec.Command)
	encodedEnv := dr.encodeExecEnv(newExec.Env)
	encodedStderr, encodedStdout := dr.encodeExecOutputs(newExec.Stderr, newExec.Stdout)

	timestamp := newExec.Timestamp
	if timestamp == nil {
		timestamp = &pkl.Duration{
			Value: float64(time.Now().Unix()),
			Unit:  pkl.Nanosecond,
		}
	}

	existingResources[resourceID] = &pklExec.ResourceExec{
		Env:             encodedEnv,
		Command:         encodedCommand,
		Stderr:          encodedStderr,
		Stdout:          encodedStdout,
		File:            &filePath,
		Timestamp:       timestamp,
		TimeoutDuration: newExec.TimeoutDuration,
	}

	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("Resources {\n")

	for id, res := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    Command = \"%s\"\n", res.Command))

		if res.TimeoutDuration != nil {
			pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %g.%s\n", res.TimeoutDuration.Value, res.TimeoutDuration.Unit.String()))
		} else {
			pklContent.WriteString(fmt.Sprintf("    TimeoutDuration = %d.s\n", dr.DefaultTimeoutSec))
		}

		if res.Timestamp != nil {
			pklContent.WriteString(fmt.Sprintf("    Timestamp = %g.%s\n", res.Timestamp.Value, res.Timestamp.Unit.String()))
		}

		pklContent.WriteString("    Env ")
		pklContent.WriteString(utils.EncodePklMap(res.Env))

		pklContent.WriteString(dr.encodeExecStderr(res.Stderr))
		pklContent.WriteString(dr.encodeExecStdout(res.Stdout))
		if res.File != nil {
			pklContent.WriteString(fmt.Sprintf("    File = \"%s\"\n", *res.File))
		} else {
			pklContent.WriteString("    File = \"\"\n")
		}

		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL file: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(dr.Fs, dr.Context, pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"", schema.SchemaVersion(dr.Context)), dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

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
