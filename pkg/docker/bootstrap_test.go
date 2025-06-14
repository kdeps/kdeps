package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestBootstrapDockerSystem(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(actionDir, 0o755)
	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		ActionDir: actionDir,
		Environment: &environment.Environment{
			DockerMode: "1",
		},
	}

	t.Run("NonDockerMode", func(t *testing.T) {
		dr.Environment.DockerMode = "0"
		apiServerMode, err := BootstrapDockerSystem(ctx, dr)
		assert.NoError(t, err)
		assert.False(t, apiServerMode)
	})

	t.Run("DockerMode", func(t *testing.T) {
		dr.Environment.DockerMode = "1"
		apiServerMode, err := BootstrapDockerSystem(ctx, dr)
		assert.Error(t, err) // Expected error due to missing OLLAMA_HOST
		assert.False(t, apiServerMode)
	})
}

func TestCreateFlagFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		err := CreateFlagFile(fs, ctx, "/tmp/flag")
		assert.NoError(t, err)
		exists, _ := afero.Exists(fs, "/tmp/flag")
		assert.True(t, exists)
	})

	t.Run("FileExists", func(t *testing.T) {
		_ = afero.WriteFile(fs, "/tmp/existing", []byte(""), 0o644)
		err := CreateFlagFile(fs, ctx, "/tmp/existing")
		assert.NoError(t, err)
	})
}

func TestPullModels(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("EmptyModels", func(t *testing.T) {
		err := pullModels(ctx, []string{}, logger)
		assert.NoError(t, err)
	})

	t.Run("ModelPull", func(t *testing.T) {
		// This test requires a running OLLAMA service and may not be suitable for all environments
		// Consider mocking the KdepsExec function for more reliable testing
		t.Skip("Skipping test that requires OLLAMA service")
	})
}

func TestStartAPIServer(t *testing.T) {
	ctx := context.Background()
	dr := &resolver.DependencyResolver{
		Logger: logging.NewTestLogger(),
	}

	t.Run("StartAPIServer", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the StartAPIServerMode function for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
		_ = ctx // Use context to avoid linter error
		_ = dr  // Use dr to avoid linter error
	})
}

func TestStartWebServer(t *testing.T) {
	ctx := context.Background()
	dr := &resolver.DependencyResolver{
		Logger: logging.NewTestLogger(),
	}

	t.Run("StartWebServer", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the StartWebServerMode function for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
		_ = ctx // Use context to avoid linter error
		_ = dr  // Use dr to avoid linter error
	})
}
