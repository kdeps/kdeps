package evaluator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
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
	logger.Fatal("apple PKL not found in PATH. Please install Apple PKL (see https://pkl-lang.org/main/current/pkl-cli/index.html#installation) for more details")
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
		return "", errors.New(errMsg)
	}

	// Ensure that the 'pkl' binary is available
	if err := EnsurePklBinaryExists(ctx, logger); err != nil {
		return "", err
	}

	stdout, stderr, exitCode, err := kdepsexec.KdepsExec(ctx, "pkl", []string{"eval", resourcePath}, "", false, false, logger)
	if err != nil {
		logger.Error("command execution failed", "stderr", stderr, "error", err)
		return "", err
	}

	if exitCode != 0 {
		errMsg := fmt.Sprintf("command failed with exit code %d: %s", exitCode, stderr)
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}

	formattedResult := fmt.Sprintf("%s\n%s", headerSection, stdout)

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
		logger.Error("failed to create temporary directory", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create a unique temporary file in the temporary directory
	tmpFile, err := afero.TempFile(fs, tmpDir, "*.pkl") // This will create a unique temporary file
	if err != nil {
		logger.Error("failed to create temporary file", "dir", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	defer func() {
		// Cleanup: Remove the temporary directory
		if removeErr := fs.RemoveAll(tmpDir); removeErr != nil {
			logger.Warn("failed to clean up temporary directory", "directory", tmpDir, "error", removeErr)
		}
	}()

	// Choose "amends" or "extends" based on isExtension
	relationship := "amends"
	if isExtension {
		relationship = "extends"
	}

	// Prepare the sections with the relationship keyword and imports
	relationshipSection := fmt.Sprintf(`%s "package://schema.kdeps.com/core@%s#/%s"`, relationship, schema.SchemaVersion(ctx), pklTemplate)
	fullSections := append([]string{relationshipSection}, sections...)

	// Write sections to the temporary file
	_, err = tmpFile.Write([]byte(strings.Join(fullSections, "\n")))
	if err != nil {
		logger.Error("failed to write to temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Process the temporary file using the provided function
	processedContent, err := processFunc(fs, ctx, tmpFile.Name(), relationshipSection, logger)
	if err != nil {
		logger.Error("failed to process temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to process temporary file: %w", err)
	}

	// Write the processed content to the final file
	err = afero.WriteFile(fs, finalFileName, []byte(processedContent), 0o644)
	if err != nil {
		logger.Error("failed to write final file", "path", finalFileName, "error", err)
		return fmt.Errorf("failed to write final file: %w", err)
	}

	return nil
}
