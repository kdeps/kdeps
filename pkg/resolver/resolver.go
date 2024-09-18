package resolver

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph"
	"github.com/spf13/afero"
)

type DependencyResolver struct {
	Fs                   afero.Fs
	Logger               *log.Logger
	ResourceDependencies map[string][]string
	DependencyGraph      []string
	VisitedPaths         map[string]bool
	Graph                *graph.DependencyGraph
	WorkDir              string
}

func NewGraphResolver(fs afero.Fs, logger *log.Logger, workDir string) (*DependencyResolver, error) {
	dependencyResolver := &DependencyResolver{
		Fs:                   fs,
		ResourceDependencies: make(map[string][]string),
		Logger:               logger,
		VisitedPaths:         make(map[string]bool),
		WorkDir:              workDir,
	}

	dependencyResolver.Graph = graph.NewDependencyGraph(fs, logger, dependencyResolver.ResourceDependencies)
	if dependencyResolver.Graph == nil {
		return nil, fmt.Errorf("failed to initialize dependency graph")
	}

	return dependencyResolver, nil
}
