package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewScaffoldCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	assert.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
	assert.Equal(t, "Scaffold specific files for an agent", cmd.Short)
	assert.Contains(t, cmd.Long, "Available resources:")
}

func TestNewScaffoldCommandNoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0755)
	assert.NoError(t, err)

	cmd := NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir})
	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestNewScaffoldCommandValidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0755)
	assert.NoError(t, err)

	validResources := []string{"client", "exec", "llm", "python", "response"}

	for _, resource := range validResources {
		cmd := NewScaffoldCommand(fs, ctx, logger)
		cmd.SetArgs([]string{testAgentDir, resource})
		err := cmd.Execute()
		assert.NoError(t, err)

		// Verify file was created
		filePath := filepath.Join(testAgentDir, "resources", resource+".pkl")
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists, "File %s should exist", filePath)
	}
}

func TestNewScaffoldCommandInvalidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0755)
	assert.NoError(t, err)

	cmd := NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir, "invalid-resource"})
	err = cmd.Execute()
	assert.NoError(t, err) // Command doesn't return error for invalid resources

	// Verify file was not created
	filePath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err := afero.Exists(fs, filePath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandMultipleResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0755)
	assert.NoError(t, err)

	cmd := NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir, "client", "exec", "invalid-resource"})
	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify valid files were created
	clientPath := filepath.Join(testAgentDir, "resources", "client.pkl")
	exists, err := afero.Exists(fs, clientPath)
	assert.NoError(t, err)
	assert.True(t, exists, "File %s should exist", clientPath)

	execPath := filepath.Join(testAgentDir, "resources", "exec.pkl")
	exists, err = afero.Exists(fs, execPath)
	assert.NoError(t, err)
	assert.True(t, exists, "File %s should exist", execPath)

	// Verify invalid file was not created
	invalidPath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err = afero.Exists(fs, invalidPath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandNoArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	err := cmd.Execute()
	assert.Error(t, err) // Should fail due to missing required argument
}
