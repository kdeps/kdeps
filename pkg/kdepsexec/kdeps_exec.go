package kdepsexec

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
		// Run the command asynchronously with timeout enforcement
		go func() {
			// Create a new context that respects the original context's timeout
			backgroundCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			// Check if original context has deadline and apply it
			if deadline, ok := ctx.Deadline(); ok {
				backgroundCtx, cancel = context.WithDeadline(context.Background(), deadline)
				defer cancel()
			}

			result, err := task.Execute(backgroundCtx)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					logger.Error("background command timed out", "command", command, "error", err)
				} else {
					logger.Error("background command failed", "command", command, "error", err)
				}
				return
			}

			// Log output from background command
			if result.Stdout != "" {
				logger.Info("background command stdout", "command", command, "output", result.Stdout)
			}
			if result.Stderr != "" {
				logger.Warn("background command stderr", "command", command, "output", result.Stderr)
			}

			if result.ExitCode != 0 {
				logger.Error("background command exited with non-zero code", "command", command, "code", result.ExitCode)
			} else {
				logger.Info("background command completed successfully", "command", command, "code", result.ExitCode)
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

// RunExecTask executes a given execute.ExecTask using the same semantics as KdepsExec.
// It allows existing code that already constructs ExecTask structs to delegate the execution here,
// satisfying the project rule that all execution flows through the kdepsexec package.
func RunExecTask(ctx context.Context, task execute.ExecTask, logger *logging.Logger, background bool) (string, string, int, error) {
	// Map fields to KdepsExec call.
	workingDir := task.Cwd
	command := task.Command
	args := task.Args

	// If Shell flag is true, execute via "sh -c <command>" for portability.
	if task.Shell {
		args = []string{"-c", command}
		command = "sh"
	}

	stdout, stderr, exitCode, err := KdepsExec(ctx, command, args, workingDir, false, background, logger)
	return stdout, stderr, exitCode, err
}
