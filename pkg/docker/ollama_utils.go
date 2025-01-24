package docker

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"

	"github.com/kdeps/kdeps/pkg/logging"
)

const (
	minPort = 11435
	maxPort = 65535
)

// parseOLLAMAHost parses the OLLAMA_HOST environment variable into host and port.
func parseOLLAMAHost(logger *logging.Logger) (string, string, error) {
	logger.Debug("parsing OLLAMA_HOST environment variable")

	hostEnv := os.Getenv("OLLAMA_HOST")
	if hostEnv == "" {
		logger.Error("the OLLAMA_HOST environment variable is not set")
		return "", "", errors.New("oLLAMA_HOST environment variable is not set")
	}

	host, port, err := net.SplitHostPort(hostEnv)
	if err != nil {
		logger.Error("invalid OLLAMA_HOST format; expected format 'host:port'", "error", err)
		return "", "", fmt.Errorf("invalid OLLAMA_HOST format: %w", err)
	}

	logger.Debug("parsed OLLAMA_HOST", "host", host, "port", port)
	return host, port, nil
}

// generateUniqueOllamaPort generates a random port, avoiding clashes with an existing port.
func generateUniqueOllamaPort(existingPort uint16) string {
	// Generate a random number using crypto/rand for better randomness
	for {
		// Generate a random number in the range [0, maxPort - minPort]
		// We use maxPort - minPort + 1 because the max value should be inclusive
		num, err := rand.Int(rand.Reader, big.NewInt(int64(maxPort-minPort+1)))
		if err != nil {
			// Handle error: crypto/rand failure
			panic("failed to generate a random number: " + err.Error())
		}

		// Ensure that the generated number fits within the desired port range
		ollamaPortNum := int(num.Int64()) + minPort

		// Check if the generated number is a valid port and doesn't clash with existing port
		if ollamaPortNum != int(existingPort) && ollamaPortNum <= maxPort {
			// Safely convert to uint16 after checking
			return strconv.Itoa(ollamaPortNum)
		}
	}
}
