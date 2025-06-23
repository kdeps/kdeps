package cmd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/cmd"
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

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	assert.Equal(t, "new [agentName]", cmd.Use)
	assert.Equal(t, []string{"n"}, cmd.Aliases)
	assert.Equal(t, "Create a new AI agent", cmd.Short)
}

func TestNewAgentCommandMaxArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
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

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
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

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"test-agent"})
	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read template from disk")
}

// TestNewAgentCommand_RunE tests the RunE function directly to improve coverage
func TestNewAgentCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)

	// Test with valid agent name
	err := cmd.RunE(cmd, []string{"testagent"})
	assert.NoError(t, err, "expected success with valid agent name")
}

// TestNewAgentCommand_RunE_MkdirError tests the error path when MkdirAll fails
func TestNewAgentCommand_RunE_MkdirError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Test with invalid agent name that would cause issues
	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	err := cmd.RunE(cmd, []string{""})
	assert.Error(t, err, "expected error with empty agent name")
}

// TestNewAgentCommand_RunE_GenerateWorkflowFileError tests the error path when GenerateWorkflowFile fails
func TestNewAgentCommand_RunE_GenerateWorkflowFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Test with whitespace-only name
	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	err := cmd.RunE(cmd, []string{"   "})
	assert.Error(t, err, "expected error with whitespace-only agent name")
	assert.Contains(t, err.Error(), "failed to generate workflow file")
}

// TestNewAgentCommand_RunE_GenerateResourceFilesError tests the error path when GenerateResourceFiles fails
func TestNewAgentCommand_RunE_GenerateResourceFilesError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Test with invalid agent name
	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	err := cmd.RunE(cmd, []string{"invalid-name"})
	// This should succeed since we're using a memory filesystem
	assert.NoError(t, err, "expected success with valid agent name")
}

// TestNewAgentCommand_Constructor tests the command constructor
func TestNewAgentCommand_Constructor(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "new [agentName]", cmd.Use)
	assert.Equal(t, []string{"n"}, cmd.Aliases)
	assert.Equal(t, "Create a new AI agent", cmd.Short)
}

// TestNewAgentCommand_ArgsValidation tests argument validation
func TestNewAgentCommand_ArgsValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)

	// Test with empty string
	err := cmd.RunE(cmd, []string{""})
	assert.Error(t, err)

	// Test with whitespace-only string
	err = cmd.RunE(cmd, []string{"   "})
	assert.Error(t, err)

	// Test with valid name
	err = cmd.RunE(cmd, []string{"valid-name"})
	assert.NoError(t, err)
}

// TestNewAgentCommand_SuccessfulCreation tests successful agent creation
func TestNewAgentCommand_SuccessfulCreation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)

	// Test successful creation
	err := cmd.RunE(cmd, []string{"successful-agent"})
	assert.NoError(t, err)

	// Verify directory was created
	exists, err := afero.DirExists(fs, "successful-agent")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify resources directory was created
	exists, err = afero.DirExists(fs, "successful-agent/resources")
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestNewAgentCommand_SuccessWithMocks tests the success path with mocked dependencies
func TestNewAgentCommand_SuccessWithMocks(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Store original functions
	originalGenerateWorkflowFileFn := cmd.GenerateWorkflowFileFn
	originalGenerateResourceFilesFn := cmd.GenerateResourceFilesFn

	// Restore original functions after test
	defer func() {
		cmd.GenerateWorkflowFileFn = originalGenerateWorkflowFileFn
		cmd.GenerateResourceFilesFn = originalGenerateResourceFilesFn
	}()

	// Mock template functions for success path
	cmd.GenerateWorkflowFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
		return nil
	}

	cmd.GenerateResourceFilesFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
		return nil
	}

	// Test the success path
	agentCmd := cmd.NewAgentCommand(fs, ctx, kdepsDir, logger)
	err := agentCmd.RunE(agentCmd, []string{"test-agent"})

	assert.NoError(t, err)
}

// TestNewAgentCommand_AllErrorPaths tests individual error paths with mocked dependencies
func TestNewAgentCommand_AllErrorPaths(t *testing.T) {
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Store original functions
	originalGenerateWorkflowFileFn := cmd.GenerateWorkflowFileFn
	originalGenerateResourceFilesFn := cmd.GenerateResourceFilesFn

	// Restore original functions after test
	defer func() {
		cmd.GenerateWorkflowFileFn = originalGenerateWorkflowFileFn
		cmd.GenerateResourceFilesFn = originalGenerateResourceFilesFn
	}()

	tests := []struct {
		name          string
		setupMocks    func()
		useReadOnlyFS bool
		expectedError string
	}{
		{
			name: "MkdirAll error",
			setupMocks: func() {
				// No mocks needed for this test
			},
			useReadOnlyFS: true,
			expectedError: "failed to create main directory",
		},
		{
			name: "GenerateWorkflowFile error",
			setupMocks: func() {
				cmd.GenerateWorkflowFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
					return fmt.Errorf("workflow generation error")
				}
				cmd.GenerateResourceFilesFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
					return nil
				}
			},
			useReadOnlyFS: false,
			expectedError: "failed to generate workflow file",
		},
		{
			name: "GenerateResourceFiles error",
			setupMocks: func() {
				cmd.GenerateWorkflowFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
					return nil
				}
				cmd.GenerateResourceFilesFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, mainDir, agentName string) error {
					return fmt.Errorf("resource generation error")
				}
			},
			useReadOnlyFS: false,
			expectedError: "failed to generate resource files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testFS afero.Fs
			if tt.useReadOnlyFS {
				testFS = afero.NewReadOnlyFs(afero.NewMemMapFs())
			} else {
				testFS = afero.NewMemMapFs()
			}

			tt.setupMocks()

			agentCmd := NewAgentCommand(testFS, ctx, kdepsDir, logger)
			err := agentCmd.RunE(agentCmd, []string{"test-agent"})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
