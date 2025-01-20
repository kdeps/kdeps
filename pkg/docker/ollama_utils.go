package docker

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
)

// parseOLLAMAHost parses the OLLAMA_HOST environment variable into host and port.
func parseOLLAMAHost(ctx context.Context, logger *logging.Logger) (string, string, error) {
	logger.Debug("Parsing OLLAMA_HOST environment variable")

	hostEnv := os.Getenv("OLLAMA_HOST")
	if hostEnv == "" {
		logger.Error("OLLAMA_HOST environment variable is not set")
		return "", "", errors.New("OLLAMA_HOST environment variable is not set")
	}

	host, port, err := net.SplitHostPort(hostEnv)
	if err != nil {
		logger.Error("Invalid OLLAMA_HOST", "format", err)
		return "", "", fmt.Errorf("Invalid OLLAMA_HOST format: %w", err)
	}

	logger.Debug("Parsed OLLAMA_HOST", "host", host, "port", port)
	return host, port, nil
}

func generateUniqueOllamaPort(ctx context.Context, existingPort uint16) string {
	rand.Seed(time.Now().UnixNano())
	minPort, maxPort := 11435, 65535

	var ollamaPortNum uint16
	for {
		ollamaPortNum = uint16(rand.Intn(maxPort-minPort+1) + minPort)
		// If ollamaPortNum doesn't clash with the existing port, break the loop
		if ollamaPortNum != existingPort {
			break
		}
	}

	return strconv.FormatUint(uint64(ollamaPortNum), 10)
}
