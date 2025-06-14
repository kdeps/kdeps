package docker

import (
    "context"

    "github.com/kdeps/kdeps/pkg/kdepsexec"
    "github.com/kdeps/kdeps/pkg/logging"
)

// KdepsExec is kept for backward compatibility; it forwards to kdepsexec.KdepsExec.
func KdepsExec(ctx context.Context, command string, args []string, workingDir string, useEnvFile bool, background bool, logger *logging.Logger) (string, string, int, error) {
    return kdepsexec.KdepsExec(ctx, command, args, workingDir, useEnvFile, background, logger)
} 