package docker

import (
	"context"
	"errors"
	"net"
	"time"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/kdeps/kdeps/pkg/logging"
)

// isServerReady checks if ollama server is ready by attempting to connect to the specified host and port
func isServerReady(host string, port string, logger *logging.Logger) bool {
	logger.Debug("Checking if ollama server is ready", "host", host, "port", port)

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		logger.Warn("Ollama server not ready", "error", err)
		return false
	}
	conn.Close()

	return true
}

// waitForServer waits until ollama server is ready by polling the specified host and port
func waitForServer(host string, port string, timeout time.Duration, logger *logging.Logger) error {
	logger.Debug("Waiting for ollama server to be ready...")

	start := time.Now()
	for {
		if isServerReady(host, port, logger) {
			logger.Debug("Ollama server is ready", "host", host, "port", port)
			return nil
		}

		if time.Since(start) > timeout {
			logger.Error("Timeout waiting for ollama server to be ready.", "host", host, "port", port)
			return errors.New("Timeout waiting for ollama server to be ready")
		}

		logger.Debug("Server not yet ready. Retrying...")
		time.Sleep(time.Second) // Sleep before the next check
	}
}

// startOllamaServer starts the ollama server command in the background using go-execute
func startOllamaServer(logger *logging.Logger) error {
	logger.Debug("Starting ollama server in the background...")

	// Run ollama server in a background goroutine using go-execute
	cmd := execute.ExecTask{
		Command:     "ollama",
		Args:        []string{"serve"},
		StreamStdio: true,
	}

	// Start the command asynchronously
	go func() {
		_, err := cmd.Execute(context.Background())
		if err != nil {
			logger.Error("Error starting ollama server: ", err)
		} else {
			logger.Debug("Ollama server exited.")
		}
	}()

	logger.Debug("Ollama server started in the background.")
	return nil
}
