package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestKdepsExec_ForegroundSuccess(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	stdout, _, code, err := KdepsExec(ctx, "echo", []string{"hello"}, "", false, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "hello")
}

func TestKdepsExec_ForegroundNonZeroExit(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	stdout, _, code, err := KdepsExec(ctx, "sh", []string{"-c", "exit 42"}, "", false, false, logger)
	assert.Error(t, err)
	assert.Equal(t, 42, code)
	assert.Empty(t, stdout)
	// stderr may or may not be empty depending on shell
}

func TestKdepsExec_EnvFile(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("FOO=bar\n"), 0o644)
	stdout, _, code, err := KdepsExec(ctx, "sh", []string{"-c", "echo $FOO"}, dir, true, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "bar")
}

func TestKdepsExec_Background(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	stdout, stderr, code, err := KdepsExec(ctx, "sleep", []string{"0.1"}, "", false, true, logger)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)
	// Give the background goroutine a moment to run
	time.Sleep(200 * time.Millisecond)
}

func TestKdepsExec_CommandNotFound(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()
	_, _, _, err := KdepsExec(ctx, "nonexistent-cmd-xyz", nil, "", false, false, logger)
	assert.Error(t, err)
}
