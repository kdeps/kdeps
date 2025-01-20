package evaluator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"

	"github.com/alexellis/go-execute/v2"
	"github.com/spf13/afero"
)

// EnsurePklBinaryExists checks if the 'pkl' binary exists in the system PATH.
func EnsurePklBinaryExists(ctx context.Context, logger *logging.Logger) error {
	binaryNames := []string{"pkl", "pkl.exe"} // Support both Unix-like and Windows binary names
	for _, binaryName := range binaryNames {
		if _, err := exec.LookPath(binaryName); err == nil {
			return nil // Found a valid binary, no error
		}
	}
	// Log the error if none of the binaries were found
	logger.Fatal("Apple PKL not found in PATH. Please install Apple PKL (see https://pkl-lang.org/main/current/pkl-cli/index.html#installation) for more details")
	os.Exit(1)
	return nil // Unreachable, but included for clarity
}

// EvalPkl evaluates the resource file at resourcePath using the 'pkl' binary.
// It expects the resourcePath to have a .pkl extension.
func EvalPkl(fs afero.Fs, ctx context.Context, resourcePath string, headerSection string, logger *logging.Logger) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Ensure that the 'pkl' binary is available
	if err := EnsurePklBinaryExists(ctx, logger); err != nil {
		return "", err
	}

	// Prepare the command to evaluate the .pkl file
	cmd := execute.ExecTask{
		Command:     "pkl",
		Args:        []string{"eval", resourcePath},
		StreamStdio: false,
	}

	// Execute the command
	result, err := cmd.Execute(ctx)
	if err != nil {
		errMsg := "command execution failed"
		logger.Error(errMsg, "error", err)
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	// Check for non-zero exit code
	if result.ExitCode != 0 {
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
		logger.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Format the result by prepending the headerSection to the command stdout
	formattedResult := fmt.Sprintf("%s\n%s", headerSection, result.Stdout)

	// Return the formatted result
	return formattedResult, nil
}

func CreateAndProcessPklFile(
	fs afero.Fs,
	ctx context.Context,
	sections []string,
	finalFileName string,
	pklTemplate string,
	logger *logging.Logger,
	processFunc func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, logger *logging.Logger) (string, error),
	isExtension bool, // New parameter to control amends vs extends
) error {
	// Create a temporary directory
	tmpDir, err := afero.TempDir(fs, "", "")
	if err != nil {
		logger.Error("Failed to create temporary directory", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create a unique temporary file in the temporary directory
	tmpFile, err := afero.TempFile(fs, tmpDir, "*.pkl") // This will create a unique temporary file
	if err != nil {
		logger.Error("Failed to create temporary file", "dir", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Choose "amends" or "extends" based on isExtension
	relationship := "amends"
	if isExtension {
		relationship = "extends"
	}

	// Prepare the sections with the relationship keyword and imports
	// amends or extends "package://schema.kdeps.com/core@0.0.34#/Kdeps.pkl"
	relationshipSection := fmt.Sprintf(`%s "package://schema.kdeps.com/core@%s#/%s"`, relationship, schema.SchemaVersion(ctx), pklTemplate)
	fullSections := append([]string{relationshipSection}, sections...)

	// Write sections to the temporary file
	_, err = tmpFile.Write([]byte(strings.Join(fullSections, "\n")))
	if err != nil {
		logger.Error("Failed to write to temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Process the temporary file using the provided function
	processedContent, err := processFunc(fs, ctx, tmpFile.Name(), relationshipSection, logger)
	if err != nil {
		logger.Error("Failed to process temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to process temporary file: %w", err)
	}

	// Write the processed content to the final file
	err = afero.WriteFile(fs, finalFileName, []byte(processedContent), 0o644)
	if err != nil {
		logger.Error("Failed to write final file", "path", finalFileName, "error", err)
		return fmt.Errorf("failed to write final file: %w", err)
	}

	return nil
}
