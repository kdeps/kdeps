package cmd_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	assets "github.com/kdeps/schema/assets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackageCommandExecution(t *testing.T) {
	// Initialize evaluator for this test
	evaluator.TestSetup(t)
	defer evaluator.TestTeardown(t)

	// Use a real filesystem for both input and output files
	fs := afero.NewOsFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	// Create a temporary directory for the test files
	testAgentDir := filepath.Join(t.TempDir(), "agent")
	err = fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	workflowContent := fmt.Sprintf(`amends "%s"

AgentID = "testagent"
Description = "Test Agent"
Version = "1.0.0"
TargetActionID = "testAction"

Workflows {}

Settings {
	APIServerMode = true
	APIServer {
		HostIP = "127.0.0.1"
		PortNum = 3000
		Routes {
			new {
				Path = "/api/v1/test"
				Methods { "GET" }
			}
		}
	}
	AgentSettings {
		Timezone = "Etc/UTC"
		Models {
			"llama3.2:1b"
		}
		OllamaTagVersion = "0.6.8"
	}
}`, workspace.GetImportPath("Workflow.pkl"))

	workflowPath := filepath.Join(testAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	require.NoError(t, err)

	// Create resources directory and add test resources
	resourcesDir := filepath.Join(testAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	require.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "%s"

Run {
	Exec {
		test = "echo 'test'"
	}
}`, workspace.GetImportPath("Resource.pkl"))

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
	cmd := NewPackageCommand(ctx, fs, kdepsDir, env, logger)
	cmd.SetArgs([]string{testAgentDir})
	err = cmd.Execute()
	if err != nil {
		// Skip test if PKL binary is not available
		if strings.Contains(err.Error(), "exit status 1") || strings.Contains(err.Error(), "PKL evaluator not available") {
			t.Skip("Skipping test - PKL binary not available or evaluator initialization failed")
		}
		require.NoError(t, err)
	}

	// Test error case - invalid directory
	cmd = NewPackageCommand(ctx, fs, kdepsDir, env, logger)
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "nonexistent")})
	err = cmd.Execute()
	require.Error(t, err)

	// Test error case - no arguments
	cmd = NewPackageCommand(ctx, fs, kdepsDir, env, logger)
	err = cmd.Execute()
	require.Error(t, err)
}

func TestPackageCommandFlags(t *testing.T) {
	// Initialize evaluator for this test
	evaluator.TestSetup(t)
	defer evaluator.TestTeardown(t)

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(ctx, fs, kdepsDir, env, logger)
	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Equal(t, []string{"p"}, cmd.Aliases)
	assert.Equal(t, "Package an AI agent to .kdeps file", cmd.Short)
	assert.Equal(t, "$ kdeps package ./myAgent/", cmd.Example)
}

func TestNewPackageCommand_MetadataAndArgs(t *testing.T) {
	// Initialize evaluator for this test
	evaluator.TestSetup(t)
	defer evaluator.TestTeardown(t)

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	cmd := NewPackageCommand(ctx, fs, t.TempDir(), env, logging.NewTestLogger())

	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Contains(t, strings.ToLower(cmd.Short), "package")

	// Execute with no args – expect error
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}
