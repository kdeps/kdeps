package evaluator

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apple/pkl-go/pkl"
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

// EvalPkl evaluates the resource file at resourcePath using the Pkl library.
// If the file content is a quoted PKL code string like "new <Dynamic,Listing,Mapping,etc..> {...}", it removes the quotes and writes it to a temporary file for evaluation.
func EvalPkl(fs afero.Fs, ctx context.Context, resourcePath string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	// Read the file content
	content, err := afero.ReadFile(fs, resourcePath)
	if err != nil {
		logger.Error("failed to read file", "resourcePath", resourcePath, "error", err)
		return "", fmt.Errorf("error reading %s: %w", resourcePath, err)
	}
	contentStr := string(content)
	logger.Debug("read file content", "resourcePath", resourcePath, "content", contentStr)

	// Check if file content is a quoted PKL code string of the form "new <Dynamic,Listing,Mapping,etc..> {...}"
	// where the block can span multiple lines
	pklPattern := regexp.MustCompile(`(?s)^(?:\"|\')\s*new\s+(Dynamic|Listing|Mapping|[a-zA-Z][a-zA-Z0-9]*)\s*\{[\s\S]*?\}\s*(?:\"|\')$`)
	dataToEvaluate := contentStr
	evalPath := resourcePath // Default to original file path
	if pklPattern.MatchString(contentStr) {
		// Remove surrounding quotes
		dataToEvaluate = strings.Trim(contentStr, "\"'")
		logger.Debug("detected quoted PKL code pattern, removed quotes", "resourcePath", resourcePath, "original", contentStr, "modified", dataToEvaluate)

		// Create a temporary file for the modified content
		tempFile, err := afero.TempFile(fs, "", "pkl-*.pkl")
		if err != nil {
			logger.Error("failed to create temporary file", "error", err)
			return "", fmt.Errorf("error creating temporary file: %w", err)
		}
		tempPath := tempFile.Name()
		defer func() {
			if err := fs.Remove(tempPath); err != nil {
				logger.Warn("failed to remove temporary file", "tempPath", tempPath, "error", err)
			}
		}()

		// Write the modified (unquoted) content to the temporary file
		if _, err := tempFile.Write([]byte(dataToEvaluate)); err != nil {
			logger.Error("failed to write modified PKL content to temporary file", "tempPath", tempPath, "error", err)
			return "", fmt.Errorf("error writing modified content to %s: %w", tempPath, err)
		}
		if err := tempFile.Close(); err != nil {
			logger.Error("failed to close temporary file", "tempPath", tempPath, "error", err)
			return "", fmt.Errorf("error closing temporary file %s: %w", tempPath, err)
		}
		logger.Debug("successfully wrote modified PKL content to temporary file", "tempPath", tempPath)

		evalPath = tempPath // Use temporary file for evaluation
	} else {
		logger.Debug("no quoted PKL pattern match", "resourcePath", resourcePath, "content", contentStr)
	}

	// Create a ModuleSource using UriSource for paths with a protocol, FileSource for relative paths
	var moduleSource *pkl.ModuleSource
	parsedURL, err := url.Parse(evalPath)
	if err == nil && parsedURL.Scheme != "" {
		// Has a protocol (e.g., file://, http://, https://)
		moduleSource = pkl.UriSource(evalPath)
	} else {
		// Absolute path without a protocol
		moduleSource = pkl.FileSource(evalPath)
	}

	if opts == nil {
		opts = func(options *pkl.EvaluatorOptions) {
			pkl.WithDefaultAllowedResources(options)
			pkl.WithOsEnv(options)
			pkl.WithDefaultAllowedModules(options)
			pkl.WithDefaultCacheDir(options)
			options.Logger = pkl.NoopLogger
			options.AllowedModules = []string{".*"}
			options.AllowedResources = []string{".*"}
			options.OutputFormat = "pcf"
		}
	}

	pklEvaluator, err := pkl.NewEvaluator(ctx, opts)
	if err != nil {
		logger.Error("failed to create Pkl evaluator", "resourcePath", resourcePath, "error", err)
		return "", fmt.Errorf("error creating evaluator for %s: %w", evalPath, err)
	}

	// Evaluate the Pkl file
	result, err := pklEvaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		errMsg := "failed to evaluate Pkl file"
		logger.Error(errMsg, "resourcePath", resourcePath, "evalPath", evalPath, "error", err)
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	// Format the result by prepending the headerSection
	formattedResult := fmt.Sprintf("%s\n%s", headerSection, result)

	// Write the formatted result to the original file
	err = afero.WriteFile(fs, resourcePath, []byte(formattedResult), 0o644)
	if err != nil {
		logger.Error("failed to write formatted result to file", "resourcePath", resourcePath, "error", err)
		return "", fmt.Errorf("error writing formatted result to %s: %w", resourcePath, err)
	}
	logger.Debug("successfully wrote formatted result to file", "resourcePath", resourcePath)

	// Return the formatted result
	return formattedResult, nil
}

func CreateAndProcessPklFile(
	fs afero.Fs,
	ctx context.Context,
	sections []string,
	finalFileName string,
	pklTemplate string,
	opts func(options *pkl.EvaluatorOptions),
	logger *logging.Logger,
	processFunc func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error),
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

	// defer func() {
	//	// Cleanup: Remove the temporary directory
	//	if removeErr := fs.RemoveAll(tmpDir); removeErr != nil {
	//		logger.Warn("failed to clean up temporary directory", "directory", tmpDir, "error", removeErr)
	//	}
	// }()

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
	processedContent, err := processFunc(fs, ctx, tmpFile.Name(), relationshipSection, opts, logger)
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
