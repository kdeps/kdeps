package kdepsexec_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"

	execute "github.com/alexellis/go-execute/v2"
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

func TestRunExecTask_Foreground(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	task := execute.ExecTask{
		Command:     "echo",
		Args:        []string{"hello"},
		StreamStdio: false,
	}

	stdout, stderr, exitCode, err := RunExecTask(ctx, task, logger, false)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

func TestRunExecTask_ShellMode(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	task := execute.ExecTask{
		Command: "echo shell-test",
		Shell:   true,
	}

	stdout, _, _, err := RunExecTask(ctx, task, logger, false)
	assert.NoError(t, err)
	assert.Equal(t, "shell-test\n", stdout)
}

func TestRunExecTask_Background(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	task := execute.ExecTask{
		Command: "sleep",
		Args:    []string{"1"},
	}

	stdout, stderr, exitCode, err := RunExecTask(ctx, task, logger, true)
	// Background mode should return immediately with zero exit code and no output
	assert.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

// Additional edge case tests to increase coverage

func TestKdepsExec_EnvFileReadError(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "kdeps-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a .env file with no read permissions
	envFile := filepath.Join(tempDir, ".env")
	err = os.WriteFile(envFile, []byte("TEST_VAR=test_value"), 0o000)
	assert.NoError(t, err)
	defer os.Chmod(envFile, 0o644) // Restore permissions for cleanup

	// Should warn but not fail
	stdout, stderr, exitCode, err := KdepsExec(ctx, "echo", []string{"hello"}, tempDir, true, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

func TestKdepsExec_ComplexEnvFile(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "kdeps-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create .env file with comments, empty lines, and multiple variables
	envContent := `# This is a comment
TEST_VAR1=value1

# Another comment
TEST_VAR2=value2

TEST_VAR3=value3
`
	envFile := filepath.Join(tempDir, ".env")
	err = os.WriteFile(envFile, []byte(envContent), 0o644)
	assert.NoError(t, err)

	stdout, stderr, exitCode, err := KdepsExec(ctx, "sh", []string{"-c", "echo $TEST_VAR1,$TEST_VAR2,$TEST_VAR3"}, tempDir, true, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, "value1,value2,value3\n", stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

func TestKdepsExec_EnvFileNoExist(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "kdeps-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// No .env file exists - should proceed normally
	stdout, stderr, exitCode, err := KdepsExec(ctx, "echo", []string{"hello"}, tempDir, true, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

func TestKdepsExec_UseEnvFileWithEmptyWorkingDir(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	// useEnvFile is true but workingDir is empty - should skip .env loading
	stdout, stderr, exitCode, err := KdepsExec(ctx, "echo", []string{"hello"}, "", true, false, logger)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", stdout)
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}

func TestKdepsExec_CommandNotFound(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	// Try to execute a non-existent command
	_, _, _, err := KdepsExec(ctx, "this-command-does-not-exist-123456", []string{}, "", false, false, logger)
	assert.Error(t, err)
}

func TestKdepsExec_ContextCancellation(t *testing.T) {
	logger := logging.GetLogger()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Command should fail due to cancelled context
	_, _, _, err := KdepsExec(ctx, "sleep", []string{"10"}, "", false, false, logger)
	assert.Error(t, err)
}

func TestKdepsExec_WithStderr(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	// Command that outputs to stderr
	stdout, stderr, exitCode, err := KdepsExec(ctx, "sh", []string{"-c", "echo 'error message' >&2; exit 1"}, "", false, false, logger)
	assert.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "error message")
	assert.Equal(t, 1, exitCode)
}

func TestRunExecTask_WithWorkingDir(t *testing.T) {
	logger := logging.GetLogger()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "kdeps-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file in temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	assert.NoError(t, err)

	task := execute.ExecTask{
		Command: "ls",
		Args:    []string{"test.txt"},
		Cwd:     tempDir,
	}

	stdout, stderr, exitCode, err := RunExecTask(ctx, task, logger, false)
	assert.NoError(t, err)
	assert.Contains(t, stdout, "test.txt")
	assert.Empty(t, stderr)
	assert.Equal(t, 0, exitCode)
}
