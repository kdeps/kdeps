package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDevBuildMode(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	kdepsDir := "/test/kdeps"
	cacheDir := filepath.Join(kdepsDir, "cache")
	binaryFile := filepath.Join(cacheDir, "kdeps")

	// Test case: Binary file exists and is valid
	require.NoError(t, fs.MkdirAll(cacheDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, binaryFile, []byte("binary content"), 0o755))

	devBuildMode, err := checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.True(t, devBuildMode, "Expected devBuildMode to be true when binary file exists")

	// Test case: Binary file does not exist
	require.NoError(t, fs.Remove(binaryFile))

	devBuildMode, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.False(t, devBuildMode, "Expected devBuildMode to be false when binary file does not exist")

	// Test case: Path exists but is not a file
	require.NoError(t, fs.Mkdir(binaryFile, 0o755))

	devBuildMode, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.False(t, devBuildMode, "Expected devBuildMode to be false when path is not a regular file")
}

func TestBuildDockerImage(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	// Create test directories
	runDir := "/test/run"
	kdepsDir := "/test/kdeps"
	require.NoError(t, fs.MkdirAll(runDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Create a mock workflow file with proper schema reference
	workflowContent := `
amends "package://schema.kdeps.com/core@1.0.0#/Workflow.pkl"

name = "test-agent"
version = "1.0.0"
description = "Test agent"
targetActionID = "test-action"
settings {
  APIServerMode = false
  agentSettings {
    packages {
      "test-package"
    }
  }
}
`
	workflowPath := filepath.Join(runDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644))

	// Create a mock package project
	pkgProject := &archiver.KdepsPackage{
		Workflow: workflowPath,
	}

	// Create a mock kdeps configuration
	kdeps := &kdCfg.Kdeps{
		RunMode: "docker",
	}

	// Create a mock Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)

	// Test case: Build Docker image
	_, _, err = BuildDockerImage(fs, ctx, kdeps, cli, runDir, kdepsDir, pkgProject, logger)
	// Since we're using a mock environment, we expect an error about the workflow file
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error reading workflow file")
}

func TestGenerateDockerfile(t *testing.T) {
	t.Parallel()

	// Test case: Basic Dockerfile generation
	dockerfile := generateDockerfile(
		"latest",
		"1.0.0",
		"localhost",
		"8080",
		"http://localhost:8080",
		"ARG TEST_ARG=value",
		"ENV TEST_ENV=value",
		"RUN echo 'test'",
		"RUN pip install test",
		"RUN conda install test",
		"2023.09",
		"1.0.0",
		"UTC",
		"8080",
		false,
		false,
		false,
		false,
	)

	// Verify basic structure
	assert.Contains(t, dockerfile, "FROM ollama/ollama:latest")
	assert.Contains(t, dockerfile, "ENV SCHEMA_VERSION=1.0.0")
	assert.Contains(t, dockerfile, "ENV OLLAMA_HOST=localhost:8080")
	assert.Contains(t, dockerfile, "ENV KDEPS_HOST=http://localhost:8080")
	assert.Contains(t, dockerfile, "ARG TEST_ARG=value")
	assert.Contains(t, dockerfile, "ENV TEST_ENV=value")
	assert.Contains(t, dockerfile, "RUN echo 'test'")
	assert.Contains(t, dockerfile, "RUN pip install test")

	// Test case: With Anaconda installation
	dockerfile = generateDockerfile(
		"latest",
		"1.0.0",
		"localhost",
		"8080",
		"http://localhost:8080",
		"",
		"",
		"",
		"",
		"",
		"2023.09",
		"1.0.0",
		"UTC",
		"8080",
		true,
		false,
		false,
		false,
	)

	assert.Contains(t, dockerfile, "COPY cache /cache")
	assert.Contains(t, dockerfile, "RUN chmod +x /cache/anaconda*")

	// Test case: With dev build mode
	dockerfile = generateDockerfile(
		"latest",
		"1.0.0",
		"localhost",
		"8080",
		"http://localhost:8080",
		"",
		"",
		"",
		"",
		"",
		"2023.09",
		"1.0.0",
		"UTC",
		"8080",
		false,
		true,
		false,
		false,
	)

	assert.Contains(t, dockerfile, "RUN cp /cache/kdeps /bin/kdeps")
	assert.Contains(t, dockerfile, "RUN chmod a+x /bin/kdeps")

	// Test case: With API server mode
	dockerfile = generateDockerfile(
		"latest",
		"1.0.0",
		"localhost",
		"8080",
		"http://localhost:8080",
		"",
		"",
		"",
		"",
		"",
		"2023.09",
		"1.0.0",
		"UTC",
		"8080",
		false,
		false,
		true,
		false,
	)

	assert.Contains(t, dockerfile, "EXPOSE 8080")
	assert.Contains(t, dockerfile, "ENTRYPOINT [\"/bin/kdeps\"]")
}

func TestBuildDockerfile(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	// Create test directories
	kdepsDir := "/test/kdeps"
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Create a mock workflow file with proper schema reference
	workflowContent := `
amends "package://schema.kdeps.com/core@1.0.0#/Workflow.pkl"

name = "test-agent"
version = "1.0.0"
description = "Test agent"
targetActionID = "test-action"
settings {
  APIServerMode = true
  APIServer {
    portNum = 3000
    hostIP = "127.0.0.1"
  }
  WebServerMode = true
  WebServer {
    portNum = 8080
    hostIP = "127.0.0.1"
  }
  agentSettings {
    packages {
      "test-package"
    }
    repositories {
      "ppa:test/repo"
    }
    pythonPackages {
      "test-python-package"
    }
    condaPackages {
      "base" {
        "conda-forge" = "test-conda-package"
      }
    }
    args {
      "TEST_ARG" = "value"
    }
    env {
      "TEST_ENV" = "value"
    }
    timezone = "UTC"
    installAnaconda = true
  }
}
`
	workflowPath := filepath.Join(kdepsDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowPath, []byte(workflowContent), 0o644))

	// Create a mock package project
	pkgProject := &archiver.KdepsPackage{
		Workflow: workflowPath,
	}

	// Create a mock kdeps configuration
	kdeps := &kdCfg.Kdeps{
		RunMode:   "docker",
		DockerGPU: "nvidia",
	}

	// Test case: Build Dockerfile
	_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
	// Since we're using a mock environment, we expect an error about the workflow file
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error reading workflow file")
}
