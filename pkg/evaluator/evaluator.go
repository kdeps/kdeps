package evaluator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/alexellis/go-execute/v2"
	"github.com/spf13/afero"
)

const schemaVersionFilePath = "../../SCHEMA_VERSION"

// EnsurePklBinaryExists checks if the 'pkl' binary exists in the system PATH.
func EnsurePklBinaryExists() error {
	binaryName := "pkl"
	if _, err := exec.LookPath(binaryName); err != nil {
		return fmt.Errorf("the binary '%s' is not found in PATH: %w", binaryName, err)
	}
	return nil
}

// EvalPkl evaluates the resource file at resourcePath using the 'pkl' binary.
// It expects the resourcePath to have a .pkl extension.
func EvalPkl(fs afero.Fs, resourcePath string) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		return "", fmt.Errorf("file '%s' must have a .pkl extension", resourcePath)
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
		return "", fmt.Errorf("command execution failed: %w", err)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		return "", fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	return result.Stdout, nil
}
