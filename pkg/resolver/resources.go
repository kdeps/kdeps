package resolver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
			return err
		}
	}

	return nil
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
func (dr *DependencyResolver) processPklFile(file string) (err error) {
	// Recover from unexpected panics while processing individual PKL files
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			dr.Logger.Error("panic while processing .pkl file", "file", file, "panic", r, "stack", string(buf[:n]))
			err = fmt.Errorf("panic processing file %s: %v", file, r)
		}
	}()
	// Load the resource file
	res, err := dr.LoadResourceFn(dr.Context, file, Resource)
	if err != nil {
		return fmt.Errorf("failed to load PKL file: %w", err)
	}

	pklRes, ok := res.(*pklResource.Resource)
	if !ok {
		return errors.New("failed to cast pklRes to *pklLLM.Resource")
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
		// Use direct evaluation to concrete type to avoid interface reflection issues
		var res *pklResource.Resource
		if err := evaluator.EvaluateModule(ctx, source, &res); err != nil {
			dr.Logger.Error("error reading resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading resource file '%s': %w", resourceFile, err)
		}
		if res == nil {
			dr.Logger.Error("got nil resource after evaluation", "resource-file", resourceFile)
			return nil, fmt.Errorf("got nil resource after evaluation for '%s'", resourceFile)
		}
		dr.Logger.Debug("successfully loaded resource", "resource-file", resourceFile)
		return res, nil

	case ExecResource:
		var res *pklExec.ExecImpl
		if err := evaluator.EvaluateModule(ctx, source, &res); err != nil {
			dr.Logger.Error("error reading exec resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading exec resource file '%s': %w", resourceFile, err)
		}
		if res == nil {
			dr.Logger.Error("got nil exec resource after evaluation", "resource-file", resourceFile)
			return nil, fmt.Errorf("got nil exec resource after evaluation for '%s'", resourceFile)
		}
		dr.Logger.Debug("successfully loaded exec resource", "resource-file", resourceFile)
		return res, nil

	case PythonResource:
		var res *pklPython.PythonImpl
		if err := evaluator.EvaluateModule(ctx, source, &res); err != nil {
			dr.Logger.Error("error reading python resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading python resource file '%s': %w", resourceFile, err)
		}
		if res == nil {
			dr.Logger.Error("got nil python resource after evaluation", "resource-file", resourceFile)
			return nil, fmt.Errorf("got nil python resource after evaluation for '%s'", resourceFile)
		}
		dr.Logger.Debug("successfully loaded python resource", "resource-file", resourceFile)
		return res, nil

	case LLMResource:
		var res *pklLLM.LLMImpl
		if err := evaluator.EvaluateModule(ctx, source, &res); err != nil {
			dr.Logger.Error("error reading llm resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading llm resource file '%s': %w", resourceFile, err)
		}
		if res == nil {
			dr.Logger.Error("got nil llm resource after evaluation", "resource-file", resourceFile)
			return nil, fmt.Errorf("got nil llm resource after evaluation for '%s'", resourceFile)
		}
		dr.Logger.Debug("successfully loaded llm resource", "resource-file", resourceFile)
		return res, nil

	case HTTPResource:
		var res *pklHTTP.HTTPImpl
		if err := evaluator.EvaluateModule(ctx, source, &res); err != nil {
			dr.Logger.Error("error reading http resource file", "resource-file", resourceFile, "error", err)
			return nil, fmt.Errorf("error reading http resource file '%s': %w", resourceFile, err)
		}
		if res == nil {
			dr.Logger.Error("got nil http resource after evaluation", "resource-file", resourceFile)
			return nil, fmt.Errorf("got nil http resource after evaluation for '%s'", resourceFile)
		}
		dr.Logger.Debug("successfully loaded http resource", "resource-file", resourceFile)
		return res, nil

	default:
		dr.Logger.Error("unknown resource type", "resource-type", resourceType)
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}
