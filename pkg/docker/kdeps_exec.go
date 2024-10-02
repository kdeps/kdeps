package docker

import (
	"context"
	"fmt"
	"kdeps/pkg/logging"

	execute "github.com/alexellis/go-execute/v2"
)

// KdepsExec executes a command and returns stdout, stderr, and the exit code using go-execute
func KdepsExec(command string, args []string) (string, string, int, error) {
	// Log the command being executed
	logging.Info("Executing command: ", command, " with args: ", args)

	// Create the command task using go-execute
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
	}

	// Execute the command
	res, err := cmd.Execute(context.Background())
	if err != nil {
		logging.Error("Command execution failed: ", err)
		return res.Stdout, res.Stderr, res.ExitCode, err
	}

	// Check for non-zero exit code
	if res.ExitCode != 0 {
		logging.Warn("Non-zero exit code: ", res.ExitCode, " Stderr: ", res.Stderr)
		return res.Stdout, res.Stderr, res.ExitCode, fmt.Errorf("non-zero exit code: %s", res.Stderr)
	}

	logging.Info("Command executed successfully: ", "command: ", command, " with exit code: ", res.ExitCode)
	return res.Stdout, res.Stderr, res.ExitCode, nil
}
