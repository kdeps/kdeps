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
	if err := validatePklFile(resourcePath, logger); err != nil {
		return "", err
	}

	evalPath, err := preparePklContent(fs, resourcePath, logger)
	if err != nil {
		return "", err
	}

	moduleSource := createModuleSource(evalPath)
	result, err := evaluateWithSdk(ctx, moduleSource, headerSection, evalPath, fs, resourcePath, opts, logger)
	if err != nil {
		return "", err
	}

	return writeResultToFile(fs, resourcePath, headerSection, result, logger)
}

func validatePklFile(resourcePath string, logger *logging.Logger) error {
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func preparePklContent(fs afero.Fs, resourcePath string, logger *logging.Logger) (string, error) {
	evalPath, err := handleQuotedPklContent(fs, resourcePath, logger)
	if err != nil {
		return "", err
	}

	return evalPath, nil
}

func handleQuotedPklContent(fs afero.Fs, resourcePath string, logger *logging.Logger) (string, error) {
	content, err := afero.ReadFile(fs, resourcePath)
	if err != nil {
		logger.Debug("failed to read file content (may be a URI)", "resourcePath", resourcePath, "error", err)
		return resourcePath, nil
	}

	contentStr := string(content)
	if !isQuotedPklContent(contentStr) {
		return resourcePath, nil
	}

	dataToEvaluate := strings.Trim(contentStr, "\"'")
	return createTempPklFile(fs, dataToEvaluate, logger)
}

func isQuotedPklContent(contentStr string) bool {
	if contentStr == "" {
		return false
	}
	pklPattern := regexp.MustCompile(`(?s)^(?:\"|\')\s*new\s+(Dynamic|Listing|Mapping|[a-zA-Z][a-zA-Z0-9]*)\s*\{[\s\S]*?\}\s*(?:\"|\')$`)
	return pklPattern.MatchString(contentStr)
}

func createTempPklFile(fs afero.Fs, dataToEvaluate string, logger *logging.Logger) (string, error) {
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

	return tempPath, nil
}

func createModuleSource(evalPath string) *pkl.ModuleSource {
	if parsed, err := url.Parse(evalPath); err == nil && parsed.Scheme != "" {
		return pkl.UriSource(evalPath)
	}
	return pkl.FileSource(evalPath)
}

func evaluateWithSdk(ctx context.Context, moduleSource *pkl.ModuleSource, headerSection, evalPath string, fs afero.Fs, resourcePath string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
	evaluator, err := createEvaluator(ctx, opts, logger)
	if err != nil {
		logger.Warn("SDK evaluator initialization failed; falling back to CLI eval", "error", err)
		return fallbackToCLI(ctx, evalPath, headerSection, fs, resourcePath, logger)
	}

	result, err := evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		logger.Warn("SDK evaluation failed; falling back to CLI eval", "error", err)
		return fallbackToCLI(ctx, evalPath, headerSection, fs, resourcePath, logger)
	}

	return result, nil
}

