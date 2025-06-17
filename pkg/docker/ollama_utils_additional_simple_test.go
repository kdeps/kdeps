package docker

import (
	"os"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
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
