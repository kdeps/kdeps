package docker

import (
	"context"
	"errors"
	"net"
	"time"

	"github.com/kdeps/kdeps/pkg/kdepsexec"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
)

// isServerReady checks if ollama server is ready by attempting to connect to the specified host and port.
func isServerReady(host string, port string, logger *logging.Logger) bool {
	logger.Debug(messages.MsgServerCheckingReady, "host", host, "port", port)

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		logger.Warn(messages.MsgServerNotReady, "error", err)
		return false
	}
	conn.Close()

	return true
}

// waitForServer waits until ollama server is ready by polling the specified host and port.
func waitForServer(host string, port string, timeout time.Duration, logger *logging.Logger) error {
	logger.Debug(messages.MsgServerWaitingReady)

	start := time.Now()
	for {
		if isServerReady(host, port, logger) {
			logger.Debug(messages.MsgServerReady, "host", host, "port", port)
			return nil
		}

		if time.Since(start) > timeout {
			logger.Error(messages.MsgServerTimeout, "host", host, "port", port)
			return errors.New("timeout waiting for ollama server to be ready")
		}

		logger.Debug(messages.MsgServerRetrying)
		time.Sleep(time.Second) // Sleep before the next check
	}
}

// startOllamaServer starts the ollama server in the background using go-execute.
// Errors are logged in the background, and the function returns immediately.
func startOllamaServer(ctx context.Context, logger *logging.Logger) {
	logger.Debug(messages.MsgStartOllamaBackground)

	_, _, _, err := kdepsexec.KdepsExec(ctx, "ollama", []string{"serve"}, "", false, true, logger)
	if err != nil {
		logger.Error(messages.MsgStartOllamaFailed, "error", err)
	}

	logger.Debug(messages.MsgOllamaStartedBackground)
}
