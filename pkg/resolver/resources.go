package resolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/schema"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	pklResource "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
)

// ResourceType defines the type of resource to load.
type ResourceType string

const (
	ExecResource   ResourceType = "exec"
	PythonResource ResourceType = "python"
	LLMResource    ResourceType = "llm"
	HTTPResource   ResourceType = "http"
	Resource       ResourceType = "resource"
)

// LoadResourceEntries loads .pkl resource files from the resources directory.
func (dr *DependencyResolver) LoadResourceEntries() error {
	workflowDir := filepath.Join(dr.WorkflowDir, "resources")
	var pklFiles []string

	// Walk through the workflowDir to find .pkl files
	err := afero.Walk(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			dr.Logger.Errorf("error accessing path %s: %v", path, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Handle dynamic and placeholder imports
			if err := dr.handleFileImports(path); err != nil {
				dr.Logger.Errorf("error reading imports for file %s: %v", path, err)
				return err
			}

			// Add the file to the list of .pkl files
			pklFiles = append(pklFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk through the workflow directory: %w", err)
	}

	// Process all .pkl files found
	for _, file := range pklFiles {
		if err := dr.processPklFile(file); err != nil {
			dr.Logger.Errorf("error processing .pkl file %s: %v", file, err)
			return err
		}
	}

	return nil
}

// handleFileImports handles dynamic and placeholder imports for a given file.
func (dr *DependencyResolver) handleFileImports(path string) error {
	// Prepend dynamic imports
	if err := dr.PrependDynamicImports(path); err != nil {
		return fmt.Errorf("failed to prepend dynamic imports for file %s: %w", path, err)
	}

	// Add placeholder imports
	if err := dr.AddPlaceholderImports(path); err != nil {
		return fmt.Errorf("failed to add placeholder imports for file %s: %w", path, err)
	}

	return nil
}

// processPklFile processes an individual .pkl file and updates dependencies.
func (dr *DependencyResolver) processPklFile(file string) error {
	// Load the resource file, without outputting to a file
	res, _, err := dr.LoadResource(dr.Context, file, Resource, false)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	pklRes, ok := res.(*pklResource.Resource)
	if !ok {
		return errors.New("failed to cast pklRes to *pklResource.Resource")
	}

	// Append the resource to the list of resources
	dr.Resources = append(dr.Resources, ResourceNodeEntry{
		ActionID:     pklRes.ActionID,
		File:         file,
		CompiledFile: "",
	})

	// Update resource dependencies
	if pklRes.Requires != nil {
		dr.ResourceDependencies[pklRes.ActionID] = *pklRes.Requires
	} else {
		dr.ResourceDependencies[pklRes.ActionID] = nil
	}

	return nil
}

// LoadResource reads a resource file and returns the parsed resource object, optionally writing the output to a temporary Pkl file.
// If outputToFile is true, the output is written to a temporary file with an 'amends' line prepended (using schema.SchemaVersion), and its path is returned; otherwise, an empty string is returned.
func (dr *DependencyResolver) LoadResource(ctx context.Context, resourceFile string, resourceType ResourceType, outputToFile bool) (interface{}, string, error) {
	// Log additional info before reading the resource
	dr.Logger.Debug("reading resource file", "resource-file", resourceFile, "resource-type", resourceType, "output-to-file", outputToFile)

	// Resolve absolute path for the resource file
	absResourceFile, err := filepath.Abs(resourceFile)
	if err != nil {
		dr.Logger.Error("failed to resolve absolute path", "resource-file", resourceFile, "error", err)
		return nil, "", fmt.Errorf("failed to resolve absolute path for %s: %w", resourceFile, err)
	}

	var outputFileName string
	if outputToFile {
		// Create a temporary output file
		outputFile, err := os.CreateTemp("", "pkl-*.pkl")
		if err != nil {
			dr.Logger.Error("failed to create temp file", "error", err)
			return nil, "", fmt.Errorf("failed to create temp file: %w", err)
		}
		outputFileName = outputFile.Name()
		outputFile.Close()
	}

	// Define an option function to configure EvaluatorOptions
	opts := func(options *pkl.EvaluatorOptions) {
		pkl.WithDefaultAllowedResources(options)
		pkl.WithOsEnv(options)
		pkl.WithDefaultAllowedModules(options)
		pkl.WithDefaultCacheDir(options)
		options.Logger = pkl.NoopLogger
		options.ResourceReaders = []pkl.ResourceReader{
			dr.MemoryReader,
			dr.SessionReader,
			dr.ToolReader,
			dr.ItemReader,
		}
		options.AllowedModules = []string{".*"}
		options.AllowedResources = []string{".*"}
		options.OutputFormat = "pcf" // Ensure Pkl output format
	}

	// Create evaluator
	evaluator, err := pkl.NewEvaluator(ctx, opts)
	if err != nil {
		dr.Logger.Error("error creating evaluator", "error", err)
		return nil, "", fmt.Errorf("error creating evaluator: %w", err)
	}
	defer func() {
		if cerr := evaluator.Close(); cerr != nil && err == nil {
			err = cerr
			dr.Logger.Error("error closing evaluator", "error", err)
		}
	}()

	// Load the resource based on the resource type
	source := pkl.FileSource(absResourceFile)
	var res interface{}
	switch resourceType {
	case Resource:
		res, err = pklResource.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
		}
	case ExecResource:
		res, err = pklExec.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading exec resource file", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error reading exec resource file '%s': %w", resourceFile, err)
		}
	case PythonResource:
		res, err = pklPython.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading python resource file", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error reading python resource file '%s': %w", resourceFile, err)
		}
	case LLMResource:
		res, err = pklLLM.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading llm resource file", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error reading llm resource file '%s': %w", resourceFile, err)
		}
	case HTTPResource:
		res, err = pklHTTP.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading http resource file", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error reading http resource file '%s': %w", resourceFile, err)
		}
	default:
		dr.Logger.Error("unknown resource type", "resource-type", resourceType)
		return nil, "", fmt.Errorf("unknown resource type: %s", resourceType)
	}

	// If outputToFile, write the output text to the temporary file with amends line
	if outputToFile {
		output, err := evaluator.EvaluateOutputText(ctx, source)
		if err != nil {
			dr.Logger.Error("error evaluating output text", "resource-file", resourceFile, "error", err)
			return nil, "", fmt.Errorf("error evaluating output text for '%s': %w", resourceFile, err)
		}
		// // Get schema version
		version := schema.SchemaVersion(ctx)
		// Prepend the amends line with the schema version
		amendsLine := fmt.Sprintf("amends \"package://schema.kdeps.com/core@%s#/Resource.pkl\"\n\n", version)
		outputContent := amendsLine + output

		if err := os.WriteFile(outputFileName, []byte(outputContent), 0o644); err != nil {
			dr.Logger.Error("failed to write to output file", "output-file", outputFileName, "error", err)
			return nil, "", fmt.Errorf("failed to write to %s: %w", outputFileName, err)
		}
	}

	dr.Logger.Debug("successfully loaded resource", "resource-file", resourceFile, "output-file", outputFileName)
	return res, outputFileName, nil
}
