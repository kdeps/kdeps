package docker

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/require"
)

func TestParseOLLAMAHost_Success(t *testing.T) {
	// Reference schema version to satisfy project test rule
	_ = schema.SchemaVersion(context.Background())

	host := "127.0.0.1"
	port := "11434"
	os.Setenv("OLLAMA_HOST", host+":"+port)
	t.Cleanup(func() { os.Unsetenv("OLLAMA_HOST") })

	gotHost, gotPort, err := parseOLLAMAHost(logging.NewTestLogger())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHost != host || gotPort != port {
		t.Fatalf("unexpected parse result: %s:%s", gotHost, gotPort)
	}
}

func TestParseOLLAMAHost_Missing(t *testing.T) {
	os.Unsetenv("OLLAMA_HOST")
	if _, _, err := parseOLLAMAHost(logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error when OLLAMA_HOST is unset")
	}
}

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

func TestGenerateUniqueOllamaPortRandomness(t *testing.T) {
	existing := uint16(12000)
	uniqueCount := 0
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		pStr := generateUniqueOllamaPort(existing)
		if pStr == strconv.Itoa(int(existing)) {
			t.Fatalf("generated port equals existing port")
		}
		val, err := strconv.Atoi(pStr)
		if err != nil {
			t.Fatalf("invalid port string: %v", err)
		}
		if val < minPort || val > maxPort {
			t.Fatalf("port out of range: %d", val)
		}
		if _, dup := seen[pStr]; !dup {
			uniqueCount++
			seen[pStr] = struct{}{}
		}
	}
	if uniqueCount < 10 {
		t.Fatalf("insufficient randomness: only %d unique ports generated", uniqueCount)
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
