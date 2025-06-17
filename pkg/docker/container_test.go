package docker

import (
	"context"
	"testing"

	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("FileExists", func(t *testing.T) {
		_ = afero.WriteFile(fs, ".env", []byte("KEY1=value1\nKEY2=value2"), 0o644)
		envSlice, err := loadEnvFile(fs, ".env")
		assert.NoError(t, err)
		assert.Len(t, envSlice, 2)
		assert.Contains(t, envSlice, "KEY1=value1")
		assert.Contains(t, envSlice, "KEY2=value2")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		envSlice, err := loadEnvFile(fs, "nonexistent.env")
		assert.NoError(t, err)
		assert.Nil(t, envSlice)
	})
}

func TestGenerateDockerCompose(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("CPU", func(t *testing.T) {
		err := GenerateDockerCompose(fs, "test", "image", "test-cpu", "127.0.0.1", "8080", "", "", true, false, "cpu")
		assert.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-cpu.yaml")
		assert.Contains(t, string(content), "test-cpu:")
		assert.Contains(t, string(content), "image: image")
	})

	t.Run("NVIDIA", func(t *testing.T) {
		err := GenerateDockerCompose(fs, "test", "image", "test-nvidia", "127.0.0.1", "8080", "", "", true, false, "nvidia")
		assert.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-nvidia.yaml")
		assert.Contains(t, string(content), "driver: nvidia")
	})

	t.Run("AMD", func(t *testing.T) {
		err := GenerateDockerCompose(fs, "test", "image", "test-amd", "127.0.0.1", "8080", "", "", true, false, "amd")
		assert.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-amd.yaml")
		assert.Contains(t, string(content), "/dev/kfd")
		assert.Contains(t, string(content), "/dev/dri")
	})

	t.Run("UnsupportedGPU", func(t *testing.T) {
		err := GenerateDockerCompose(fs, "test", "image", "test-unsupported", "127.0.0.1", "8080", "", "", true, false, "unsupported")
		assert.Error(t, err)
	})
}

func TestCreateDockerContainer(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	assert.NoError(t, err)

	t.Run("APIModeWithoutPort", func(t *testing.T) {
		_, err := CreateDockerContainer(fs, ctx, "test", "image", "127.0.0.1", "", "", "", "cpu", true, false, cli)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "portNum must be non-empty")
	})

	t.Run("WebModeWithoutPort", func(t *testing.T) {
		_, err := CreateDockerContainer(fs, ctx, "test", "image", "127.0.0.1", "8080", "127.0.0.1", "", "cpu", false, true, cli)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webPortNum must be non-empty")
	})

	t.Run("ContainerExists", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the Docker client for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
	})
}

func TestLoadEnvFile_VariousCases(t *testing.T) {
	fs := afero.NewOsFs()
	dir := t.TempDir()

	t.Run("file-missing", func(t *testing.T) {
		envs, err := loadEnvFile(fs, filepath.Join(dir, "missing.env"))
		require.NoError(t, err)
		require.Nil(t, envs)
	})

	t.Run("valid-env", func(t *testing.T) {
		path := filepath.Join(dir, "good.env")
		content := "FOO=bar\nHELLO=world\n"
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o644))

		envs, err := loadEnvFile(fs, path)
		require.NoError(t, err)
		// Convert slice to joined string for easier contains checks irrespective of order.
		joined := strings.Join(envs, ",")
		require.Contains(t, joined, "FOO=bar")
		require.Contains(t, joined, "HELLO=world")
	})
}

func TestLoadEnvFileMissingAndSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	// Case 1: file missing returns nil slice, no error
	envs, err := loadEnvFile(fs, "/tmp/not_existing.env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs != nil {
		t.Fatalf("expected nil slice for missing file, got %v", envs)
	}

	// Case 2: valid .env file parsed
	tmpDir, _ := afero.TempDir(fs, "", "env")
	fname := tmpDir + "/.env"
	content := "FOO=bar\nHELLO=world"
	_ = afero.WriteFile(fs, fname, []byte(content), 0o644)

	envs, err = loadEnvFile(fs, fname)
	if err != nil {
		t.Fatalf("loadEnvFile error: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(envs))
	}
	joined := strings.Join(envs, ",")
	if !strings.Contains(joined, "FOO=bar") || !strings.Contains(joined, "HELLO=world") {
		t.Fatalf("parsed env slice missing values: %v", envs)
	}
}

func TestGenerateDockerComposeCPU(t *testing.T) {
	fs := afero.NewOsFs()
	err := GenerateDockerCompose(fs, "agent", "image:tag", "agent-cpu", "127.0.0.1", "5000", "", "", true, false, "cpu")
	if err != nil {
		t.Fatalf("GenerateDockerCompose error: %v", err)
	}
	expected := "agent_docker-compose-cpu.yaml"
	exists, _ := afero.Exists(fs, expected)
	if !exists {
		t.Fatalf("expected compose file %s", expected)
	}
}
