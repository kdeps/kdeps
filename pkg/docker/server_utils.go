package docker

import (
	"context"
	"errors"
	"net"
	"time"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
)

// isServerReady checks if ollama server is ready by attempting to connect to the specified host and port.
func isServerReady(host string, port string, logger *logging.Logger) bool {
	logger.Debug("checking if ollama server is ready", "host", host, "port", port)

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		logger.Warn("ollama server not ready", "error", err)
		return false
	}
	conn.Close()

	return true
}

// waitForServer waits until ollama server is ready by polling the specified host and port.
func waitForServer(host string, port string, timeout time.Duration, logger *logging.Logger) error {
	logger.Debug("waiting for ollama server to be ready...")

	start := time.Now()
	for {
		if isServerReady(host, port, logger) {
			logger.Debug("ollama server is ready", "host", host, "port", port)
			return nil
		}

		if time.Since(start) > timeout {
			logger.Error("timeout waiting for ollama server to be ready.", "host", host, "port", port)
			return errors.New("Timeout waiting for ollama server to be ready")
		}

		logger.Debug("server not yet ready. Retrying...")
		time.Sleep(time.Second) // Sleep before the next check
	}
}

// startOllamaServer starts the ollama server command in the background using go-execute.
func startOllamaServer(ctx context.Context, logger *logging.Logger) error {
	logger.Debug("starting ollama server in the background...")

	// Run ollama server in a background goroutine using go-execute
	cmd := execute.ExecTask{
		Command:     "ollama",
		Args:        []string{"serve"},
		StreamStdio: true,
	}

	// Start the command asynchronously
	go func() {
		_, err := cmd.Execute(ctx)
		if err != nil {
			logger.Error("error starting ollama server: ", err)
		} else {
			logger.Debug("ollama server exited.")
		}
	}()

	logger.Debug("ollama server started in the background.")
	return nil
}
