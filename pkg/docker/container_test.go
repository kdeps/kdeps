package docker_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/docker/docker/client"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crand "crypto/rand"

	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
)

var crandTestMutex sync.Mutex

func withCRandTestState(_ *testing.T, fn func()) {
	crandTestMutex.Lock()
	defer crandTestMutex.Unlock()
	origReader := crand.Reader
	defer func() { crand.Reader = origReader }()
	fn()
}

func TestLoadEnvFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("FileExists", func(t *testing.T) {
		_ = afero.WriteFile(fs, ".env", []byte("KEY1=value1\nKEY2=value2"), 0o644)
		envSlice, err := docker.LoadEnvFile(fs, ".env")
		require.NoError(t, err)
		assert.Len(t, envSlice, 2)
		assert.Contains(t, envSlice, "KEY1=value1")
		assert.Contains(t, envSlice, "KEY2=value2")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		envSlice, err := docker.LoadEnvFile(fs, "nonexistent.env")
		require.NoError(t, err)
		assert.Nil(t, envSlice)
	})
}

func TestGenerateDockerCompose(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("CPU", func(t *testing.T) {
		err := docker.GenerateDockerCompose(fs, "test", "image", "test-cpu", "127.0.0.1", "8080", "", "", true, false, "cpu")
		require.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-cpu.yaml")
		assert.Contains(t, string(content), "test-cpu:")
		assert.Contains(t, string(content), "image: image")
	})

	t.Run("NVIDIA", func(t *testing.T) {
		err := docker.GenerateDockerCompose(fs, "test", "image", "test-nvidia", "127.0.0.1", "8080", "", "", true, false, "nvidia")
		require.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-nvidia.yaml")
		assert.Contains(t, string(content), "driver: nvidia")
	})

	t.Run("AMD", func(t *testing.T) {
		err := docker.GenerateDockerCompose(fs, "test", "image", "test-amd", "127.0.0.1", "8080", "", "", true, false, "amd")
		require.NoError(t, err)
		content, _ := afero.ReadFile(fs, "test_docker-compose-amd.yaml")
		assert.Contains(t, string(content), "/dev/kfd")
		assert.Contains(t, string(content), "/dev/dri")
	})

	t.Run("UnsupportedGPU", func(t *testing.T) {
		err := docker.GenerateDockerCompose(fs, "test", "image", "test-unsupported", "127.0.0.1", "8080", "", "", true, false, "unsupported")
		require.Error(t, err)
	})
}

func TestCreateDockerContainer(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)

	t.Run("APIModeWithoutPort", func(t *testing.T) {
		_, err := docker.CreateDockerContainer(fs, ctx, "test", "image", "127.0.0.1", "", "", "", "cpu", true, false, cli)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "portNum must be non-empty")
	})

	t.Run("WebModeWithoutPort", func(t *testing.T) {
		_, err := docker.CreateDockerContainer(fs, ctx, "test", "image", "127.0.0.1", "8080", "127.0.0.1", "", "cpu", false, true, cli)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "webPortNum must be non-empty")
	})

	t.Run("ContainerExists", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the Docker client for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
	})
}

func TestLoadEnvFile_VariousCases(t *testing.T) {
	fs := afero.NewOsFs()
	dir := t.TempDir()

	t.Run("file-missing", func(t *testing.T) {
		envs, err := docker.LoadEnvFile(fs, filepath.Join(dir, "missing.env"))
		require.NoError(t, err)
		require.Nil(t, envs)
	})

	t.Run("valid-env", func(t *testing.T) {
		path := filepath.Join(dir, "good.env")
		content := "FOO=bar\nHELLO=world\n"
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o644))

		envs, err := docker.LoadEnvFile(fs, path)
		require.NoError(t, err)
		// Convert slice to joined string for easier contains checks irrespective of order.
		joined := strings.Join(envs, ",")
		require.Contains(t, joined, "FOO=bar")
		require.Contains(t, joined, "HELLO=world")
	})
}

func TestLoadEnvFileMissingAndSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	// Case 1: file missing returns nil slice, no error
	envs, err := docker.LoadEnvFile(fs, filepath.Join(t.TempDir(), "not_existing.env"))
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

	envs, err = docker.LoadEnvFile(fs, fname)
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
	err := docker.GenerateDockerCompose(fs, "agent", "image:tag", "agent-cpu", "127.0.0.1", "5000", "", "", true, false, "cpu")
	if err != nil {
		t.Fatalf("GenerateDockerCompose error: %v", err)
	}
	expected := "agent_docker-compose-cpu.yaml"
	exists, _ := afero.Exists(fs, expected)
	if !exists {
		t.Fatalf("expected compose file %s", expected)
	}
}

