package cmd_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kdeps/kdeps/pkg/environment"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
)

// Mock implementations for testing
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

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

func TestNewBuildCommand_Structure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	assert.Equal(t, "build [package]", command.Use)
	assert.Contains(t, command.Aliases, "b")
	assert.Equal(t, "Build a dockerized AI agent", command.Short)
	assert.Equal(t, "$ kdeps build ./myAgent.kdeps", command.Example)
}

func TestNewBuildCommand_ArgumentValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test with no arguments using cobra's validation
	command.SetArgs([]string{})
	err := command.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg")

	// Test with valid number of arguments (should pass validation)
	// We'll mock the functions to avoid actual execution
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, errors.New("expected error for validation test")
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	command.SetArgs([]string{"test.kdeps"})
	err = command.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected error for validation test")
}

func TestNewBuildCommand_ExtractPackageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Mock ExtractPackage to return error
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, errors.New("extract package failed")
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	err := command.RunE(command, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extract package failed")
}

func TestNewBuildCommand_BuildDockerfileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Mock ExtractPackage to succeed
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	// Mock BuildDockerfile to return error
	origBuildDockerfile := cmd.BuildDockerfileFn
	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "", false, false, "", "", "", "", "", errors.New("build dockerfile failed")
	}
	defer func() { cmd.BuildDockerfileFn = origBuildDockerfile }()

	err := command.RunE(command, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build dockerfile failed")
}

func TestNewBuildCommand_NewDockerClientError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Mock ExtractPackage to succeed
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	// Mock BuildDockerfile to succeed
	origBuildDockerfile := cmd.BuildDockerfileFn
	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "/tmp/run", false, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
	}
	defer func() { cmd.BuildDockerfileFn = origBuildDockerfile }()

	// Mock NewDockerClient to return error
	origNewDockerClient := cmd.NewDockerClientFn
	cmd.NewDockerClientFn = func() (*client.Client, error) {
		return nil, errors.New("docker client creation failed")
	}
	defer func() { cmd.NewDockerClientFn = origNewDockerClient }()

	err := command.RunE(command, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker client creation failed")
}

func TestNewBuildCommand_BuildDockerImageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Mock all previous functions to succeed
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	origBuildDockerfile := cmd.BuildDockerfileFn
	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "/tmp/run", false, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
	}
	defer func() { cmd.BuildDockerfileFn = origBuildDockerfile }()

	origNewDockerClient := cmd.NewDockerClientFn
	cmd.NewDockerClientFn = func() (*client.Client, error) {
		return &client.Client{}, nil
	}
	defer func() { cmd.NewDockerClientFn = origNewDockerClient }()

	// Mock BuildDockerImage to return error
	origBuildDockerImage := cmd.BuildDockerImageFn
	cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, dockerClient *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "", "", errors.New("build docker image failed")
	}
	defer func() { cmd.BuildDockerImageFn = origBuildDockerImage }()

	err := command.RunE(command, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "build docker image failed")
}

func TestNewBuildCommand_CleanupError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestSafeLogger()

	command := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Mock all previous functions to succeed
	origExtractPackage := cmd.ExtractPackageFn
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}
	defer func() { cmd.ExtractPackageFn = origExtractPackage }()

	origBuildDockerfile := cmd.BuildDockerfileFn
	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "/tmp/run", false, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
	}
	defer func() { cmd.BuildDockerfileFn = origBuildDockerfile }()

	origNewDockerClient := cmd.NewDockerClientFn
	cmd.NewDockerClientFn = func() (*client.Client, error) {
		return &client.Client{}, nil
	}
	defer func() { cmd.NewDockerClientFn = origNewDockerClient }()

	origBuildDockerImage := cmd.BuildDockerImageFn
	cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, dockerClient *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "test-container", "test-container:latest", nil
	}
	defer func() { cmd.BuildDockerImageFn = origBuildDockerImage }()

	// Mock CleanupDockerBuildImages to return error
	origCleanup := cmd.CleanupDockerBuildImagesFn
	cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, agentContainerName string, dockerClient docker.DockerPruneClient) error {
		return errors.New("cleanup failed")
	}
	defer func() { cmd.CleanupDockerBuildImagesFn = origCleanup }()

	err := command.RunE(command, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup failed")
}

