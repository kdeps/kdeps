package docker

import (
	"os"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

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
