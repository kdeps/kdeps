package evaluator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"kdeps/pkg/logging"
	"kdeps/pkg/schema"

	"github.com/alexellis/go-execute/v2"
	"github.com/spf13/afero"
)

// EnsurePklBinaryExists checks if the 'pkl' binary exists in the system PATH.
func EnsurePklBinaryExists() error {
	logging.Info("schema.SchemaVersion:", schema.SchemaVersion)
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

func CreateAndProcessPklFile(
	fs afero.Fs,
	sections []string,
	finalFileName string,
	pklTemplate string,
	importSections []string,
	processFunc func(fs afero.Fs, tmpFile string) (string, error),
) error {

	// Create a temporary directory
	tmpDir, err := afero.TempDir(fs, "", "")
	if err != nil {
		logging.Error("Failed to create temporary directory", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create a unique temporary file in the temporary directory
	tmpFile, err := afero.TempFile(fs, tmpDir, "*.pkl") // This will create a unique temporary file
	if err != nil {
		logging.Error("Failed to create temporary file", "dir", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Format the import sections as `import "file.pkl"`
	var formattedImports []string
	for _, importFile := range importSections {
		formattedImports = append(formattedImports, fmt.Sprintf(`import "%s"`, importFile))
	}

	// Prepare the sections with the "amends" and imports
	// amends "package://schema.kdeps.com/core@0.0.34#/Kdeps.pkl"
	amendsSection := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/%s"`, schema.SchemaVersion, pklTemplate)
	fullSections := append([]string{amendsSection}, formattedImports...)
	fullSections = append(fullSections, sections...)

	// Write sections to the temporary file
	_, err = tmpFile.Write([]byte(strings.Join(fullSections, "\n")))
	if err != nil {
		logging.Error("Failed to write to temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Process the temporary file using the provided function
	processedContent, err := processFunc(fs, tmpFile.Name())
	if err != nil {
		logging.Error("Failed to process temporary file", "path", tmpFile.Name(), "error", err)
		return fmt.Errorf("failed to process temporary file: %w", err)
	}

	// Write the processed content to the final file
	err = afero.WriteFile(fs, finalFileName, []byte(processedContent), 0644)
	if err != nil {
		logging.Error("Failed to write final file", "path", finalFileName, "error", err)
		return fmt.Errorf("failed to write final file: %w", err)
	}

	return nil
}
