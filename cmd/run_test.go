package cmd_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	versionpkg "github.com/kdeps/kdeps/pkg/version"
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
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
	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package file
	cmd = NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package content
	cmd = NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
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
		ollamaImageTag = "%s"
	}
}`, schema.SchemaVersion(ctx), versionpkg.DefaultOllamaImageTag)

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

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to docker client initialization
}

func TestNewRunCommand_MetadataAndErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewRunCommand(fs, ctx, "/tmp/kdeps", nil, logging.NewTestLogger())

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

// TestNewRunCommand_RunE tests the RunE function directly to improve coverage
func TestNewRunCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test with non-existent package file
	err := cmd.RunE(cmd, []string{"/does/not/exist.kdeps"})
	assert.Error(t, err, "expected error from RunE due to missing package file")
}

// TestNewRunCommand_RunE_ExtractPackageError tests the error path when ExtractPackage fails
func TestNewRunCommand_RunE_ExtractPackageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test with invalid package file
	err := cmd.RunE(cmd, []string{"invalid.kdeps"})
	assert.Error(t, err, "expected error when ExtractPackage fails")
}

// TestNewRunCommand_RunE_BuildDockerfileError tests the error path when BuildDockerfile fails
func TestNewRunCommand_RunE_BuildDockerfileError(t *testing.T) {
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when BuildDockerfile fails")
}

// TestNewRunCommand_RunE_DockerClientError tests the error path when Docker client creation fails
func TestNewRunCommand_RunE_DockerClientError(t *testing.T) {
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when Docker client creation fails")
}

// TestNewRunCommand_RunE_BuildDockerImageError tests the error path when BuildDockerImage fails
func TestNewRunCommand_RunE_BuildDockerImageError(t *testing.T) {
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when BuildDockerImage fails")
}

// TestNewRunCommand_RunE_CleanupError tests the error path when CleanupDockerBuildImages fails
func TestNewRunCommand_RunE_CleanupError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a package structure that will pass ExtractPackage and BuildDockerfile but fail CleanupDockerBuildImages
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when CleanupDockerBuildImages fails")
}

// TestNewRunCommand_RunE_CreateDockerContainerError tests the error path when CreateDockerContainer fails
func TestNewRunCommand_RunE_CreateDockerContainerError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create a package structure that will pass all previous steps but fail CreateDockerContainer
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

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err = cmd.RunE(cmd, []string{validKdepsPath})
	assert.Error(t, err, "expected error when CreateDockerContainer fails")
}

// TestNewRunCommand_Constructor tests the command constructor
func TestNewRunCommand_Constructor(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "run [package]", cmd.Use)
	assert.Equal(t, []string{"r"}, cmd.Aliases)
	assert.Equal(t, "Build and run a dockerized AI agent container", cmd.Short)
	assert.Equal(t, "$ kdeps run ./myAgent.kdeps", cmd.Example)
}

// TestNewRunCommand_Success tests the full success path with mocked dependencies
func TestNewRunCommand_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Store original functions
	originalExtractPackageFn := cmd.ExtractPackageFn
	originalBuildDockerfileFn := cmd.BuildDockerfileFn
	originalNewDockerClientFn := cmd.NewDockerClientFn
	originalBuildDockerImageFn := cmd.BuildDockerImageFn
	originalCleanupDockerBuildImagesFn := cmd.CleanupDockerBuildImagesFn
	originalNewDockerClientAdapterFn := cmd.NewDockerClientAdapterFn
	originalCreateDockerContainerFn := cmd.CreateDockerContainerFn
	originalPrintlnFn := cmd.PrintlnFn

	// Restore original functions after test
	defer func() {
		cmd.ExtractPackageFn = originalExtractPackageFn
		cmd.BuildDockerfileFn = originalBuildDockerfileFn
		cmd.NewDockerClientFn = originalNewDockerClientFn
		cmd.BuildDockerImageFn = originalBuildDockerImageFn
		cmd.CleanupDockerBuildImagesFn = originalCleanupDockerBuildImagesFn
		cmd.NewDockerClientAdapterFn = originalNewDockerClientAdapterFn
		cmd.CreateDockerContainerFn = originalCreateDockerContainerFn
		cmd.PrintlnFn = originalPrintlnFn
	}()

	// Mock all injectable functions for success path
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
	}

	cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "/tmp/run-dir", true, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
	}

	cmd.NewDockerClientFn = func() (*client.Client, error) {
		return &client.Client{}, nil
	}

	cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "test-agent", "test-agent:latest", nil
	}

	cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, cName string, cli docker.DockerPruneClient) error {
		return nil
	}

	cmd.NewDockerClientAdapterFn = func(dockerClient *client.Client) *docker.DockerClientAdapter {
		return &docker.DockerClientAdapter{}
	}

	cmd.CreateDockerContainerFn = func(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum, webHostIP, webPortNum, gpu string, apiMode, webMode bool, cli docker.DockerClient) (string, error) {
		return "container-id-123", nil
	}

	var printedMessage string
	cmd.PrintlnFn = func(a ...interface{}) (n int, err error) {
		printedMessage = fmt.Sprintln(a...)
		return len(printedMessage), nil
	}

	// Test the success path
	runCmd := cmd.NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	err := runCmd.RunE(runCmd, []string{"test.kdeps"})

	assert.NoError(t, err)
	assert.Contains(t, printedMessage, "container-id-123")
}

// TestNewRunCommand_AllErrorPaths tests individual error paths with mocked dependencies
func TestNewRunCommand_AllErrorPaths(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Store original functions
	originalExtractPackageFn := cmd.ExtractPackageFn
	originalBuildDockerfileFn := cmd.BuildDockerfileFn
	originalNewDockerClientFn := cmd.NewDockerClientFn
	originalBuildDockerImageFn := cmd.BuildDockerImageFn
	originalCleanupDockerBuildImagesFn := cmd.CleanupDockerBuildImagesFn
	originalNewDockerClientAdapterFn := cmd.NewDockerClientAdapterFn
	originalCreateDockerContainerFn := cmd.CreateDockerContainerFn
	originalPrintlnFn := cmd.PrintlnFn

	// Restore original functions after test
	defer func() {
		cmd.ExtractPackageFn = originalExtractPackageFn
		cmd.BuildDockerfileFn = originalBuildDockerfileFn
		cmd.NewDockerClientFn = originalNewDockerClientFn
		cmd.BuildDockerImageFn = originalBuildDockerImageFn
		cmd.CleanupDockerBuildImagesFn = originalCleanupDockerBuildImagesFn
		cmd.NewDockerClientAdapterFn = originalNewDockerClientAdapterFn
		cmd.CreateDockerContainerFn = originalCreateDockerContainerFn
		cmd.PrintlnFn = originalPrintlnFn
	}()

	tests := []struct {
		name          string
		setupMocks    func()
		expectedError string
	}{
		{
			name: "ExtractPackage error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return nil, fmt.Errorf("extract error")
				}
			},
			expectedError: "extract error",
		},
		{
			name: "BuildDockerfile error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
				}
				cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
					return "", false, false, "", "", "", "", "", fmt.Errorf("dockerfile error")
				}
			},
			expectedError: "dockerfile error",
		},
		{
			name: "NewDockerClient error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
				}
				cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
					return "/tmp/run-dir", true, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
				}
				cmd.NewDockerClientFn = func() (*client.Client, error) {
					return nil, fmt.Errorf("docker client error")
				}
			},
			expectedError: "docker client error",
		},
		{
			name: "BuildDockerImage error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
				}
				cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
					return "/tmp/run-dir", true, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
				}
				cmd.NewDockerClientFn = func() (*client.Client, error) {
					return &client.Client{}, nil
				}
				cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
					return "", "", fmt.Errorf("build image error")
				}
			},
			expectedError: "build image error",
		},
		{
			name: "CleanupDockerBuildImages error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
				}
				cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
					return "/tmp/run-dir", true, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
				}
				cmd.NewDockerClientFn = func() (*client.Client, error) {
					return &client.Client{}, nil
				}
				cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
					return "test-agent", "test-agent:latest", nil
				}
				cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, cName string, cli docker.DockerPruneClient) error {
					return fmt.Errorf("cleanup error")
				}
			},
			expectedError: "cleanup error",
		},
		{
			name: "CreateDockerContainer error",
			setupMocks: func() {
				cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
					return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
				}
				cmd.BuildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
					return "/tmp/run-dir", true, false, "127.0.0.1", "8080", "127.0.0.1", "3000", "cpu", nil
				}
				cmd.NewDockerClientFn = func() (*client.Client, error) {
					return &client.Client{}, nil
				}
				cmd.BuildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
					return "test-agent", "test-agent:latest", nil
				}
				cmd.CleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, cName string, cli docker.DockerPruneClient) error {
					return nil
				}
				cmd.NewDockerClientAdapterFn = func(dockerClient *client.Client) *docker.DockerClientAdapter {
					return &docker.DockerClientAdapter{}
				}
				cmd.CreateDockerContainerFn = func(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum, webHostIP, webPortNum, gpu string, apiMode, webMode bool, cli docker.DockerClient) (string, error) {
					return "", fmt.Errorf("create container error")
				}
			},
			expectedError: "create container error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			runCmd := cmd.NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
			err := runCmd.RunE(runCmd, []string{"test.kdeps"})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
