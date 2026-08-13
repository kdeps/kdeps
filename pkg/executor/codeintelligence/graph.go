//go:build !js

package codeintelligence

import (
	"errors"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const (
	defaultGraphDBDir  = ".kdeps"
	defaultGraphDBFile = "graph.db"
)

// isGraphOperation reports whether op is handled by the kartographer-backed
// folder index/graph operations rather than LSP/rg.
//
//nolint:exhaustive // only classifies the 4 graph ops; every other operation falls to the default case
func isGraphOperation(op domain.CodeIntelligenceOperation) bool {
	switch op {
	case domain.CodeIntOpIndexFolder, domain.CodeIntOpGraphFile, domain.CodeIntOpGraphTopic, domain.CodeIntOpGraphAll:
		return true
	default:
		return false
	}
}

// graphDBPath resolves the bbolt graph index path: config.GraphDBPath when
// set, otherwise "<path>/.kdeps/graph.db".
func graphDBPath(config *domain.CodeIntelligenceConfig) (string, error) {
	if config.GraphDBPath != "" {
		return config.GraphDBPath, nil
	}
	if config.Path == "" {
		return "", errors.New("codeIntelligence: graphDBPath or path is required to locate the graph index")
	}
	return filepath.Join(config.Path, defaultGraphDBDir, defaultGraphDBFile), nil
}

// executeGraph dispatches indexFolder/graphFile/graphTopic/graphAll against a
// kartographer-backed bbolt folder index.
//
//nolint:exhaustive // only reached for the 4 graph ops (see isGraphOperation); every other operation falls to the default case
func (e *Executor) executeGraph(config *domain.CodeIntelligenceConfig) (interface{}, error) {
	dbPath, err := graphDBPath(config)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	if mkdirErr := AppFS.MkdirAll(filepath.Dir(dbPath), 0o700); mkdirErr != nil {
		return result(false, map[string]interface{}{resultError: mkdirErr.Error()}), mkdirErr
	}

	ig, err := graph.NewIndexedGraph(AppFS, log.Default(), dbPath)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	defer ig.Close()

	switch config.Operation {
	case domain.CodeIntOpIndexFolder:
		return graphIndexFolder(ig, config)
	case domain.CodeIntOpGraphFile:
		return graphFile(ig, config)
	case domain.CodeIntOpGraphTopic:
		return graphTopic(ig, config)
	case domain.CodeIntOpGraphAll:
		return graphAll(ig)
	default:
		return nil, errors.New("codeIntelligence: unsupported graph operation " + string(config.Operation))
	}
}

func graphIndexFolder(ig *graph.IndexedGraph, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for indexFolder")
	}
	n, err := ig.IndexFolder(config.Path, config.Extensions)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	return result(true, map[string]interface{}{
		"path":         config.Path,
		"filesIndexed": n,
	}), nil
}

func graphFile(ig *graph.IndexedGraph, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for graphFile")
	}
	references, related, err := ig.GraphFile(config.Path)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	return result(true, map[string]interface{}{
		"path":           config.Path,
		"references":     references,
		"relatedByTopic": related,
	}), nil
}

func graphTopic(ig *graph.IndexedGraph, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Topic == "" {
		return nil, errors.New("codeIntelligence: topic is required for graphTopic")
	}
	files, references, err := ig.GraphTopic(config.Topic)
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	return result(true, map[string]interface{}{
		"topic":      config.Topic,
		"files":      files,
		"references": references,
	}), nil
}

func graphAll(ig *graph.IndexedGraph) (interface{}, error) {
	references, roots, err := ig.GraphAll()
	if err != nil {
		return result(false, map[string]interface{}{resultError: err.Error()}), err
	}
	return result(true, map[string]interface{}{
		"references": references,
		"roots":      roots,
	}), nil
}
