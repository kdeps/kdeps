package evaluator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"kdeps/pkg/logging"

	"github.com/alexellis/go-execute/v2"
	"github.com/spf13/afero"
)

const schemaVersionFilePath = "../../SCHEMA_VERSION"

// EnsurePklBinaryExists checks if the 'pkl' binary exists in the system PATH.
func EnsurePklBinaryExists() error {
	binaryName := "pkl"
	if _, err := exec.LookPath(binaryName); err != nil {
		errMsg := fmt.Sprintf("the binary '%s' is not found in PATH", binaryName)
		logging.Error(errMsg, "error", err)
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

// EvalPkl evaluates the resource file at resourcePath using the 'pkl' binary.
// It expects the resourcePath to have a .pkl extension.
func EvalPkl(fs afero.Fs, resourcePath string) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logging.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Ensure that the 'pkl' binary is available
	if err := EnsurePklBinaryExists(); err != nil {
		return "", err
	}

	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", resourcePath},
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(context.Background())
	if err != nil {
		errMsg := "command execution failed"
		logging.Error(errMsg, "error", err)
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		logging.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	return result.Stdout, nil
}
