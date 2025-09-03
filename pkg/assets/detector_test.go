package assets

import (
	"os"
	"testing"
)

func TestIsDockerMode(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	for _, env := range []string{"KDEPS_DOCKER_MODE", "DOCKER_KDEPS_DIR", "DOCKER_KDEPS_PATH", "DOCKER_RUN_MODE", "DOCKER_GPU"} {
		originalEnv[env] = os.Getenv(env)
	}
	defer func() {
		for env, value := range originalEnv {
			if value == "" {
				os.Unsetenv(env)
			} else {
				os.Setenv(env, value)
			}
		}
	}()

	tests := []struct {
		name     string
		envVar   string
		value    string
		expected bool
	}{
		{"KDEPS_DOCKER_MODE set", "KDEPS_DOCKER_MODE", "true", true},
		{"DOCKER_KDEPS_DIR set", "DOCKER_KDEPS_DIR", "/some/path", true},
		{"DOCKER_KDEPS_PATH set", "DOCKER_KDEPS_PATH", "/another/path", true},
		{"DOCKER_RUN_MODE set", "DOCKER_RUN_MODE", "container", true},
		{"DOCKER_GPU set", "DOCKER_GPU", "nvidia", true},
		{"No Docker env vars", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all Docker env vars first
			for _, env := range []string{"KDEPS_DOCKER_MODE", "DOCKER_KDEPS_DIR", "DOCKER_KDEPS_PATH", "DOCKER_RUN_MODE", "DOCKER_GPU"} {
				os.Unsetenv(env)
			}

			// Set the specific env var for this test
			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.value)
			}

			result := IsDockerMode()
			if result != tt.expected {
				t.Errorf("IsDockerMode() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsTestEnvironment(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	originalEnv := os.Getenv("GO_TEST_TIMEOUT_SCALE")
	defer func() {
		os.Args = originalArgs
		if originalEnv == "" {
			os.Unsetenv("GO_TEST_TIMEOUT_SCALE")
		} else {
			os.Setenv("GO_TEST_TIMEOUT_SCALE", originalEnv)
		}
	}()

	tests := []struct {
		name     string
		args     []string
		envVar   string
		expected bool
	}{
		{"go-build with _test", []string{"/tmp/go-build", "main_test.go"}, "", true},
		{"binary with .test suffix", []string{"/path/to/binary.test"}, "", true},
		{"GO_TEST_TIMEOUT_SCALE set", []string{"/path/to/binary"}, "2", true},
		{"normal binary", []string{"/path/to/binary"}, "", false},
		{"empty args", []string{}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			if tt.envVar == "" {
				os.Unsetenv("GO_TEST_TIMEOUT_SCALE")
			} else {
				os.Setenv("GO_TEST_TIMEOUT_SCALE", tt.envVar)
			}

			result := IsTestEnvironment()
			if result != tt.expected {
				t.Errorf("IsTestEnvironment() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestShouldUseEmbeddedAssets(t *testing.T) {
	// This should return true since we're in a test environment
	if !ShouldUseEmbeddedAssets() {
		t.Error("Expected ShouldUseEmbeddedAssets to return true in test environment")
	}
}

func TestIsTestMode(t *testing.T) {
	// Test with a valid testing.T
	if !IsTestMode(t) {
		t.Error("Expected IsTestMode to return true when passed a valid testing.T")
	}

	// Test with nil
	if IsTestMode(nil) {
		t.Error("Expected IsTestMode to return false when passed nil")
	}
}
