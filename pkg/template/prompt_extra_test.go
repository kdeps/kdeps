package template

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestPromptForAgentNameNonInteractive verifies that the helper returns the fixed
// value when NON_INTERACTIVE is set, without awaiting user input.
func TestPromptForAgentNameNonInteractive(t *testing.T) {
	// Backup existing value
	orig := os.Getenv("NON_INTERACTIVE")
	defer os.Setenv("NON_INTERACTIVE", orig)

	os.Setenv("NON_INTERACTIVE", "1")

	name, err := promptForAgentName()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "test-agent" {
		t.Errorf("expected 'test-agent', got %q", name)
	}
}

// TestGenerateAgentBasic creates an agent in a mem-fs and ensures that the core files
// are generated without touching the real filesystem.
func TestGenerateAgentBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	baseDir := "/workspace"
	agentName := "client"

	if err := GenerateAgent(fs, ctx, logger, baseDir, agentName); err != nil {
		t.Fatalf("GenerateAgent failed: %v", err)
	}

	// Expected files
	expects := []string{
		filepath.Join(baseDir, agentName, "workflow.pkl"),
		filepath.Join(baseDir, agentName, "resources", "client.pkl"),
		filepath.Join(baseDir, agentName, "resources", "exec.pkl"),
	}

	for _, path := range expects {
		exists, err := afero.Exists(fs, path)
		if err != nil {
			t.Fatalf("error checking %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected file %s to be generated", path)
		}
	}
}
