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

// IsServerReady checks if a server is ready on the specified host and port.
func IsServerReady(host string, port string, logger *logging.Logger) bool {
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

// WaitForServer waits for a server to be ready on the specified host and port.
func WaitForServer(host string, port string, timeout time.Duration, logger *logging.Logger) error {
	logger.Debug(messages.MsgServerWaitingReady)

	start := time.Now()
	for {
		if IsServerReady(host, port, logger) {
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

// StartOllamaServer starts the Ollama server in the background.
func StartOllamaServer(ctx context.Context, logger *logging.Logger) {
	logger.Debug(messages.MsgStartOllamaBackground)

	_, _, _, err := kdepsexec.KdepsExec(ctx, "ollama", []string{"serve"}, "", false, true, logger)
	if err != nil {
		logger.Error(messages.MsgStartOllamaFailed, "error", err)
	}

	logger.Debug(messages.MsgOllamaStartedBackground)
}
