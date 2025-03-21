package docker

import (
	"context"
	"fmt"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
)

// KdepsExec executes a command and returns stdout, stderr, and the exit code using go-execute.
func KdepsExec(ctx context.Context, command string, args []string, logger *logging.Logger) (string, string, int, error) {
	// Log the command being executed
	logger.Debug("executing", "command", command, "args", args)

	// Create the command task using go-execute
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
	}

	// Execute the command
	res, err := cmd.Execute(ctx)
	if err != nil {
		logger.Error("command execution failed", "error", err)
		return res.Stdout, res.Stderr, res.ExitCode, err
	}

	// Check for non-zero exit code
	if res.ExitCode != 0 {
		logger.Warn("non-zero exit code", "exit code", res.ExitCode, " Stderr: ", res.Stderr)
		return res.Stdout, res.Stderr, res.ExitCode, fmt.Errorf("non-zero exit code: %s", res.Stderr)
	}

	logger.Debug("command executed successfully: ", "command: ", command, " with exit code: ", res.ExitCode)
	return res.Stdout, res.Stderr, res.ExitCode, nil
}
