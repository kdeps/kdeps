package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewAddCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	assert.Equal(t, "install [package]", cmd.Use)
	assert.Equal(t, []string{"i"}, cmd.Aliases)
	assert.Equal(t, "Install an AI agent locally", cmd.Short)
	assert.Equal(t, "$ kdeps install ./myAgent.kdeps", cmd.Example)
}

func TestNewAddCommandExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0755)
	assert.NoError(t, err)

	// Create test package file
	agentKdepsPath := filepath.Join(testDir, "agent.kdeps")
	err = afero.WriteFile(fs, agentKdepsPath, []byte("test package"), 0644)
	assert.NoError(t, err)

	// Test error case - no arguments
	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package file
	cmd = NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package content
	cmd = NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{agentKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewAddCommandValidPackage(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0755)
	assert.NoError(t, err)

	// Create test package file with valid structure
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test\nversion: 1.0.0"), 0644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0755)
	assert.NoError(t, err)

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0644)
	assert.NoError(t, err)

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to invalid package format, but in a different way
}
