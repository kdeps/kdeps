package docker

import (
	"bytes"
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

func setupTestImage(t *testing.T) (afero.Fs, *logging.Logger, *archiver.KdepsPackage) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	pkgProject := &archiver.KdepsPackage{
		Workflow: "test-workflow",
	}
	return fs, logger, pkgProject
}

func TestCheckDevBuildMode(t *testing.T) {
	fs, logger, _ := setupTestImage(t)
	kdepsDir := "/test"

	t.Run("FileDoesNotExist", func(t *testing.T) {
		isDev, err := checkDevBuildMode(fs, kdepsDir, logger)
		assert.NoError(t, err)
		assert.False(t, isDev)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Create the directory and file
		err := fs.MkdirAll(filepath.Join(kdepsDir, "cache"), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(kdepsDir, "cache", "kdeps"), []byte("test"), 0o644)
		require.NoError(t, err)

		isDev, err := checkDevBuildMode(fs, kdepsDir, logger)
		assert.NoError(t, err)
		assert.True(t, isDev)
	})
}

func TestGenerateParamsSection(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		result := generateParamsSection("TEST", nil)
		assert.Empty(t, result)
	})

	t.Run("WithParams", func(t *testing.T) {
		params := map[string]string{
			"param1": "value1",
			"param2": "value2",
		}
		result := generateParamsSection("TEST", params)
		assert.Contains(t, result, "TEST param1=\"value1\"")
		assert.Contains(t, result, "TEST param2=\"value2\"")
	})
}

func TestGenerateDockerfile(t *testing.T) {
	t.Run("BasicGeneration", func(t *testing.T) {
		result := generateDockerfile(
			"latest",
			"1.0",
			"localhost",
			"8080",
			"http://localhost:8080",
			"",
			"ENV TEST=test",
			"",
			"",
			"",
			"",
			"",
			"UTC",
			"8080",
			false,
			false,
			false,
			false,
		)
		assert.Contains(t, result, "FROM ollama/ollama:latest")
		assert.Contains(t, result, "ENV SCHEMA_VERSION=1.0")
		assert.Contains(t, result, "ENV OLLAMA_HOST=localhost:8080")
		assert.Contains(t, result, "ENV KDEPS_HOST=http://localhost:8080")
		assert.Contains(t, result, "ENV TEST=test")
	})

	t.Run("WithAnaconda", func(t *testing.T) {
		result := generateDockerfile(
			"latest",
			"1.0",
			"localhost",
			"8080",
			"http://localhost:8080",
			"",
			"",
			"",
			"",
			"",
			"2023.09",
			"",
			"UTC",
			"8080",
			true,
			false,
			false,
			false,
		)
		assert.Contains(t, result, "RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh")
	})
}

func TestCopyFilesToRunDir(t *testing.T) {
	fs, logger, _ := setupTestImage(t)
	ctx := context.Background()
	downloadDir := "/download"
	runDir := "/run"

	t.Run("NoFiles", func(t *testing.T) {
		err := copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
		assert.Error(t, err)
	})

	t.Run("WithFiles", func(t *testing.T) {
		// Create test files
		err := fs.MkdirAll(downloadDir, 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(downloadDir, "test.txt"), []byte("test"), 0o644)
		require.NoError(t, err)

		err = copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
		assert.NoError(t, err)

		// Verify file was copied
		exists, err := afero.Exists(fs, filepath.Join(runDir, "cache", "test.txt"))
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestPrintDockerBuildOutput(t *testing.T) {
	t.Run("ValidOutput", func(t *testing.T) {
		output := `{"stream": "Step 1/10 : FROM base\n"}
{"stream": " ---> abc123\n"}
{"stream": "Step 2/10 : RUN command\n"}
{"stream": " ---> def456\n"}`
		err := printDockerBuildOutput(bytes.NewReader([]byte(output)))
		assert.NoError(t, err)
	})

	t.Run("ErrorOutput", func(t *testing.T) {
		output := `{"error": "Build failed"}`
		err := printDockerBuildOutput(bytes.NewReader([]byte(output)))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Build failed")
	})
}

func TestBuildDockerfile(t *testing.T) {
	fs, logger, pkgProject := setupTestImage(t)
	ctx := context.Background()
	kdepsDir := "/test"

	t.Run("MissingConfig", func(t *testing.T) {
		kdeps := &kdCfg.Kdeps{}
		_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidConfig", func(t *testing.T) {
		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
		}
		_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})
}

func TestBuildDockerImage(t *testing.T) {
	fs, logger, pkgProject := setupTestImage(t)
	ctx := context.Background()
	runDir := "/run"
	kdepsDir := "/test"

	// Create a mock Docker client
	mockClient := &client.Client{}

	t.Run("MissingWorkflow", func(t *testing.T) {
		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
		}
		_, _, err := BuildDockerImage(fs, ctx, kdeps, mockClient, runDir, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidWorkflow", func(t *testing.T) {
		// Create workflow file
		err := fs.MkdirAll(filepath.Join(kdepsDir, "workflows"), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(kdepsDir, "workflows", "test-workflow.yaml"), []byte(`
name: test-workflow
version: 1.0
`), 0o644)
		require.NoError(t, err)

		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
		}
		_, _, err = BuildDockerImage(fs, ctx, kdeps, mockClient, runDir, kdepsDir, pkgProject, logger)
		assert.Error(t, err) // Expected error due to mock client
	})
}
