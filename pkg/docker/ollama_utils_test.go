package docker

import (
	"os"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestParseOLLAMAHost(t *testing.T) {
	logger := logging.GetLogger()

	t.Run("missing env var", func(t *testing.T) {
		os.Unsetenv("OLLAMA_HOST")
		host, port, err := parseOLLAMAHost(logger)
		assert.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("invalid format", func(t *testing.T) {
		os.Setenv("OLLAMA_HOST", "notaport")
		host, port, err := parseOLLAMAHost(logger)
		assert.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("valid host:port", func(t *testing.T) {
		os.Setenv("OLLAMA_HOST", "127.0.0.1:12345")
		host, port, err := parseOLLAMAHost(logger)
		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1", host)
		assert.Equal(t, "12345", port)
	})
}

func TestGenerateUniqueOllamaPort(t *testing.T) {
	existingPort := uint16(12345)
	for i := 0; i < 10; i++ {
		portStr := generateUniqueOllamaPort(existingPort)
		port, err := strconv.Atoi(portStr)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, port, minPort)
		assert.LessOrEqual(t, port, maxPort)
		assert.NotEqual(t, int(existingPort), port)
	}

	t.Run("should not return the existing port", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			portStr := generateUniqueOllamaPort(existingPort)
			assert.NotEqual(t, strconv.Itoa(int(existingPort)), portStr)
		}
	})
}