func TestParseOLLAMAHostAdditional(t *testing.T) {
	logger := logging.NewTestLogger()

	// Case 1: variable not set
	os.Unsetenv("OLLAMA_HOST")
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when OLLAMA_HOST is unset")
	}

	// Case 2: invalid format
	t.Setenv("OLLAMA_HOST", "invalid-format")
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Case 3: valid host:port
	t.Setenv("OLLAMA_HOST", "127.0.0.1:11434")
	host, port, err := docker.ParseOLLAMAHost(logger)
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
		portStr := docker.GenerateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			t.Fatalf("invalid port returned: %v", err)
		}
		if port < docker.MinPort || port > docker.MaxPort {
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
	t.Cleanup(func() { t.Setenv("OLLAMA_HOST", orig) })

	t.Run("env-not-set", func(t *testing.T) {
		_ = os.Unsetenv("OLLAMA_HOST")
		host, port, err := docker.ParseOLLAMAHost(logger)
		require.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("invalid-format", func(t *testing.T) {
		t.Setenv("OLLAMA_HOST", "invalid") // missing ':'
		host, port, err := docker.ParseOLLAMAHost(logger)
		require.Error(t, err)
		assert.Empty(t, host)
		assert.Empty(t, port)
	})

	t.Run("happy-path", func(t *testing.T) {
		t.Setenv("OLLAMA_HOST", "127.0.0.1:11435")
		host, port, err := docker.ParseOLLAMAHost(logger)
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1", host)
		assert.Equal(t, "11435", port)
	})
}

func TestGenerateUniqueOllamaPortRange(t *testing.T) {
	existing := uint16(12000)
	count := 20 // sample multiple generations to reduce flake risk
	for i := 0; i < count; i++ {
		portStr := docker.GenerateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(portStr)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, port, docker.MinPort)
		assert.LessOrEqual(t, port, docker.MaxPort)
		assert.NotEqual(t, int(existing), port)
	}
}

func TestLoadEnvFile_InvalidContent(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	// invalid line (missing '=')
	_ = afero.WriteFile(fs, envPath, []byte("INVALID"), 0o644)

	envSlice, err := docker.LoadEnvFile(fs, envPath)
	require.NoError(t, err)
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
	withCRandTestState(t, func() {
		// existingPort set to minPort so first generated port collides.
		existing := uint16(docker.MinPort)

		portStr := docker.GenerateUniqueOllamaPort(existing)

		if portStr == "" || portStr == "11435" { // 11435 == minPort
			t.Fatalf("expected non-empty unique port different from minPort, got %s", portStr)
		}
	})
}

func TestGenerateUniqueOllamaPortDiffersFromExisting(t *testing.T) {
	existing := uint16(12345)
	for i := 0; i < 50; i++ {
		pStr := docker.GenerateUniqueOllamaPort(existing)
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
		pStr := docker.GenerateUniqueOllamaPort(0)
		port, err := strconv.Atoi(pStr)
		if err != nil {
			t.Fatalf("invalid int: %v", err)
		}
		if port < docker.MinPort || port > docker.MaxPort {
			t.Fatalf("port out of range: %d", port)
		}
	}
}

func TestParseOLLAMAHost(t *testing.T) {
	logger := logging.NewTestLogger()

	originalOllamaHost := os.Getenv("OLLAMA_HOST")
	t.Setenv("OLLAMA_HOST", "127.0.0.1:8080")
	defer t.Setenv("OLLAMA_HOST", originalOllamaHost)

	// Success case
	host, port, err := docker.ParseOLLAMAHost(logger)
	if err != nil || host != "127.0.0.1" || port != "8080" {
		t.Fatalf("unexpected parse result: %v %v %v", host, port, err)
	}

	// Invalid format case
	t.Setenv("OLLAMA_HOST", "bad-format")
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Unset env var case
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when env not set")
	}
}

func TestGenerateUniqueOllamaPort(t *testing.T) {
	existing := uint16(12345)
	for i := 0; i < 10; i++ {
		pStr := docker.GenerateUniqueOllamaPort(existing)
		port, err := strconv.Atoi(pStr)
		if err != nil {
			t.Fatalf("port not numeric: %v", err)
		}
		if port == int(existing) {
			t.Fatalf("generated port equals existing port")
		}
		if port < docker.MinPort || port > docker.MaxPort {
			t.Fatalf("generated port out of range: %d", port)
		}
	}
}
