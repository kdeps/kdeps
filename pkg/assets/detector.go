package assets

import (
	"os"
	"strings"
	"testing"
)

// ShouldUseEmbeddedAssets determines if we should use embedded PKL assets instead of external URLs
func ShouldUseEmbeddedAssets() bool {
	return IsDockerMode() || IsTestEnvironment()
}

// IsDockerMode checks if we're running in Docker mode
func IsDockerMode() bool {
	// Check for Docker environment variables that kdeps uses
	dockerEnvVars := []string{
		"KDEPS_DOCKER_MODE",
		"DOCKER_KDEPS_DIR",
		"DOCKER_KDEPS_PATH",
		"DOCKER_RUN_MODE",
		"DOCKER_GPU",
	}

	for _, envVar := range dockerEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	// Also check if we're inside a Docker container
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	return false
}

// IsTestEnvironment checks if we're running in a test environment
func IsTestEnvironment() bool {
	// Check if we're running under go test
	for _, arg := range os.Args {
		if strings.Contains(arg, "go-build") && strings.Contains(arg, "_test") {
			return true
		}
		if strings.HasSuffix(arg, ".test") {
			return true
		}
	}

	// Check for test-related environment variables
	if strings.HasSuffix(os.Args[0], ".test") {
		return true
	}

	// Check if testing.Testing() would return true (this is a bit of a hack)
	// We can't call testing.Testing() directly since it's not always available
	if os.Getenv("GO_TEST_TIMEOUT_SCALE") != "" {
		return true
	}

	return false
}

// IsTestMode is a helper that can be called from test contexts
func IsTestMode(t *testing.T) bool {
	return t != nil
}
