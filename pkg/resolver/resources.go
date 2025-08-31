package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/evaluator"
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
	walkFn := dr.WalkFn
	if walkFn == nil {
		walkFn = afero.Walk
	}

	err := walkFn(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			dr.Logger.Errorf("error accessing path %s: %v", path, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Handle dynamic and placeholder imports
			if err := dr.handleFileImports(path); err != nil {
				dr.Logger.Errorf("error processing imports for file %s: %v", path, err)
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
			// Continue processing other files instead of failing completely
			// This allows the system to work even if some resource files are malformed
			continue
		}
	}

	return nil
}

// loadResourceWithFallback tries to load a resource file with different resource types as fallback
func (dr *DependencyResolver) loadResourceWithFallback(file string) (interface{}, error) {
	resourceTypes := []ResourceType{Resource, LLMResource, HTTPResource, PythonResource, ExecResource}

	for _, resourceType := range resourceTypes {
		res, err := dr.LoadResourceFn(dr.Context, file, resourceType)
		if err != nil {
			dr.Logger.Debug("failed to load resource with type", "file", file, "type", resourceType, "error", err)
			continue
		}

		dr.Logger.Debug("successfully loaded resource", "file", file, "type", resourceType)

		// If we successfully loaded as a specific resource type, try to convert it to Resource type
		if resourceType != Resource {
			// Try to convert the loaded resource to Resource type
			convertedRes, convertErr := dr.convertToResourceType(res, resourceType, file)
			if convertErr == nil {
				return convertedRes, nil
			}
			dr.Logger.Debug("failed to convert resource to Resource type", "file", file, "originalType", resourceType, "error", convertErr)
			// Continue with the original loaded resource if conversion fails
		}

		return res, nil
	}

	return nil, fmt.Errorf("failed to load resource with any type")
}

// convertToResourceType attempts to convert a loaded resource to Resource type
func (dr *DependencyResolver) convertToResourceType(res interface{}, originalType ResourceType, file string) (interface{}, error) {
	// Try to load the same file as Resource type
	resourceRes, err := dr.LoadResourceFn(dr.Context, file, Resource)
	if err != nil {
		return nil, fmt.Errorf("failed to load as Resource type: %w", err)
	}
	return resourceRes, nil
}

// handleFileImports handles dynamic and placeholder imports for a given file.
func (dr *DependencyResolver) handleFileImports(path string) error {
	// Prepend dynamic imports
	if dr.PrependDynamicImportsFn != nil {
		if err := dr.PrependDynamicImportsFn(path); err != nil {
			return fmt.Errorf("failed to prepend dynamic imports for file %s: %w", path, err)
		}
	} else if err := dr.PrependDynamicImports(path); err != nil {
		return fmt.Errorf("failed to prepend dynamic imports for file %s: %w", path, err)
	}

	// Add placeholder imports
	if dr.AddPlaceholderImportsFn != nil {
		if err := dr.AddPlaceholderImportsFn(path); err != nil {
			return fmt.Errorf("failed to add placeholder imports for file %s: %w", path, err)
		}
	} else if err := dr.AddPlaceholderImports(path); err != nil {
		return fmt.Errorf("failed to add placeholder imports for file %s: %w", path, err)
	}

	return nil
}

// processPklFile processes an individual .pkl file and updates dependencies.
func (dr *DependencyResolver) processPklFile(file string) error {
	// Check if file exists before trying to load it
	if _, err := dr.Fs.Stat(file); err != nil {
		dr.Logger.Warn("PKL file does not exist, skipping", "file", file, "error", err)
		return nil // Skip missing files instead of failing
	}

	// Try to load the resource file, with fallback to different resource types
	res, err := dr.loadResourceWithFallback(file)
	if err != nil {
		dr.Logger.Error("failed to load PKL file with any resource type", "file", file, "error", err)
		return fmt.Errorf("failed to load PKL file %s with any resource type: %w", file, err)
	}

	var pklRes pklResource.Resource
	if ptr, ok := res.(*pklResource.Resource); ok {
		pklRes = *ptr
	} else if resource, ok := res.(pklResource.Resource); ok {
		pklRes = resource
	} else {
		dr.Logger.Error("failed to cast resource to pklResource.Resource",
			"file", file,
			"actualType", fmt.Sprintf("%T", res))
		return fmt.Errorf("failed to cast resource to pklResource.Resource for file %s (actual type: %T)", file, res)
	}

	// Append the resource to the list of resources
	dr.Resources = append(dr.Resources, ResourceNodeEntry{
		ActionID: pklRes.ActionID,
		File:     file,
	})

	// Update resource dependencies
	if pklRes.Requires != nil {
		dr.ResourceDependencies[pklRes.ActionID] = *pklRes.Requires
	} else {
		dr.ResourceDependencies[pklRes.ActionID] = nil
	}

	return nil
}

// LoadResource reads a resource file and returns the parsed resource object or an error.
func (dr *DependencyResolver) LoadResource(ctx context.Context, resourceFile string, resourceType ResourceType) (interface{}, error) {
	// Log additional info before reading the resource
	dr.Logger.Debug("reading resource file", "resource-file", resourceFile, "resource-type", resourceType)

	// Create evaluator using centralized helper in pkg/evaluator with readers
	evaluator, err := evaluator.NewConfiguredEvaluator(ctx, "", dr.getResourceReaders())
	if err != nil {
		dr.Logger.Error("error creating evaluator", "error", err)
		return nil, fmt.Errorf("error creating evaluator: %w", err)
	}
	defer func() {
		if cerr := evaluator.Close(); cerr != nil && err == nil {
			err = cerr
			dr.Logger.Error("error closing evaluator", "error", err)
		}
	}()

	// Load the resource based on the resource type
	source := pkl.FileSource(resourceFile)
	switch resourceType {
	case Resource:
		res, err := pklResource.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded resource", "resource-file", resourceFile)
		return res, nil

	case ExecResource:
		res, err := pklExec.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading exec resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading exec resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded exec resource", "resource-file", resourceFile)
		return res, nil

	case PythonResource:
		res, err := pklPython.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading python resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading python resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded python resource", "resource-file", resourceFile)
		return res, nil

	case LLMResource:
		res, err := pklLLM.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading llm resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading llm resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded llm resource", "resource-file", resourceFile)
		return res, nil

	case HTTPResource:
		res, err := pklHTTP.Load(ctx, evaluator, source)
		if err != nil {
			dr.Logger.Error("error reading http resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading http resource file '%s': %w", resourceFile, err)
		}
		dr.Logger.Debug("successfully loaded http resource", "resource-file", resourceFile)
		return res, nil

	default:
		dr.Logger.Error("unknown resource type", "resource-type", resourceType)
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}
