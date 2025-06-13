package docker

import (
	"os"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestParseOLLAMAHostErrors(t *testing.T) {
	logger := logging.NewTestLogger()

	// Ensure variable unset triggers error
	os.Unsetenv("OLLAMA_HOST")
	_, _, err := parseOLLAMAHost(logger)
	if err == nil {
		t.Errorf("expected error when OLLAMA_HOST is not set")
	}

	// Invalid format
	os.Setenv("OLLAMA_HOST", "badformat")
	_, _, err = parseOLLAMAHost(logger)
	if err == nil {
		t.Errorf("expected error for invalid host format")
	}
}

func TestGenerateUniqueOllamaPortExtra(t *testing.T) {
	existing := uint16(12345)
	// Generate multiple times to ensure randomness and uniqueness vs existing.
	for i := 0; i < 20; i++ {
		portStr := generateUniqueOllamaPort(existing)
		portNum, err := strconv.Atoi(portStr)
		require.NoError(t, err)
		require.GreaterOrEqual(t, portNum, int(minPort))
		require.LessOrEqual(t, portNum, int(maxPort))
		require.NotEqual(t, int(existing), portNum)
	}
}
