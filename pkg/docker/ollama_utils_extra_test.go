package docker

import (
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
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
