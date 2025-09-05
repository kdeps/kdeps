package evaluator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apple/pkl-go/pkl"
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

// NewConfiguredEvaluator creates a Pkl evaluator with preconfigured defaults and optional readers.
// Use outputFormat to override the default renderer (e.g., "pcf", "json").
func NewConfiguredEvaluator(ctx context.Context, outputFormat string, readers []pkl.ResourceReader) (pkl.Evaluator, error) {
	opts := func(options *pkl.EvaluatorOptions) {
		pkl.WithDefaultAllowedResources(options)
		pkl.WithOsEnv(options)
		pkl.WithDefaultAllowedModules(options)
		pkl.WithDefaultCacheDir(options)
		options.Logger = pkl.NoopLogger
		if outputFormat != "" {
			options.OutputFormat = outputFormat
		}
		// Allow all; actual access is governed by registered readers and sources used
		options.AllowedModules = []string{".*"}
		options.AllowedResources = []string{".*"}
		if len(readers) > 0 {
			options.ResourceReaders = readers
		}
	}
	return pkl.NewEvaluator(ctx, opts)
}

// fallbackToCLI executes PKL evaluation using CLI when SDK fails.
func fallbackToCLI(ctx context.Context, evalPath string, headerSection string, fs afero.Fs, resourcePath string, logger *logging.Logger) (string, error) {
	stdout, stderr, exitCode, execErr := kdepsexec.KdepsExec(ctx, "pkl", []string{"eval", evalPath}, "", false, false, logger)
	if execErr != nil {
		logger.Error("CLI command execution failed", "stderr", stderr, "error", execErr)
		return "", fmt.Errorf("cli error: %w", execErr)
	}
	if exitCode != 0 {
		errMsg := fmt.Sprintf("cli command failed with exit code %d: %s", exitCode, stderr)
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}
	formattedResult := fmt.Sprintf("%s\n%s", headerSection, stdout)

	if u, err := url.Parse(resourcePath); err != nil || u.Scheme == "" || u.Scheme == "file" {
		if err := afero.WriteFile(fs, resourcePath, []byte(formattedResult), 0o644); err != nil {
			logger.Error("failed to write formatted result to file", "resourcePath", resourcePath, "error", err)
			return "", fmt.Errorf("error writing formatted result to %s: %w", resourcePath, err)
		}
	}
	return formattedResult, nil
}

// EvalPkl evaluates the resource at resourcePath using the Pkl SDK.
// If the file content is a quoted PKL code string like "new <Dynamic,Listing,Mapping,etc..> {...}",
// it removes the quotes and writes it to a temporary file for evaluation.
func EvalPkl(
	fs afero.Fs,
	ctx context.Context,
	resourcePath string,
	headerSection string,
	opts func(options *pkl.EvaluatorOptions),
	logger *logging.Logger,
) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return "", errors.New(errMsg)
	}

	// Read the file content or URI
	content, err := afero.ReadFile(fs, resourcePath)
	if err != nil {
		// Not all resourcePaths are filesystem paths (could be package:// URI).
		// Only log debug here; continue with SDK source resolution below.
		logger.Debug("failed to read file content (may be a URI)", "resourcePath", resourcePath, "error", err)
	}
	contentStr := string(content)

	// Detect quoted PKL code pattern: "new <Type> { ... }" or 'new <Type> { ... }'
	pklPattern := regexp.MustCompile(`(?s)^(?:\"|\')\s*new\s+(Dynamic|Listing|Mapping|[a-zA-Z][a-zA-Z0-9]*)\s*\{[\s\S]*?\}\s*(?:\"|\')$`)
	evalPath := resourcePath
	if contentStr != "" && pklPattern.MatchString(contentStr) {
		// Remove surrounding quotes
		dataToEvaluate := strings.Trim(contentStr, "\"'")
		// Write modified content to a temp file for evaluation
		tempFile, err := afero.TempFile(fs, "", "pkl-*.pkl")
		if err != nil {
			logger.Error("failed to create temporary file", "error", err)
			return "", fmt.Errorf("error creating temporary file: %w", err)
		}
		tempPath := tempFile.Name()
		if _, err := tempFile.Write([]byte(dataToEvaluate)); err != nil {
			_ = tempFile.Close()
			_ = fs.Remove(tempPath)
			logger.Error("failed to write modified PKL content to temporary file", "tempPath", tempPath, "error", err)
			return "", fmt.Errorf("error writing modified content to %s: %w", tempPath, err)
		}
		if err := tempFile.Close(); err != nil {
			_ = fs.Remove(tempPath)
			logger.Error("failed to close temporary file", "tempPath", tempPath, "error", err)
			return "", fmt.Errorf("error closing temporary file %s: %w", tempPath, err)
		}
		defer func() { _ = fs.Remove(tempPath) }()
		evalPath = tempPath
	}

	// Build module source: UriSource if has scheme, else FileSource
	var moduleSource *pkl.ModuleSource
	if parsed, err := url.Parse(evalPath); err == nil && parsed.Scheme != "" {
		moduleSource = pkl.UriSource(evalPath)
	} else {
		moduleSource = pkl.FileSource(evalPath)
	}

	var evaluator pkl.Evaluator
	if opts == nil {
		var eErr error
		evaluator, eErr = NewConfiguredEvaluator(ctx, "pcf", nil)
		if eErr != nil {
			err = eErr
		}
	} else {
		var eErr error
		evaluator, eErr = pkl.NewEvaluator(ctx, opts)
		if eErr != nil {
			err = eErr
		}
	}
	if err != nil {
		// Fallback to CLI if SDK cannot initialize (e.g., version detection issue)
		logger.Warn("SDK evaluator initialization failed; falling back to CLI eval", "error", err)
		result, cliErr := fallbackToCLI(ctx, evalPath, headerSection, fs, resourcePath, logger)
		if cliErr != nil {
			return "", fmt.Errorf("sdk init error: %w; cli error: %w", err, cliErr)
		}
		return result, nil
	}

	// Evaluate text output
	result, err := evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		// Fallback to CLI on evaluation error
		logger.Warn("SDK evaluation failed; falling back to CLI eval", "error", err)
		result, cliErr := fallbackToCLI(ctx, evalPath, headerSection, fs, resourcePath, logger)
		if cliErr != nil {
			return "", fmt.Errorf("sdk eval error: %w; cli error: %w", err, cliErr)
		}
		return result, nil
	}

	formattedResult := fmt.Sprintf("%s\n%s", headerSection, result)

	// Write the formatted result back to the original path if it is a filesystem path
	if u, err := url.Parse(resourcePath); err != nil || u.Scheme == "" || u.Scheme == "file" {
		if err := afero.WriteFile(fs, resourcePath, []byte(formattedResult), 0o644); err != nil {
			logger.Error("failed to write formatted result to file", "resourcePath", resourcePath, "error", err)
			return "", fmt.Errorf("error writing formatted result to %s: %w", resourcePath, err)
		}
	}

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

