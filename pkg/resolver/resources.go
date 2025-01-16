package resolver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kdeps/kdeps/pkg/resource"

	"github.com/spf13/afero"
)

// LoadResourceEntries loads .pkl resource files from the resources directory
func (dr *DependencyResolver) LoadResourceEntries() error {
	workflowDir := filepath.Join(dr.AgentDir, "resources")
	var pklFiles []string

	// Walk through the workflowDir to find .pkl files
	err := afero.Walk(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			dr.Logger.Errorf("Error accessing path %s: %v", path, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Handle dynamic and placeholder imports
			if err := dr.handleFileImports(path); err != nil {
				dr.Logger.Errorf("Error processing imports for file %s: %v", path, err)
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
			dr.Logger.Errorf("Error processing .pkl file %s: %v", file, err)
			return err
		}
	}

	return nil
}

// handleFileImports handles dynamic and placeholder imports for a given file
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

// processPklFile processes an individual .pkl file and updates dependencies
func (dr *DependencyResolver) processPklFile(file string) error {
	// Load the resource file
	pklRes, err := resource.LoadResource(dr.Context, file, dr.Logger)
	if err != nil {
		return fmt.Errorf("failed to load resource from .pkl file %s: %w", file, err)
	}

	// Append the resource to the list of resources
	dr.Resources = append(dr.Resources, ResourceNodeEntry{
		Id:   pklRes.Id,
		File: file,
	})

	// Update resource dependencies
	if pklRes.Requires != nil {
		dr.ResourceDependencies[pklRes.Id] = *pklRes.Requires
	} else {
		dr.ResourceDependencies[pklRes.Id] = nil
	}

	return nil
}