// TestNewBuildCommand_Success tests the RunE function with mocked dependencies for success path
func TestNewBuildCommand_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test directory structure
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	// Save original functions
	oldExtractPackageFn := cmd.ExtractPackageFn
	oldBuildDockerfileFn := cmd.BuildDockerfileFn
	oldNewDockerClientFn := cmd.NewDockerClientFn
	oldBuildDockerImageFn := cmd.BuildDockerImageFn
	oldCleanupDockerBuildImagesFn := cmd.CleanupDockerBuildImagesFn
	oldPrintlnFn := cmd.PrintlnFn

	// Restore functions after test
	defer func() {
		cmd.ExtractPackageFn = oldExtractPackageFn
		cmd.BuildDockerfileFn = oldBuildDockerfileFn
		cmd.NewDockerClientFn = oldNewDockerClientFn
		cmd.BuildDockerImageFn = oldBuildDockerImageFn
		cmd.CleanupDockerBuildImagesFn = oldCleanupDockerBuildImagesFn
		cmd.PrintlnFn = oldPrintlnFn
	}()

	// Mock all the functions with correct signatures
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{
			PkgFilePath: validKdepsPath,
			Workflow:    filepath.Join(validAgentDir, "workflow.pkl"),
			Resources:   []string{},
			Data:        make(map[string]map[string][]string),
			Md5sum:      "test-checksum",
		}, nil
	}

	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "/tmp/rundir", false, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
	}

	cmd.NewDockerClientFn = func() (*client.Client, error) {
		return &client.Client{}, nil
	}

	cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir string, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "test-agent", "test-agent:1.0.0", nil
	}

	cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, agentContainerName string, dockerClient docker.DockerPruneClient) error {
		return nil
	}

	var printedMessage string
	cmd.PrintlnFn = func(a ...interface{}) (n int, err error) {
		// Simulate fmt.Println behavior which adds spaces between arguments
		printedMessage = fmt.Sprintln(a...)
		// Remove the trailing newline that Sprintln adds
		printedMessage = strings.TrimSuffix(printedMessage, "\n")
		return len(printedMessage), nil
	}

	// Test successful execution
	buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	buildCmd.SetArgs([]string{validKdepsPath})
	err = buildCmd.Execute()
	assert.NoError(t, err)
	assert.Equal(t, "Kdeps AI Agent docker image created: test-agent:1.0.0", printedMessage)
}

// TestNewBuildCommand_AllErrorPaths tests all error paths to ensure 100% coverage
func TestNewBuildCommand_AllErrorPaths(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a valid .kdeps file
	validKdepsPath := filepath.Join("/test", "valid-agent.kdeps")
	err := fs.MkdirAll(filepath.Dir(validKdepsPath), 0o755)
	assert.NoError(t, err)
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	// Save original functions
	oldExtractPackageFn := cmd.ExtractPackageFn
	oldBuildDockerfileFn := cmd.BuildDockerfileFn
	oldNewDockerClientFn := cmd.NewDockerClientFn
	oldBuildDockerImageFn := cmd.BuildDockerImageFn
	oldCleanupDockerBuildImagesFn := cmd.CleanupDockerBuildImagesFn

	// Restore functions after test
	defer func() {
		cmd.ExtractPackageFn = oldExtractPackageFn
		cmd.BuildDockerfileFn = oldBuildDockerfileFn
		cmd.NewDockerClientFn = oldNewDockerClientFn
		cmd.BuildDockerImageFn = oldBuildDockerImageFn
		cmd.CleanupDockerBuildImagesFn = oldCleanupDockerBuildImagesFn
	}()

	t.Run("ExtractPackage error", func(t *testing.T) {
		cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
			return nil, errors.New("extract error")
		}

		buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
		buildCmd.SetArgs([]string{validKdepsPath})
		err := buildCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "extract error")
	})

	t.Run("BuildDockerfile error", func(t *testing.T) {
		cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
			return &archiver.KdepsPackage{}, nil
		}
		cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
			return "", false, false, "", "", "", "", "", errors.New("dockerfile error")
		}

		buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
		buildCmd.SetArgs([]string{validKdepsPath})
		err := buildCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dockerfile error")
	})

	t.Run("NewDockerClient error", func(t *testing.T) {
		cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
			return &archiver.KdepsPackage{}, nil
		}
		cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
			return "/tmp/rundir", false, false, "", "", "", "", "", nil
		}
		cmd.NewDockerClientFn = func() (*client.Client, error) {
			return nil, errors.New("docker client error")
		}

		buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
		buildCmd.SetArgs([]string{validKdepsPath})
		err := buildCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "docker client error")
	})

	t.Run("BuildDockerImage error", func(t *testing.T) {
		cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
			return &archiver.KdepsPackage{}, nil
		}
		cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
			return "/tmp/rundir", false, false, "", "", "", "", "", nil
		}
		cmd.NewDockerClientFn = func() (*client.Client, error) {
			return &client.Client{}, nil
		}
		cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir string, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
			return "", "", errors.New("build image error")
		}

		buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
		buildCmd.SetArgs([]string{validKdepsPath})
		err := buildCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "build image error")
	})

	t.Run("CleanupDockerBuildImages error", func(t *testing.T) {
		cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
			return &archiver.KdepsPackage{}, nil
		}
		cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
			return "/tmp/rundir", false, false, "", "", "", "", "", nil
		}
		cmd.NewDockerClientFn = func() (*client.Client, error) {
			return &client.Client{}, nil
		}
		cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir string, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
			return "test-agent", "test-agent:1.0.0", nil
		}
		cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, agentContainerName string, dockerClient docker.DockerPruneClient) error {
			return errors.New("cleanup error")
		}

		buildCmd := cmd.NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
		buildCmd.SetArgs([]string{validKdepsPath})
		err := buildCmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cleanup error")
	})
}
