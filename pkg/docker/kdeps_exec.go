package docker

import (
	"context"
	"fmt"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/charmbracelet/log"
)

// KdepsExec executes a command and returns stdout, stderr, and the exit code using go-execute
func KdepsExec(command string, args []string, logger *log.Logger) (string, string, int, error) {
	// Log the command being executed
	logger.Debug("Executing", "command", command, "args", args)

	// Create the command task using go-execute
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
	}

	// Execute the command
	res, err := cmd.Execute(context.Background())
	if err != nil {
		logger.Error("Command execution failed", "error", err)
		return res.Stdout, res.Stderr, res.ExitCode, err
	}

	// Check for non-zero exit code
	if res.ExitCode != 0 {
		logger.Warn("Non-zero exit code", "exit code", res.ExitCode, " Stderr: ", res.Stderr)
		return res.Stdout, res.Stderr, res.ExitCode, fmt.Errorf("non-zero exit code: %s", res.Stderr)
	}

	logger.Debug("Command executed successfully: ", "command: ", command, " with exit code: ", res.ExitCode)
	return res.Stdout, res.Stderr, res.ExitCode, nil
}
