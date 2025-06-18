//go:build test
// +build test

package kdepsexec

import (
	"context"

	"github.com/kdeps/kdeps/pkg/logging"
)

// KdepsExec is a test stub that bypasses actual command execution.
func KdepsExec(ctx context.Context, command string, args []string, workingDir string, useEnvFile, background bool, logger *logging.Logger) (string, string, int, error) {
	// Record debug message to demonstrate invocation during tests.
	logger.Debug("stub KdepsExec invoked", "cmd", command)
	return "", "", 0, nil
}