func createEvaluator(ctx context.Context, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (pkl.Evaluator, error) {
	if opts == nil {
		return NewConfiguredEvaluator(ctx, "pcf", nil)
	}
	return pkl.NewEvaluator(ctx, opts)
}

func writeResultToFile(fs afero.Fs, resourcePath, headerSection, result string, logger *logging.Logger) (string, error) {
	formattedResult := fmt.Sprintf("%s\n%s", headerSection, result)

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
	isExtension bool,
) error {
	tmpFile, cleanup, err := setupTempFile(fs, logger)
	if err != nil {
		return err
	}
	defer cleanup()

	relationshipSection, fullSections := preparePklSections(ctx, sections, pklTemplate, isExtension)

	if err := writeTempFile(tmpFile, fullSections, logger); err != nil {
		return err
	}

	processedContent, err := processTempFile(fs, ctx, tmpFile.Name(), relationshipSection, opts, processFunc, logger)
	if err != nil {
		return err
	}

	return writeFinalFile(fs, finalFileName, processedContent, logger)
}

func setupTempFile(fs afero.Fs, logger *logging.Logger) (afero.File, func(), error) {
	tmpDir, err := afero.TempDir(fs, "", "")
	if err != nil {
		logger.Error("failed to create temporary directory", "error", err)
		return nil, nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	tmpFile, err := afero.TempFile(fs, tmpDir, "*.pkl")
	if err != nil {
		logger.Error("failed to create temporary file", "error", err)
		return nil, nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	cleanup := func() {
		tmpFile.Close()
		if removeErr := fs.RemoveAll(tmpDir); removeErr != nil {
			logger.Warn("failed to clean up temporary directory", "directory", tmpDir, "error", removeErr)
		}
	}

	return tmpFile, cleanup, nil
}

func preparePklSections(ctx context.Context, sections []string, pklTemplate string, isExtension bool) (string, []string) {
	relationship := "amends"
	if isExtension {
		relationship = "extends"
	}

	relationshipSection := fmt.Sprintf(`%s "package://schema.kdeps.com/core@%s#/%s"`, relationship, schema.SchemaVersion(ctx), pklTemplate)
	fullSections := append([]string{relationshipSection}, sections...)

	return relationshipSection, fullSections
}

func writeTempFile(tmpFile afero.File, fullSections []string, logger *logging.Logger) error {
	_, err := tmpFile.Write([]byte(strings.Join(fullSections, "\n")))
	if err != nil {
		logger.Error("failed to write to temporary file", "error", err)
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	return nil
}

func processTempFile(fs afero.Fs, ctx context.Context, tmpFileName, relationshipSection string, opts func(options *pkl.EvaluatorOptions), processFunc func(fs afero.Fs, ctx context.Context, tmpFile string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error), logger *logging.Logger) (string, error) {
	processedContent, err := processFunc(fs, ctx, tmpFileName, relationshipSection, opts, logger)
	if err != nil {
		logger.Error("failed to process temporary file", "error", err)
		return "", fmt.Errorf("failed to process temporary file: %w", err)
	}
	return processedContent, nil
}

func writeFinalFile(fs afero.Fs, finalFileName, processedContent string, logger *logging.Logger) error {
	err := afero.WriteFile(fs, finalFileName, []byte(processedContent), 0o644)
	if err != nil {
		logger.Error("failed to write final file", "error", err)
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
	if err := validatePklFile(resourcePath, logger); err != nil {
		return err
	}

	evalPath, err := preparePklContent(fs, resourcePath, logger)
	if err != nil {
		return err
	}

	moduleSource := createModuleSource(evalPath)
	return validateWithSdk(ctx, moduleSource, evalPath, logger)
}

func validateWithSdk(ctx context.Context, moduleSource *pkl.ModuleSource, evalPath string, logger *logging.Logger) error {
	evaluator, err := NewConfiguredEvaluator(ctx, "pcf", nil)
	if err != nil {
		logger.Warn("SDK evaluator initialization failed; falling back to CLI validation", "error", err)
		cliErr := validateWithCli(ctx, evalPath, logger)
		return fmt.Errorf("sdk init error: %w; cli error: %w", err, cliErr)
	}

	_, err = evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		logger.Warn("SDK validation failed; falling back to CLI validation", "error", err)
		cliErr := validateWithCli(ctx, evalPath, logger)
		return fmt.Errorf("sdk validation error: %w; cli error: %w", err, cliErr)
	}

	return nil
}

func validateWithCli(ctx context.Context, evalPath string, logger *logging.Logger) error {
	_, stderr, exitCode, execErr := kdepsexec.KdepsExec(ctx, "pkl", []string{"eval", evalPath}, "", false, false, logger)
	if execErr != nil {
		logger.Error("CLI command execution failed", "stderr", stderr, "error", execErr)
		return fmt.Errorf("cli error: %w", execErr)
	}
	if exitCode != 0 {
		errMsg := fmt.Sprintf("cli validation failed with exit code %d: %s", exitCode, stderr)
		logger.Error(errMsg)
		return errors.New(errMsg)
	}
	return nil
}
