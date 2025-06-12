package kdepsexec

import (
	"context"
	"testing"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestRunExecTask_Foreground(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
