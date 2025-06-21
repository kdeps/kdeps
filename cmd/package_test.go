package cmd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackageCommandExecution(t *testing.T) {
	// Use a real filesystem for both input and output files
	fs := afero.NewOsFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a temporary directory for the test files
	testAgentDir := filepath.Join(t.TempDir(), "agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "testagent"
description = "Test Agent"
version = "1.0.0"
targetActionID = "testAction"

workflows {
	default {
		name = "Default Workflow"
		description = "Default workflow for testing"
		steps {
			step1 {
				name = "Test Step"
				description = "A test step"
				actionID = "testAction"
			}
		}
	}
}

settings {
	APIServerMode = true
	APIServer {
		hostIP = "127.0.0.1"
		portNum = 3000
		routes {
			new {
				path = "/api/v1/test"
				methods {
					"GET"
				}
			}
		}
	}
	agentSettings {
		timezone = "Etc/UTC"
		models {
			"llama3.2:1b"
		}
		ollamaImageTag = "0.6.8"
	}
}`, schema.SchemaVersion(ctx))

	workflowPath := filepath.Join(testAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	require.NoError(t, err)

	// Create resources directory and add test resources
	resourcesDir := filepath.Join(testAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	require.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

actionID = "testAction"
run {
	exec {
		test = "echo 'test'"
	}
}`, schema.SchemaVersion(ctx))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		require.NoError(t, err)
	}

	// Create a temporary directory for the test output
	testDir := t.TempDir()
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(kdepsDir)

	// Test successful case
	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	cmd.SetArgs([]string{testAgentDir})
	err = cmd.Execute()
	assert.NoError(t, err)

	// Test error case - invalid directory
	cmd = NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "nonexistent")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - no arguments
	cmd = NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestPackageCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Equal(t, []string{"p"}, cmd.Aliases)
	assert.Equal(t, "Package an AI agent to .kdeps file", cmd.Short)
	assert.Equal(t, "$ kdeps package ./myAgent/", cmd.Example)
}

func TestNewPackageCommand_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	cmd := NewPackageCommand(fs, ctx, "/tmp/kdeps", env, logging.NewTestLogger())

	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Contains(t, strings.ToLower(cmd.Short), "package")

	// Execute with no args – expect error
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

// TestNewPackageCommand_RunE tests the RunE function directly to improve coverage
func TestNewPackageCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)

	// Test with non-existent directory
	err := cmd.RunE(cmd, []string{"/does/not/exist"})
	assert.Error(t, err, "expected error from RunE due to missing directory")
}

// TestNewPackageCommand_RunE_FindWorkflowFileError tests the error path when FindWorkflowFile fails
func TestNewPackageCommand_RunE_FindWorkflowFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a directory without a workflow file
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	err = cmd.RunE(cmd, []string{testDir})
	assert.Error(t, err, "expected error when FindWorkflowFile fails")
}

// TestNewPackageCommand_RunE_LoadWorkflowError tests the error path when LoadWorkflow fails
func TestNewPackageCommand_RunE_LoadWorkflowError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a directory with an invalid workflow file
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create an invalid workflow file
	workflowPath := filepath.Join(testDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("invalid workflow content"), 0o644)
	assert.NoError(t, err)

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	err = cmd.RunE(cmd, []string{testDir})
	assert.Error(t, err, "expected error when LoadWorkflow fails")
}

// TestNewPackageCommand_RunE_CompileProjectError tests the error path when CompileProject fails
func TestNewPackageCommand_RunE_CompileProjectError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a directory with a valid workflow file but missing resources
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create a minimal valid workflow file
	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "testagent"
description = "Test Agent"
version = "1.0.0"
targetActionID = "testAction"

workflows {}

settings {
	APIServerMode = true
	APIServer {
		hostIP = "127.0.0.1"
		portNum = 3000
		routes {
			new {
				path = "/api/v1/test"
				methods {
					"GET"
				}
			}
		}
	}
	agentSettings {
		timezone = "Etc/UTC"
		models {
			"llama3.2:1b"
		}
		ollamaImageTag = "0.6.8"
	}
}`, schema.SchemaVersion(ctx))

	workflowPath := filepath.Join(testDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	assert.NoError(t, err)

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	err = cmd.RunE(cmd, []string{testDir})
	assert.Error(t, err, "expected error when CompileProject fails due to missing resources")
}

// TestNewPackageCommand_Constructor tests the command constructor
func TestNewPackageCommand_Constructor(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Equal(t, []string{"p"}, cmd.Aliases)
	assert.Equal(t, "Package an AI agent to .kdeps file", cmd.Short)
	assert.Equal(t, "$ kdeps package ./myAgent/", cmd.Example)
}

// TestNewPackageCommand_ErrorStyling tests that error messages are properly styled
func TestNewPackageCommand_ErrorStyling(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)

	// Test with non-existent directory to trigger error styling
	err := cmd.RunE(cmd, []string{"/does/not/exist"})
	assert.Error(t, err)

	// Check that the error message contains styling (errorStyle.Render)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "Error finding workflow file")
}
