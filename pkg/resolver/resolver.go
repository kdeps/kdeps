package resolver

import (
	"fmt"
	"kdeps/pkg/resource"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph"
	"github.com/spf13/afero"
)

type DependencyResolver struct {
	Fs                   afero.Fs
	Logger               *log.Logger
	Resources            []ResourceNodeEntry
	ResourceDependencies map[string][]string
	DependencyGraph      []string
	VisitedPaths         map[string]bool
	Graph                *graph.DependencyGraph
	AgentDir             string
}

type ResourceNodeEntry struct {
	Id   string `pkl:"id"`
	File string `pkl:"file"`
}

func NewGraphResolver(fs afero.Fs, logger *log.Logger, agentDir string) (*DependencyResolver, error) {
	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		AgentDir:             agentDir,
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger, dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, fmt.Errorf("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}

func (dr *DependencyResolver) LoadResourceEntries() error {
	// Get all .pkl files in the directory using afero
	if err := afero.Walk(dr.Fs, filepath.Join(dr.AgentDir, "resources"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// LogError("Error walking through files", err)
			return err
		}

		// Check if the file has a .pkl extension
		if !info.IsDir() && filepath.Ext(path) == ".pkl" {
			// Load the resource file
			pklRes, err := resource.LoadResource(path)
			if err != nil {
				fmt.Errorf("Error loading .pkl file "+path, err)
				return nil // Continue walking even if there’s an error
			}

			dr.Resources = append(dr.Resources, ResourceNodeEntry{
				Id:   pklRes.Id,
				File: path,
			})

			if pklRes.Requires != nil {
				dr.ResourceDependencies[pklRes.Id] = *pklRes.Requires
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
