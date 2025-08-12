package resolver

import (
	"context"
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
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) HandlePython(actionID string, pythonBlock *pklPython.ResourcePython) error {
	// Synchronously decode the python block.
	if err := dr.decodePythonBlock(pythonBlock); err != nil {
		dr.Logger.Error("failed to decode python block", "actionID", actionID, "error", err)
		return err
	}

	// Process the python block asynchronously in a goroutine.
	go func(aID string, block *pklPython.ResourcePython) {
		if err := dr.processPythonBlock(aID, block); err != nil {
			// Log the error; additional error handling can be added here if needed.
			dr.Logger.Error("failed to process python block", "actionID", aID, "error", err)
		}
	}(actionID, pythonBlock)

	// Return immediately while the python block is processed in the background.
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
	if dr.AnacondaInstalled && pythonBlock.CondaEnvironment != nil && *pythonBlock.CondaEnvironment != "" {
		if err := dr.activateCondaEnvironment(*pythonBlock.CondaEnvironment); err != nil {
			return err
		}

		defer dr.deactivateCondaEnvironment()
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
	var execErr error
	if dr.ExecTaskRunnerFn != nil {
		execStdout, execStderr, execErr = dr.ExecTaskRunnerFn(dr.Context, cmd)
	} else {
		execStdout, execStderr, _, execErr = kdepsexec.RunExecTask(dr.Context, cmd, dr.Logger, false)
	}
	if execErr != nil {
		return fmt.Errorf("execution failed: %w", execErr)
	}

	pythonBlock.Stdout = &execStdout
	pythonBlock.Stderr = &execStderr

	ts := pkl.Duration{
		Value: float64(time.Now().Unix()),
		Unit:  pkl.Nanosecond,
	}
	pythonBlock.Timestamp = &ts

	return dr.AppendPythonEntry(actionID, pythonBlock)
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

func (dr *DependencyResolver) AppendPythonEntry(resourceID string, newPython *pklPython.ResourcePython) error {
	pklPath := filepath.Join(dr.ActionDir, "python/"+dr.RequestID+"__python_output.pkl")

	res, err := dr.LoadResource(dr.Context, pklPath, PythonResource)
	if err != nil {
		return fmt.Errorf("failed to load PKL: %w", err)
	}

	pklRes, ok := res.(*pklPython.PythonImpl)
	if !ok {
		return errors.New("failed to cast pklRes to *pklPython.Resource")
	}

	resources := pklRes.GetResources()
	if resources == nil {
		emptyMap := make(map[string]*pklPython.ResourcePython)
		resources = &emptyMap
	}
	existingResources := *resources

	var filePath string
	if newPython.Stdout != nil {
		filePath, err = dr.WritePythonStdoutToFile(resourceID, newPython.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write stdout to file: %w", err)
		}
		newPython.File = &filePath
	}

	encodedScript := utils.EncodeValue(newPython.Script)
	encodedEnv := dr.encodePythonEnv(newPython.Env)
	encodedStderr, encodedStdout := dr.encodePythonOutputs(newPython.Stderr, newPython.Stdout)

	timeoutDuration := newPython.TimeoutDuration
	if timeoutDuration == nil {
		sec := dr.DefaultTimeoutSec
		if sec <= 0 {
			sec = 60
		}
		timeoutDuration = &pkl.Duration{Value: float64(sec), Unit: pkl.Second}
	}

	timestamp := &pkl.Duration{
		Value: float64(time.Now().Unix()),
		Unit:  pkl.Nanosecond,
	}

	existingResources[resourceID] = &pklPython.ResourcePython{
		Env:             encodedEnv,
		Script:          encodedScript,
		Stderr:          encodedStderr,
		Stdout:          encodedStdout,
		File:            &filePath,
		Timestamp:       timestamp,
		TimeoutDuration: timeoutDuration,
	}

	var pklContent strings.Builder
	pklContent.WriteString(fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Python.pkl\"\n\n", schema.SchemaVersion(dr.Context)))
	pklContent.WriteString("Resources {\n")

	for id, res := range existingResources {
		pklContent.WriteString(fmt.Sprintf("  [\"%s\"] {\n", id))
		pklContent.WriteString(fmt.Sprintf("    Script = \"%s\"\n", res.Script))

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

		pklContent.WriteString(dr.encodePythonStderr(res.Stderr))
		pklContent.WriteString(dr.encodePythonStdout(res.Stdout))
		pklContent.WriteString(fmt.Sprintf("    File = \"%s\"\n", *res.File))

		pklContent.WriteString("  }\n")
	}
	pklContent.WriteString("}\n")

	if err := afero.WriteFile(dr.Fs, pklPath, []byte(pklContent.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write PKL file: %w", err)
	}

	evaluatedContent, err := evaluator.EvalPkl(
		dr.Fs,
		dr.Context,
		pklPath,
		fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Python.pkl\"", schema.SchemaVersion(dr.Context)),
		nil,
		dr.Logger,
	)
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL: %w", err)
	}

	return afero.WriteFile(dr.Fs, pklPath, []byte(evaluatedContent), 0o644)
}

func (dr *DependencyResolver) encodePythonEnv(env *map[string]string) *map[string]string {
	if env == nil {
		return nil
	}
	encoded := make(map[string]string)
	for k, v := range *env {
		encoded[k] = utils.EncodeValue(v)
	}
	return &encoded
}

func (dr *DependencyResolver) encodePythonOutputs(stderr, stdout *string) (*string, *string) {
	return utils.EncodeValuePtr(stderr), utils.EncodeValuePtr(stdout)
}

func (dr *DependencyResolver) encodePythonStderr(stderr *string) string {
	if stderr == nil {
		return "    Stderr = \"\"\n"
	}
	return fmt.Sprintf("    Stderr = #\"\"\"\n%s\n\"\"\"#\n", *stderr)
}

func (dr *DependencyResolver) encodePythonStdout(stdout *string) string {
	if stdout == nil {
		return "    Stdout = \"\"\n"
	}
	return fmt.Sprintf("    Stdout = #\"\"\"\n%s\n\"\"\"#\n", *stdout)
}
