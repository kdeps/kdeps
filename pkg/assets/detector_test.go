package assets

import (
	"os"
	"testing"
)

func TestIsDockerMode(t *testing.T) {
	// Test environment variable detection
	os.Setenv("KDEPS_DOCKER_MODE", "true")
	defer os.Unsetenv("KDEPS_DOCKER_MODE")

	if !IsDockerMode() {
		t.Error("Expected IsDockerMode to return true when KDEPS_DOCKER_MODE is set")
	}
}

func TestIsTestEnvironment(t *testing.T) {
	// This should return true since we're running under go test
	if !IsTestEnvironment() {
		t.Error("Expected IsTestEnvironment to return true when running under go test")
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
