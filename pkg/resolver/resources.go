package resolver

import (
	"fmt"
	"kdeps/pkg/resource"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

func (dr *DependencyResolver) LoadResourceEntries() error {
	workflowDir := filepath.Join(dr.AgentDir, "resources")
	var pklFiles []string
	if err := afero.Walk(dr.Fs, workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Errorf("Error walking through files: %s - %s", workflowDir, err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			if err := dr.PrependDynamicImports(path); err != nil {
				fmt.Errorf("Failed to prepend dynamic imports "+path, err)
			}

			if err := dr.AddPlaceholderImports(path); err != nil {
				fmt.Errorf("Unable to create placeholder imports for .pkl file "+path, err)
			}

			pklFiles = append(pklFiles, path)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, file := range pklFiles {
		// Load the resource file
		pklRes, err := resource.LoadResource(*dr.Context, file, dr.Logger)
		if err != nil {
			fmt.Errorf("Error loading .pkl file "+file, err)
		}

		dr.Resources = append(dr.Resources, ResourceNodeEntry{
			Id:   pklRes.Id,
			File: file,
		})

		if pklRes.Requires != nil {
			dr.ResourceDependencies[pklRes.Id] = *pklRes.Requires
		} else {
			dr.ResourceDependencies[pklRes.Id] = nil
		}
	}

	return nil
}
