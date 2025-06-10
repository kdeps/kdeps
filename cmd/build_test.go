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

func TestNewBuildCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Equal(t, []string{"b"}, cmd.Aliases)
	assert.Equal(t, "Build a dockerized AI agent", cmd.Short)
	assert.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
}

func TestNewBuildCommandExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0755)
	assert.NoError(t, err)

	// Create a valid workflow file
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err = fs.MkdirAll(validAgentDir, 0755)
	assert.NoError(t, err)

	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "test-agent"
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

	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0755)
	assert.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

actionID = "testAction"
run {
	exec {
		["test"] = "echo 'test'"
	}
}`, schema.SchemaVersion(ctx))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0644)
		assert.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0644)
	assert.NoError(t, err)

	// Test error case - no arguments
	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - nonexistent file
	cmd = NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package content
	invalidKdepsPath := filepath.Join(testDir, "invalid.kdeps")
	err = afero.WriteFile(fs, invalidKdepsPath, []byte("invalid package"), 0644)
	assert.NoError(t, err)
	cmd = NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{invalidKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewBuildCommandDockerErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0755)
	assert.NoError(t, err)

	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

name = "test-agent"
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

	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0755)
	assert.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"

actionID = "testAction"
run {
	exec {
		["test"] = "echo 'test'"
	}
}`, schema.SchemaVersion(ctx))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0644)
		assert.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to docker client initialization
}
