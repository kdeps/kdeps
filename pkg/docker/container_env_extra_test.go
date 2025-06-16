package docker

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestLoadEnvFileMissingAndSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	// Case 1: file missing returns nil slice, no error
	envs, err := loadEnvFile(fs, "/tmp/not_existing.env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envs != nil {
		t.Fatalf("expected nil slice for missing file, got %v", envs)
	}

	// Case 2: valid .env file parsed
	tmpDir, _ := afero.TempDir(fs, "", "env")
	fname := tmpDir + "/.env"
	content := "FOO=bar\nHELLO=world"
	_ = afero.WriteFile(fs, fname, []byte(content), 0o644)

	envs, err = loadEnvFile(fs, fname)
	if err != nil {
		t.Fatalf("loadEnvFile error: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(envs))
	}
	joined := strings.Join(envs, ",")
	if !strings.Contains(joined, "FOO=bar") || !strings.Contains(joined, "HELLO=world") {
		t.Fatalf("parsed env slice missing values: %v", envs)
	}
}

func TestGenerateDockerComposeCPU(t *testing.T) {
	fs := afero.NewOsFs()
	err := GenerateDockerCompose(fs, "agent", "image:tag", "agent-cpu", "127.0.0.1", "5000", "", "", true, false, "cpu")
	if err != nil {
		t.Fatalf("GenerateDockerCompose error: %v", err)
	}
	expected := "agent_docker-compose-cpu.yaml"
	exists, _ := afero.Exists(fs, expected)
	if !exists {
		t.Fatalf("expected compose file %s", expected)
	}
}
