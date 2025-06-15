package docker

import (
	"strconv"
	"testing"
)

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