// ValidatePkl validates a PKL file for syntax correctness without modifying the file.
// This function only checks if the PKL file can be evaluated but doesn't write the result back.
func ValidatePkl(
	fs afero.Fs,
	ctx context.Context,
	resourcePath string,
	logger *logging.Logger,
) error {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// Read the file content or URI
	content, err := afero.ReadFile(fs, resourcePath)
	if err != nil {
		// Not all resourcePaths are filesystem paths (could be package:// URI).
		// Only log debug here; continue with SDK source resolution below.
		logger.Debug("failed to read file content (may be a URI)", "resourcePath", resourcePath, "error", err)
	}
	contentStr := string(content)

	// Detect quoted PKL code pattern: "new <Type> { ... }" or 'new <Type> { ... }'
	pklPattern := regexp.MustCompile(`(?s)^(?:\"|\')\s*new\s+(Dynamic|Listing|Mapping|[a-zA-Z][a-zA-Z0-9]*)\s*\{[\s\S]*?\}\s*(?:\"|\')$`)
	evalPath := resourcePath
	var tempPath string

	if contentStr != "" && pklPattern.MatchString(contentStr) {
		// Remove surrounding quotes
		dataToEvaluate := strings.Trim(contentStr, "\"'")
		// Write modified content to a temp file for evaluation
		tempFile, err := afero.TempFile(fs, "", "pkl-*.pkl")
		if err != nil {
			logger.Error("failed to create temporary file", "error", err)
			return fmt.Errorf("error creating temporary file: %w", err)
		}
		tempPath = tempFile.Name()
		if _, err := tempFile.Write([]byte(dataToEvaluate)); err != nil {
			_ = tempFile.Close()
			_ = fs.Remove(tempPath)
			logger.Error("failed to write modified PKL content to temporary file", "tempPath", tempPath, "error", err)
			return fmt.Errorf("error writing modified content to %s: %w", tempPath, err)
		}
		if err := tempFile.Close(); err != nil {
			_ = fs.Remove(tempPath)
			logger.Error("failed to close temporary file", "tempPath", tempPath, "error", err)
			return fmt.Errorf("error closing temporary file %s: %w", tempPath, err)
		}
		defer func() { _ = fs.Remove(tempPath) }()
		evalPath = tempPath
	}

	// Build module source: UriSource if has scheme, else FileSource
	var moduleSource *pkl.ModuleSource
	if parsed, err := url.Parse(evalPath); err == nil && parsed.Scheme != "" {
		moduleSource = pkl.UriSource(evalPath)
	} else {
		moduleSource = pkl.FileSource(evalPath)
	}

	// Create evaluator for validation
	evaluator, err := NewConfiguredEvaluator(ctx, "pcf", nil)
	if err != nil {
		// Fallback to CLI if SDK cannot initialize (e.g., version detection issue)
		logger.Warn("SDK evaluator initialization failed; falling back to CLI validation", "error", err)
		_, stderr, exitCode, execErr := kdepsexec.KdepsExec(ctx, "pkl", []string{"eval", evalPath}, "", false, false, logger)
		if execErr != nil {
			logger.Error("CLI command execution failed", "stderr", stderr, "error", execErr)
			return fmt.Errorf("sdk init error: %w; cli error: %w", err, execErr)
		}
		if exitCode != 0 {
			errMsg := fmt.Sprintf("cli validation failed with exit code %d: %s", exitCode, stderr)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		return nil
	}

	// Evaluate for validation only (don't capture output)
	_, err = evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		// Fallback to CLI on evaluation error
		logger.Warn("SDK validation failed; falling back to CLI validation", "error", err)
		_, stderr, exitCode, execErr := kdepsexec.KdepsExec(ctx, "pkl", []string{"eval", evalPath}, "", false, false, logger)
		if execErr != nil {
			logger.Error("CLI command execution failed", "stderr", stderr, "error", execErr)
			return fmt.Errorf("sdk validation error: %w; cli error: %w", err, execErr)
		}
		if exitCode != 0 {
			errMsg := fmt.Sprintf("cli validation failed with exit code %d: %s", exitCode, stderr)
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		return nil
	}

	return nil
}
