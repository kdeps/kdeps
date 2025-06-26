package evaluator

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

// Injectable function declarations for better testability
var (
	// ExecLookPathFunc allows injection of executable lookup
	ExecLookPathFunc = func(file string) (string, error) {
		return exec.LookPath(file)
	}

	// OsExitFunc allows injection of os.Exit for testing
	OsExitFunc = func(code int) {
		os.Exit(code)
	}

	// NewPklEvaluatorFunc allows injection of PKL evaluator creation
	NewPklEvaluatorFunc = func(ctx context.Context, opts func(options *pkl.EvaluatorOptions)) (pkl.Evaluator, error) {
		return pkl.NewEvaluator(ctx, opts)
	}

	// AferoTempFileFunc allows injection of temporary file creation
	AferoTempFileFunc = func(fs afero.Fs, dir, pattern string) (afero.File, error) {
		return afero.TempFile(fs, dir, pattern)
	}

	// AferoTempDirFunc allows injection of temporary directory creation
	AferoTempDirFunc = func(fs afero.Fs, dir, prefix string) (string, error) {
		return afero.TempDir(fs, dir, prefix)
	}

	// CreateKdepsTempDirFunc allows injection of organized kdeps temporary directory creation
	CreateKdepsTempDirFunc = func(fs afero.Fs, requestID string, suffix string) (string, error) {
		return utils.CreateKdepsTempDir(fs, requestID, suffix)
	}

	// CreateKdepsTempFileFunc allows injection of organized kdeps temporary file creation
	CreateKdepsTempFileFunc = func(fs afero.Fs, requestID string, pattern string) (afero.File, error) {
		return utils.CreateKdepsTempFile(fs, requestID, pattern)
	}

	// AferoReadFileFunc allows injection of file reading
	AferoReadFileFunc = func(fs afero.Fs, filename string) ([]byte, error) {
		return afero.ReadFile(fs, filename)
	}

	// AferoWriteFileFunc allows injection of file writing
	AferoWriteFileFunc = func(fs afero.Fs, filename string, data []byte, perm os.FileMode) error {
		return afero.WriteFile(fs, filename, data, perm)
	}
)

// EnsurePklBinaryExists checks if the 'pkl' binary exists in the system PATH.
func EnsurePklBinaryExists(ctx context.Context, logger *logging.Logger) error {
	binaryNames := []string{"pkl", "pkl.exe"} // Support both Unix-like and Windows binary names
	for _, binaryName := range binaryNames {
		if _, err := ExecLookPathFunc(binaryName); err == nil {
			return nil // Found a valid binary, no error
		}
	}
	// Log the error if none of the binaries were found
	logger.Fatal("apple PKL not found in PATH. Please install Apple PKL (see https://pkl-lang.org/main/current/pkl-cli/index.html#installation) for more details")
	OsExitFunc(1)
	return nil // Unreachable, but included for clarity
}

// EvalPkl evaluates the resource file at resourcePath using the Pkl library.
func EvalPkl(fs afero.Fs, ctx context.Context, resourcePath string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	// Use the resource path directly for evaluation
	evalPath := resourcePath
	logger.Debug("evaluating PKL file", "resourcePath", resourcePath)

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

	pklEvaluator, err := NewPklEvaluatorFunc(ctx, opts)
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

	// Only write back to file for local files, not package URIs
	if !strings.HasPrefix(resourcePath, "package://") {
		// Write the formatted result to the original file
		err = AferoWriteFileFunc(fs, resourcePath, []byte(formattedResult), 0o644)
		if err != nil {
			logger.Error("failed to write formatted result to file", "resourcePath", resourcePath, "error", err)
			return "", fmt.Errorf("error writing formatted result to %s: %w", resourcePath, err)
		}
		logger.Debug("successfully wrote formatted result to file", "resourcePath", resourcePath)
	} else {
		logger.Debug("skipping file write for package URI", "resourcePath", resourcePath)
	}

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
	isExtension bool, // Parameter to control amends vs extends
	requestID string, // Parameter for organized temporary file creation
) error {
	// Create a temporary directory using organized structure
	var tmpDir string
	var err error

	if requestID != "" {
		// Use organized temp directory structure for kdeps
		tmpDir, err = CreateKdepsTempDirFunc(fs, requestID, "pkl-eval")
	} else {
		// Fallback to regular temp directory
		tmpDir, err = AferoTempDirFunc(fs, "", "")
	}

	if err != nil {
		logger.Error("failed to create temporary directory", "path", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create a unique temporary file in the temporary directory
	var tmpFile afero.File
	if requestID != "" {
		// Use organized temp file creation
		tmpFile, err = CreateKdepsTempFileFunc(fs, requestID, "pkl-eval-*.pkl")
	} else {
		// Fallback to regular temp file creation
		tmpFile, err = AferoTempFileFunc(fs, tmpDir, "*.pkl")
	}

	if err != nil {
		logger.Error("failed to create temporary file", "dir", tmpDir, "error", err)
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tmpFile.Close()

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
	err = AferoWriteFileFunc(fs, finalFileName, []byte(processedContent), 0o644)
	if err != nil {
		logger.Error("failed to write final file", "path", finalFileName, "error", err)
		return fmt.Errorf("failed to write final file: %w", err)
	}

	return nil
}
