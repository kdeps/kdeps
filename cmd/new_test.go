package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentCommandExecution(t *testing.T) {
	// Use a real filesystem for output files
	fs := afero.NewOsFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Create a temporary directory for the test output
	testDir := t.TempDir()
	err := os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(kdepsDir)

	// Set NON_INTERACTIVE to avoid prompts
	oldNonInteractive := os.Getenv("NON_INTERACTIVE")
	os.Setenv("NON_INTERACTIVE", "1")
	defer func() {
		if oldNonInteractive != "" {
			os.Setenv("NON_INTERACTIVE", oldNonInteractive)
		} else {
			os.Unsetenv("NON_INTERACTIVE")
		}
	}()

	// Test with agent name
	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"testagent"})
	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify agent directory was created
	exists, err := afero.DirExists(fs, "testagent")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify required files were created
	requiredFiles := []string{
		"workflow.pkl",
		"resources/client.pkl",
		"resources/exec.pkl",
		"resources/llm.pkl",
		"resources/python.pkl",
		"resources/response.pkl",
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join("testagent", file)
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists, "File %s should exist", filePath)

		// Verify file contents
		content, err := afero.ReadFile(fs, filePath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content, "File %s should not be empty", filePath)
	}

	// Test without agent name - should fail because agent name is required
	cmd = NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{})
	err = cmd.Execute()
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "accepts 1 arg", "unexpected error message")
	}
}

func TestNewAgentCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	assert.Equal(t, "new [agentName]", cmd.Use)
	assert.Equal(t, []string{"n"}, cmd.Aliases)
	assert.Equal(t, "Create a new AI agent", cmd.Short)
}

func TestNewAgentCommandMaxArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"test-agent", "extra-arg"})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 2")
}

func TestNewAgentCommandEmptyName(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"   "})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent name cannot be empty or only whitespace")
}

func TestNewAgentCommandTemplateError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Create a temporary directory for the test output
	testDir := t.TempDir()
	err := os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(kdepsDir)

	// Set TEMPLATE_DIR to a non-existent directory to force a template error
	oldTemplateDir := os.Getenv("TEMPLATE_DIR")
	os.Setenv("TEMPLATE_DIR", "/nonexistent")
	defer func() {
		if oldTemplateDir != "" {
			os.Setenv("TEMPLATE_DIR", oldTemplateDir)
		} else {
			os.Unsetenv("TEMPLATE_DIR")
		}
	}()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"test-agent"})
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template from disk")
}
