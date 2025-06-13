package docker

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestParseOLLAMAHostExtra(t *testing.T) {
	logger := logging.NewTestLogger()
	orig := os.Getenv("OLLAMA_HOST")
	t.Cleanup(func() { _ = os.Setenv("OLLAMA_HOST", orig) })

	t.Run("env-not-set", func(t *testing.T) {
		_ = os.Unsetenv("OLLAMA_HOST")
		host, port, err := parseOLLAMAHost(logger)
		assert.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("invalid-format", func(t *testing.T) {
		_ = os.Setenv("OLLAMA_HOST", "invalid") // missing ':'
		host, port, err := parseOLLAMAHost(logger)
		assert.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("happy-path", func(t *testing.T) {
		_ = os.Setenv("OLLAMA_HOST", "127.0.0.1:11435")
		host, port, err := parseOLLAMAHost(logger)
		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1", host)
		assert.Equal(t, "11435", port)
	})
}

func TestGenerateUniqueOllamaPortRange(t *testing.T) {
	existing := uint16(12000)
	count := 20 // sample multiple generations to reduce flake risk
	for i := 0; i < count; i++ {
		portStr := generateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(portStr)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, port, minPort)
		assert.LessOrEqual(t, port, maxPort)
		assert.NotEqual(t, int(existing), port)
	}
}

func TestLoadEnvFile_InvalidContent(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// invalid line (missing '=')
	_ = afero.WriteFile(fs, envPath, []byte("INVALID"), 0o644)

	envSlice, err := loadEnvFile(fs, envPath)
	assert.NoError(t, err)
	// godotenv treats 'INVALID' as key "" with value "INVALID", leading to "=INVALID" entry.
	assert.Equal(t, []string{"=INVALID"}, envSlice)
}
