package docker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
)

// KdepsExec executes a command with optional working directory, .env support, and background toggle.
func KdepsExec(
	ctx context.Context,
	command string,
	args []string,
	workingDir string, // Optional: pass "" to use current working dir
	useEnvFile bool, // Load .env file from workingDir
	background bool, // Run in background (true = don't wait)
	logger *logging.Logger,
) (string, string, int, error) {
	logger.Debug("executing", "command", command, "args", args, "dir", workingDir, "background", background)

	task := execute.ExecTask{
		Command:     command,
		Args:        args,
		Cwd:         workingDir,
		StreamStdio: true,
	}

	// Load .env if requested
	if useEnvFile && workingDir != "" {
		envFile := filepath.Join(workingDir, ".env")
		if _, err := os.Stat(envFile); err == nil {
			content, err := os.ReadFile(envFile)
			if err != nil {
				logger.Warn("failed to read .env", "file", envFile, "error", err)
			} else {
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" && !strings.HasPrefix(line, "#") {
						task.Env = append(task.Env, line)
					}
				}
				logger.Debug("Loaded .env file", "envFile", envFile)
			}
		}
	}

	if background {
		// Run the command asynchronously
		go func() {
			_, err := task.Execute(ctx)
			if err != nil {
				logger.Error("background command failed", "error", err)
			}
		}()
		logger.Info("background command started", "command", command)
		return "", "", 0, nil
	}

	// Run command in foreground
	result, err := task.Execute(ctx)
	if err != nil {
		logger.Error("command execution failed", "error", err)
		return result.Stdout, result.Stderr, result.ExitCode, err
	}

	if result.ExitCode != 0 {
		logger.Warn("command exited with non-zero code", "code", result.ExitCode, "stderr", result.Stderr)
		return result.Stdout, result.Stderr, result.ExitCode, errors.New("non-zero exit code")
	}

	logger.Info("command executed successfully", "code", result.ExitCode)
	return result.Stdout, result.Stderr, result.ExitCode, nil
}
