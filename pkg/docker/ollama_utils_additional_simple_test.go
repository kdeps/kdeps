package docker

import (
	"os"
	"strconv"
	"testing"

	crand "crypto/rand"
	"path/filepath"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestParseOLLAMAHostAdditional(t *testing.T) {
	logger := logging.NewTestLogger()

	// Case 1: variable not set
	os.Unsetenv("OLLAMA_HOST")
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when OLLAMA_HOST is unset")
	}

	// Case 2: invalid format
	os.Setenv("OLLAMA_HOST", "invalid-format")
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Case 3: valid host:port
	os.Setenv("OLLAMA_HOST", "127.0.0.1:11434")
	host, port, err := parseOLLAMAHost(logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "127.0.0.1" || port != "11434" {
		t.Fatalf("unexpected parse result: %s %s", host, port)
	}
}

func TestGenerateUniqueOllamaPortAdditional(t *testing.T) {
	existing := uint16(11434)
	for i := 0; i < 100; i++ {
		portStr := generateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			t.Fatalf("invalid port returned: %v", err)
		}
		if port < minPort || port > maxPort {
			t.Fatalf("port out of range: %d", port)
		}
		if port == int(existing) {
			t.Fatalf("generated port equals existing port")
		}
	}
}

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

// stubReader allows us to control the bytes returned by crypto/rand.Reader so we can
// force generateUniqueOllamaPort to hit its collision branch once.
// It will return all-zero bytes on the first call, then all-0xFF bytes afterwards.
// This causes the first generated port to equal minPort and collide with existingPort,
// ensuring the loop executes at least twice.

type stubReader struct{ call int }

func (s *stubReader) Read(p []byte) (int, error) {
	// crypto/rand.Int reads len(m.Bytes()) bytes (here 2). Provide deterministic data:
	// First call -> 0x0000 to generate num=0 (collision). Second call -> 0x0002 to generate num=2 (unique).
	val := byte(0x00)
	if s.call > 0 {
		val = 0x02
	}
	for i := range p {
		p[i] = val
	}
	s.call++
	return len(p), nil
}

func TestGenerateUniqueOllamaPort_CollisionLoop(t *testing.T) {
	// Swap out crypto/rand.Reader with our stub and restore afterwards.
	orig := crand.Reader
	crand.Reader = &stubReader{}
	t.Cleanup(func() { crand.Reader = orig })

	// existingPort set to minPort so first generated port collides.
	existing := uint16(minPort)

	portStr := generateUniqueOllamaPort(existing)

	if portStr == "" || portStr == "11435" { // 11435 == minPort
		t.Fatalf("expected non-empty unique port different from minPort, got %s", portStr)
	}
}

func TestGenerateUniqueOllamaPortDiffersFromExisting(t *testing.T) {
	existing := uint16(12345)
	for i := 0; i < 50; i++ {
		pStr := generateUniqueOllamaPort(existing)
		if pStr == "" {
			t.Fatalf("empty port returned")
		}
		if pStr == "12345" {
			t.Fatalf("generated same port as existing")
		}
	}
}

func TestGenerateUniqueOllamaPortWithinRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		pStr := generateUniqueOllamaPort(0)
		port, err := strconv.Atoi(pStr)
		if err != nil {
			t.Fatalf("invalid int: %v", err)
		}
		if port < minPort || port > maxPort {
			t.Fatalf("port out of range: %d", port)
		}
	}
}

func TestParseOLLAMAHost(t *testing.T) {
	logger := logging.NewTestLogger()

	// Success case
	if err := os.Setenv("OLLAMA_HOST", "127.0.0.1:8080"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	host, port, err := parseOLLAMAHost(logger)
	if err != nil || host != "127.0.0.1" || port != "8080" {
		t.Fatalf("unexpected parse result: %v %v %v", host, port, err)
	}

	// Invalid format case
	if err := os.Setenv("OLLAMA_HOST", "bad-format"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Unset env var case
	if err := os.Unsetenv("OLLAMA_HOST"); err != nil {
		t.Fatalf("failed to unset env: %v", err)
	}
	if _, _, err := parseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when env not set")
	}
}

func TestGenerateUniqueOllamaPort(t *testing.T) {
	existing := uint16(12345)
	for i := 0; i < 10; i++ {
		pStr := generateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(pStr)
		if err != nil {
			t.Fatalf("port not numeric: %v", err)
		}
		if port == int(existing) {
			t.Fatalf("generated port equals existing port")
		}
		if port < minPort || port > maxPort {
			t.Fatalf("generated port out of range: %d", port)
		}
	}
}
