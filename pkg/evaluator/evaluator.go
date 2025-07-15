package evaluator

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// EvalPkl evaluates the resource file at resourcePath using the singleton PKL evaluator.
// If the file content is a quoted PKL code string like "new <Dynamic,Listing,Mapping,etc..> {...}", it removes the quotes and writes it to a temporary file for evaluation.
func EvalPkl(fs afero.Fs, ctx context.Context, resourcePath string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
	// Validate that the file has a .pkl extension
	if filepath.Ext(resourcePath) != ".pkl" {
		errMsg := fmt.Sprintf("file '%s' must have a .pkl extension", resourcePath)
		logger.Error(errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	// Get the singleton evaluator
	evaluator, err := GetEvaluator()
	if err != nil {
		logger.Error("failed to get PKL evaluator", "resourcePath", resourcePath, "error", err)
		return "", fmt.Errorf("error getting evaluator for %s: %w", resourcePath, err)
	}

	// Create a ModuleSource using UriSource for paths with a protocol, FileSource for relative paths
	var moduleSource *pkl.ModuleSource
	parsedURL, err := url.Parse(resourcePath)
	if err == nil && parsedURL.Scheme != "" {
		// Has a protocol (e.g., file://, http://, https://)
		moduleSource = pkl.UriSource(resourcePath)
	} else {
		// Absolute path without a protocol
		moduleSource = pkl.FileSource(resourcePath)
	}

	// Evaluate the Pkl file
	result, err := evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		errMsg := "failed to evaluate Pkl file"
		logger.Error(errMsg, "resourcePath", resourcePath, "error", err)
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
	relationshipSection := fmt.Sprintf(`%s "package://schema.kdeps.com/core@%s#/%s"`, relationship, schema.Version(ctx), pklTemplate)
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

// EvaluateAllPklFilesInDirectory evaluates all PKL files in the given directory to test for any problems.
// This is useful during packaging to ensure all PKL files are valid and can be evaluated without errors.
func EvaluateAllPklFilesInDirectory(fs afero.Fs, ctx context.Context, dir string, logger *logging.Logger) error {
	// Get the singleton evaluator
	evaluator, err := GetEvaluator()
	if err != nil {
		return fmt.Errorf("PKL evaluator not available: %w", err)
	}

	// Walk through the directory and find all PKL files
	err = afero.Walk(fs, dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-PKL files
		if info.IsDir() || filepath.Ext(path) != ".pkl" {
			return nil
		}

		logger.Debug("evaluating PKL file", "file", path)

		// Create module source
		moduleSource := pkl.FileSource(path)

		// Evaluate the PKL file
		_, err = evaluator.EvaluateOutputText(ctx, moduleSource)
		if err != nil {
			logger.Error("PKL file evaluation failed", "file", path, "error", err)
			return fmt.Errorf("evaluation failed for %s: %w", path, err)
		}

		logger.Debug("PKL file evaluation successful", "file", path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to evaluate PKL files in directory %s: %w", dir, err)
	}

	logger.Info("all PKL files evaluated successfully", "directory", dir)
	return nil
}

// EvaluateText evaluates PKL text directly using the singleton evaluator
func EvaluateText(ctx context.Context, pklText string, logger *logging.Logger) (string, error) {
	// Get the singleton evaluator
	evaluator, err := GetEvaluator()
	if err != nil {
		return "", fmt.Errorf("PKL evaluator not available: %w", err)
	}

	// Create a text source
	moduleSource := pkl.TextSource(pklText)

	// Evaluate the PKL text
	result, err := evaluator.EvaluateOutputText(ctx, moduleSource)
	if err != nil {
		logger.Error("PKL text evaluation failed", "error", err)
		return "", fmt.Errorf("evaluation failed: %w", err)
	}

	return result, nil
}
