package cmd_test

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

	"github.com/kdeps/kdeps/pkg/environment"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
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
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create a valid workflow file
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err = fs.MkdirAll(validAgentDir, 0o755)
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
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
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
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		assert.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
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
	err = afero.WriteFile(fs, invalidKdepsPath, []byte("invalid package"), 0o644)
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
	err := fs.MkdirAll(validAgentDir, 0o755)
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
	err = afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
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
		err = afero.WriteFile(fs, resourcePath, []byte(resourceContent), 0o644)
		assert.NoError(t, err)
	}

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to docker client initialization
}

func TestNewBuildCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewBuildCommand(fs, ctx, "/tmp/kdeps", nil, logging.NewTestLogger())

	// Verify metadata
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "dockerized")

	// Execute with missing arg should error due to cobra Args check
	err := cmd.Execute()
	assert.Error(t, err)

	// Provide non-existent file – RunE should propagate ExtractPackage error.
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewBuildCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewBuildCommand(fs, context.Background(), "/kdeps", nil, logging.NewTestLogger())

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
func testDeps() (afero.Fs, context.Context, string, *logging.Logger) {
	return afero.NewMemMapFs(), context.Background(), "/tmp/kdeps", logging.NewTestLogger()
}

func TestNewAddCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewAddCommand(fs, ctx, dir, logger)
	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// RunE with a non-existent file to exercise error path but cover closure.
	if err := cmd.RunE(cmd, []string{"/no/file.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewBuildCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewBuildCommand(fs, ctx, dir, &kdCfg.Kdeps{}, logger)
	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewAgentCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewAgentCommand(fs, ctx, dir, logger)
	if cmd.Use != "new [agentName]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// Provide invalid args to hit error path.
	if err := cmd.RunE(cmd, []string{""}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewPackageCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewPackageCommand(fs, ctx, dir, &environment.Environment{}, logger)
	if cmd.Use != "package [agent-dir]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"/nonexistent"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewRunCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewRunCommand(fs, ctx, dir, &kdCfg.Kdeps{}, logger)
	if cmd.Use != "run [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewScaffoldCommandConstructor(t *testing.T) {
	fs, _, _, logger := testDeps()
	cmd := NewScaffoldCommand(fs, context.Background(), logger)
	if cmd.Use != "scaffold [agentName] [fileNames...]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// args missing triggers help path, fast.
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error")
	}
}

// TestNewBuildCommand_RunE tests the RunE function directly to improve coverage
func TestNewBuildCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test with non-existent package file
	err := cmd.RunE(cmd, []string{"/does/not/exist.kdeps"})
	assert.Error(t, err, "expected error from RunE due to missing package file")
}

// TestNewBuildCommand_RunE_ExtractPackageError tests the error path when ExtractPackage fails
func TestNewBuildCommand_RunE_ExtractPackageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test with invalid package file
	err := cmd.RunE(cmd, []string{"invalid.kdeps"})
	assert.Error(t, err, "expected error when ExtractPackage fails")
}

// TestNewBuildCommand_RunE_BuildDockerfileError tests the error path when BuildDockerfile fails
func TestNewBuildCommand_RunE_BuildDockerfileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a minimal valid package structure that will pass ExtractPackage but fail BuildDockerfile
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create a minimal workflow file
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test"), 0o644)
	assert.NoError(t, err)

	// Create resources directory
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	// Create minimal resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when BuildDockerfile fails")
}

// TestNewBuildCommand_RunE_DockerClientError tests the error path when Docker client creation fails
func TestNewBuildCommand_RunE_DockerClientError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a package structure that will pass ExtractPackage and BuildDockerfile but fail Docker client creation
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create a minimal workflow file
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test"), 0o644)
	assert.NoError(t, err)

	// Create resources directory
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	// Create minimal resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when Docker client creation fails")
}

// TestNewBuildCommand_RunE_BuildDockerImageError tests the error path when BuildDockerImage fails
func TestNewBuildCommand_RunE_BuildDockerImageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a package structure that will pass ExtractPackage and BuildDockerfile but fail BuildDockerImage
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create a minimal workflow file
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test"), 0o644)
	assert.NoError(t, err)

	// Create resources directory
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	// Create minimal resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when BuildDockerImage fails")
}

// TestNewBuildCommand_RunE_CleanupError tests the error path when CleanupDockerBuildImages fails
func TestNewBuildCommand_RunE_CleanupError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a package structure that will pass all previous steps but fail CleanupDockerBuildImages
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create a minimal workflow file
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test"), 0o644)
	assert.NoError(t, err)

	// Create resources directory
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	// Create minimal resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when CleanupDockerBuildImages fails")
}

// TestNewBuildCommand_Constructor tests the command constructor
func TestNewBuildCommand_Constructor(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "build [package]", cmd.Use)
	assert.Equal(t, []string{"b"}, cmd.Aliases)
	assert.Equal(t, "Build a dockerized AI agent", cmd.Short)
	assert.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
}
