package kdepsexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestKdepsExec(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	t.Run("SimpleCommand", func(t *testing.T) {
		stdout, stderr, exitCode, err := KdepsExec(ctx, "echo", []string{"hello"}, "", false, false, logger)
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", stdout)
		assert.Empty(t, stderr)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("WithEnvFile", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "kdeps-test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		envFile := filepath.Join(tempDir, ".env")
		err = os.WriteFile(envFile, []byte("TEST_VAR=test_value"), 0o644)
		assert.NoError(t, err)

		stdout, stderr, exitCode, err := KdepsExec(ctx, "sh", []string{"-c", "echo $TEST_VAR"}, tempDir, true, false, logger)
		assert.NoError(t, err)
		assert.Equal(t, "test_value\n", stdout)
		assert.Empty(t, stderr)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("BackgroundCommand", func(t *testing.T) {
		stdout, stderr, exitCode, err := KdepsExec(ctx, "sleep", []string{"1"}, "", false, true, logger)
		assert.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("NonZeroExitCode", func(t *testing.T) {
		stdout, stderr, exitCode, err := KdepsExec(ctx, "false", []string{}, "", false, false, logger)
		assert.Error(t, err)
		assert.Empty(t, stdout)
		assert.Empty(t, stderr)
		assert.NotEqual(t, 0, exitCode)
	})
}
