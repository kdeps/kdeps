package docker

import (
	"context"
	"errors"
	"kdeps/pkg/logging"
	"net"
	"time"

	execute "github.com/alexellis/go-execute/v2"
)

// isServerReady checks if ollama server is ready by attempting to connect to the specified host and port
func isServerReady(host string, port string) bool {
	logging.Info("Checking if ollama server is ready at ", host, ":", port)

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		logging.Warn("Ollama server not ready: ", err)
		return false
	}
	conn.Close()

	logging.Info("Ollama server is ready at ", host, ":", port)
	return true
}

// waitForServer waits until ollama server is ready by polling the specified host and port
func waitForServer(host string, port string, timeout time.Duration) error {
	logging.Info("Waiting for ollama server to be ready...")

	start := time.Now()
	for {
		if isServerReady(host, port) {
			logging.Info("Ollama server is ready at ", host, ":", port)
			return nil
		}

		if time.Since(start) > timeout {
			logging.Error("Timeout waiting for ollama server to be ready. Host: ", host, " Port: ", port)
			return errors.New("Timeout waiting for ollama server to be ready")
		}

		logging.Info("Server not yet ready. Retrying...")
		time.Sleep(time.Second) // Sleep before the next check
	}
}

// startOllamaServer starts the ollama server command in the background using go-execute
func startOllamaServer() error {
	logging.Info("Starting ollama server in the background...")

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
			logging.Error("Error starting ollama server: ", err)
		} else {
			logging.Info("Ollama server exited.")
		}
	}()

	logging.Info("Ollama server started in the background.")
	return nil
}
