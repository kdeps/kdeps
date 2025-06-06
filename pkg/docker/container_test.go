package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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
