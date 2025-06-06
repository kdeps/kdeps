package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCleanupDockerBuildImages(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	assert.NoError(t, err)

	t.Run("NoContainers", func(t *testing.T) {
		err := CleanupDockerBuildImages(fs, ctx, "nonexistent", cli)
		assert.NoError(t, err)
	})

	t.Run("ContainerExists", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the Docker client for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
	})
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	environ := &environment.Environment{DockerMode: "1"}
	logger := logging.NewTestLogger() // Mock logger

	t.Run("NonDockerMode", func(t *testing.T) {
		environ.DockerMode = "0"
		Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})

	t.Run("DockerMode", func(t *testing.T) {
		environ.DockerMode = "1"
		Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})
}

func TestCleanupFlagFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger() // Mock logger

	t.Run("FilesExist", func(t *testing.T) {
		_ = afero.WriteFile(fs, "/tmp/flag1", []byte(""), 0o644)
		_ = afero.WriteFile(fs, "/tmp/flag2", []byte(""), 0o644)
		cleanupFlagFiles(fs, []string{"/tmp/flag1", "/tmp/flag2"}, logger)
		// No assertions, just ensure it doesn't panic
	})

	t.Run("FilesDoNotExist", func(t *testing.T) {
		cleanupFlagFiles(fs, []string{"/tmp/nonexistent1", "/tmp/nonexistent2"}, logger)
		// No assertions, just ensure it doesn't panic
	})
}
