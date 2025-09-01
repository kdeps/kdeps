package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewRunCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger)
	assert.Equal(t, "run [package]", cmd.Use)
	assert.Equal(t, []string{"r"}, cmd.Aliases)
	assert.Equal(t, "Build and run a dockerized AI agent container", cmd.Short)
	assert.Equal(t, "$ kdeps run ./myAgent.kdeps", cmd.Example)
}

func TestNewRunCommandExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create test package file
	agentKdepsPath := filepath.Join(testDir, "agent.kdeps")
	err = afero.WriteFile(fs, agentKdepsPath, []byte("test package"), 0o644)
	assert.NoError(t, err)

	// Test error case - no arguments
	cmd := NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger)
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package file
	cmd = NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package content
	cmd = NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{agentKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewRunCommandDockerErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create test package file with valid structure but that will fail docker operations
	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

Name = "test-agent"
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
				Methods {
					"GET"
				}
			}
		}
	}
	AgentSettings {
		Timezone = "Etc/UTC"
		Models {
			"llama3.2:1b"
		}
		OllamaImageTag = "0.6.8"
	}
}`, schema.SchemaVersion(ctx))

	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

ActionID = "testAction"
run {
	Exec {
		["test"] = "echo 'test'"
	}
}`, schema.SchemaVersion(ctx))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to docker client initialization
}

func TestNewRunCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewRunCommand(ctx, fs, "/tmp/kdeps", nil, logging.NewTestLogger())

	// metadata assertions
	assert.Equal(t, "run [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "dockerized")

	// missing arg should error
	err := cmd.Execute()
	assert.Error(t, err)

	// non-existent file should propagate error
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err = cmd.Execute()
	assert.Error(t, err)
}
