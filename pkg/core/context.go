package core

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/spf13/afero"
)

// KdepsContext holds the unified context for kdeps operations
type KdepsContext struct {
	GraphID        string
	CurrentAgent   string
	CurrentVersion string
	KdepsPath      string
	PklresReader   *pklres.PklResourceReader
	AgentReader    *agent.PklResourceReader
	Logger         *logging.Logger
	Fs             afero.Fs
}

var (
	globalContext *KdepsContext
	contextMutex  sync.RWMutex
)

// InitializeContext creates and initializes the global kdeps context
func InitializeContext(fs afero.Fs, graphID, currentAgent, currentVersion, kdepsPath string, logger *logging.Logger) error {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	// Initialize pklres reader
	pklresReader, err := pklres.InitializePklResource(graphID, currentAgent, currentVersion, kdepsPath, fs)
	if err != nil {
		return fmt.Errorf("failed to initialize pklres reader: %w", err)
	}

	// Set the global pklres reader
	pklres.SetGlobalPklresReader(pklresReader)

	// Initialize agent reader
	agentReader, err := agent.InitializeAgent(fs, kdepsPath, currentAgent, currentVersion, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize agent reader: %w", err)
	}

	globalContext = &KdepsContext{
		GraphID:        graphID,
		CurrentAgent:   currentAgent,
		CurrentVersion: currentVersion,
		KdepsPath:      kdepsPath,
		PklresReader:   pklresReader,
		AgentReader:    agentReader,
		Logger:         logger,
		Fs:             fs,
	}

	logger.Debug("initialized global kdeps context",
		"graphID", graphID,
		"agent", currentAgent,
		"version", currentVersion,
		"kdepsPath", kdepsPath)

	return nil
}

// GetContext returns the global kdeps context
func GetContext() *KdepsContext {
	contextMutex.RLock()
	defer contextMutex.RUnlock()
	return globalContext
}

// UpdateContext updates the global context with new parameters
func UpdateContext(graphID, currentAgent, currentVersion, kdepsPath string) error {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	if globalContext == nil {
		return errors.New("global context not initialized")
	}

	// Update context fields
	if graphID != "" {
		globalContext.GraphID = graphID
	}
	if currentAgent != "" {
		globalContext.CurrentAgent = currentAgent
	}
	if currentVersion != "" {
		globalContext.CurrentVersion = currentVersion
	}
	if kdepsPath != "" {
		globalContext.KdepsPath = kdepsPath
	}

	// Update pklres reader context
	if err := pklres.UpdateGlobalPklresReaderContext(graphID, currentAgent, currentVersion, kdepsPath); err != nil {
		return fmt.Errorf("failed to update pklres reader context: %w", err)
	}

	// Update agent reader context (agent reader handles its own context updates)
	globalContext.Logger.Debug("updated global kdeps context",
		"graphID", globalContext.GraphID,
		"agent", globalContext.CurrentAgent,
		"version", globalContext.CurrentVersion,
		"kdepsPath", globalContext.KdepsPath)

	return nil
}

// CloseContext closes the global context and cleans up resources
func CloseContext() error {
	contextMutex.Lock()
	defer contextMutex.Unlock()

	if globalContext == nil {
		return nil
	}

	var errs []error

	// Close agent reader
	if globalContext.AgentReader != nil {
		if err := globalContext.AgentReader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close agent reader: %w", err))
		}
	}

	// Note: pklres reader uses global singleton, so we don't close it here
	// It will be managed by the pklres package

	globalContext = nil

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// ResolveActionID provides a simplified action ID resolution
func (ctx *KdepsContext) ResolveActionID(actionID string) (string, error) {
	if ctx == nil || ctx.AgentReader == nil {
		return "", errors.New("context or agent reader not initialized")
	}

	// If already in canonical format, return as-is
	if len(actionID) > 0 && actionID[0] == '@' {
		return actionID, nil
	}

	// Use current context to resolve
	if ctx.CurrentAgent == "" || ctx.CurrentVersion == "" {
		return "", errors.New("current agent and version must be set for action ID resolution")
	}

	// Create URI for agent ID resolution
	query := url.Values{}
	query.Set("op", "resolve")
	query.Set("agent", ctx.CurrentAgent)
	query.Set("version", ctx.CurrentVersion)
	uri := url.URL{
		Scheme:   "agent",
		Path:     "/" + actionID,
		RawQuery: query.Encode(),
	}

	resolvedIDBytes, err := ctx.AgentReader.Read(uri)
	if err != nil {
		return "", fmt.Errorf("failed to resolve action ID: %w", err)
	}

	return string(resolvedIDBytes), nil
}
