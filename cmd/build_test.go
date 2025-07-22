package cmd_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	assets "github.com/kdeps/schema/assets"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/environment"
)

func TestNewBuildCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger)
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Equal(t, []string{"b"}, cmd.Aliases)
	assert.Equal(t, "Build a dockerized AI agent", cmd.Short)
	assert.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
}

func TestNewBuildCommandExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	// Create test directory
	testDir := filepath.Join("/test")
	err = fs.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	// Create a valid workflow file
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err = fs.MkdirAll(validAgentDir, 0o755)
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

	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	require.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	require.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "%s"

actionID = "testAction"
run {
	exec {
		["test"] = "echo 'test'"
	}
}`, workspace.GetImportPath("Resource.pkl"))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		require.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	require.NoError(t, err)

	// Test error case - no arguments
	cmd := NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger)
	err = cmd.Execute()
	require.Error(t, err)

	// Test error case - nonexistent file
	cmd = NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	require.Error(t, err)

	// Test error case - invalid package content
	invalidKdepsPath := filepath.Join(testDir, "invalid.kdeps")
	err = afero.WriteFile(fs, invalidKdepsPath, []byte("invalid package"), 0o644)
	require.NoError(t, err)
	cmd = NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{invalidKdepsPath})
	err = cmd.Execute()
	require.Error(t, err)
}

func TestNewBuildCommandDockerErrors(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	// Create test directory
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err = fs.MkdirAll(validAgentDir, 0o755)
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

	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	require.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	require.NoError(t, err)

	resourceContent := fmt.Sprintf(`amends "%s"

actionID = "testAction"
run {
	exec {
		["test"] = "echo 'test'"
	}
}`, workspace.GetImportPath("Resource.pkl"))

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		require.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	require.NoError(t, err)

	cmd := NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	require.Error(t, err) // Should fail due to docker client initialization
}

func TestNewBuildCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewBuildCommand(ctx, fs, t.TempDir(), nil, logging.NewTestLogger())

	// Verify metadata
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "dockerized")

	// Execute with missing arg should error due to cobra Args check
	err := cmd.Execute()
	require.Error(t, err)

	// Provide non-existent file – RunE should propagate ExtractPackage error.
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err = cmd.Execute()
	require.Error(t, err)
}

func TestNewBuildCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewBuildCommand(context.Background(), fs, "/kdeps", nil, logging.NewTestLogger())

	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "b" {
		t.Fatalf("expected alias 'b'")
	}
	if cmd.Short == "" {
		t.Fatalf("Short description should not be empty")
	}
}

// helper returns common deps for command constructors.
func testDeps(t *testing.T) (afero.Fs, context.Context, string, *logging.Logger) {
	return afero.NewMemMapFs(), context.Background(), t.TempDir(), logging.NewTestLogger()
}

func TestNewAddCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps(t)
	cmd := NewAddCommand(ctx, fs, dir, logger)
	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// RunE with a non-existent file to exercise error path but cover closure.
	if err := cmd.RunE(cmd, []string{"/no/file.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewBuildCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps(t)
	cmd := NewBuildCommand(ctx, fs, dir, &kdeps.Kdeps{}, logger)
	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewAgentCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps(t)
	cmd := NewAgentCommand(ctx, fs, dir, logger)
	if cmd.Use != "new [agentName]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// Provide invalid args to hit error path.
	if err := cmd.RunE(cmd, []string{""}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewPackageCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps(t)
	cmd := NewPackageCommand(ctx, fs, dir, &environment.Environment{}, logger)
	if cmd.Use != "package [agent-dir]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"/nonexistent"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewRunCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps(t)
	cmd := NewRunCommand(ctx, fs, dir, &kdeps.Kdeps{}, logger)
	if cmd.Use != "run [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewScaffoldCommandConstructor(t *testing.T) {
	fs, _, _, logger := testDeps(t)
	cmd := NewScaffoldCommand(context.Background(), fs, logger)
	if cmd.Use != "scaffold [agentName] [fileNames...]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// args missing triggers help path, fast.
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error")
	}
}
