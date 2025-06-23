package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/kdeps/kdeps/pkg/version"
)

// TestVersionConstants verifies version constants are properly defined and follow expected patterns.
//
// IMPORTANT: These tests are designed to be resilient to version bumps. They test:
// - Format/pattern compliance (e.g., semantic versioning)
// - Non-empty values and reasonable defaults
// - Cross-package usage patterns
//
// They do NOT test specific hardcoded values, which would break on every version bump.
// When bumping versions, you should only need to update pkg/version/version.go,
// not these tests.

func TestVersionVariables(t *testing.T) {
	// Test that Version has a default value
	assert.Equal(t, "dev", Version)

	// Test that Commit has a default value
	assert.Equal(t, "", Commit)

	// Test that we can modify the variables
	originalVersion := Version
	originalCommit := Commit

	Version = "1.0.0"
	Commit = "abc123"

	assert.Equal(t, "1.0.0", Version)
	assert.Equal(t, "abc123", Commit)

	// Restore original values
	Version = originalVersion
	Commit = originalCommit

	assert.Equal(t, "dev", Version)
	assert.Equal(t, "", Commit)
}

func TestVersion(t *testing.T) {
	// Test case 1: Check if version string is not empty
	if Version == "" {
		t.Errorf("Version string is empty, expected a non-empty version")
	}
	t.Log("Version string test passed")
}

func TestVersionDefaults(t *testing.T) {
	require.Equal(t, "dev", Version)
	require.Equal(t, "", Commit)
}

func TestDefaultVersionValues(t *testing.T) {
	if Version != "dev" {
		t.Errorf("expected default Version 'dev', got %s", Version)
	}
	if Commit != "" {
		t.Errorf("expected default Commit '', got %s", Commit)
	}
}

func TestOverrideVersionValues(t *testing.T) {
	origVer, origCommit := Version, Commit
	Version = "1.2.3"
	Commit = "abc123"

	if Version != "1.2.3" {
		t.Errorf("override failed for Version, got %s", Version)
	}
	if Commit != "abc123" {
		t.Errorf("override failed for Commit, got %s", Commit)
	}

	// restore
	Version, Commit = origVer, origCommit
}

func TestVersionVars(t *testing.T) {
	if Version == "" {
		t.Fatalf("Version should not be empty")
	}
	// Commit may be empty in dev builds but accessing it should not panic.
	_ = Commit
}

func TestVersionConstants(t *testing.T) {
	t.Run("ExternalToolVersions", func(t *testing.T) {
		assert.NotEmpty(t, PklVersion, "PklVersion should not be empty")
		assert.NotEmpty(t, AnacondaVersion, "AnacondaVersion should not be empty")
		assert.NotEmpty(t, SchemaVersion, "SchemaVersion should not be empty")

		// Verify expected format
		assert.Contains(t, PklVersion, ".", "PklVersion should contain version dots")
		assert.Contains(t, AnacondaVersion, ".", "AnacondaVersion should contain version dots")
		assert.Contains(t, SchemaVersion, ".", "SchemaVersion should contain version dots")
	})

	t.Run("DockerImageTags", func(t *testing.T) {
		assert.NotEmpty(t, DefaultOllamaImageTag, "DefaultOllamaImageTag should not be empty")
		assert.Equal(t, "latest", LatestTag, "LatestTag should be 'latest'")
		assert.Equal(t, "latest", LatestVersionPlaceholder, "LatestVersionPlaceholder should be 'latest'")

		// Verify expected format
		assert.Contains(t, DefaultOllamaImageTag, ".", "DefaultOllamaImageTag should contain version dots")
	})

	t.Run("VersionConsistency", func(t *testing.T) {
		// Instead of hardcoded values, test that versions follow expected patterns
		// and are being used consistently across the codebase

		// Test version format patterns
		assert.Regexp(t, `^\d+\.\d+\.\d+$`, PklVersion, "PklVersion should follow semantic versioning (x.y.z)")
		assert.Regexp(t, `^\d{4}\.\d{1,2}-\d+$`, AnacondaVersion, "AnacondaVersion should follow year.month-build format")
		assert.Regexp(t, `^\d+\.\d+\.\d+$`, SchemaVersion, "SchemaVersion should follow semantic versioning (x.y.z)")
		assert.Regexp(t, `^\d+\.\d+\.\d+$`, DefaultOllamaImageTag, "DefaultOllamaImageTag should follow semantic versioning (x.y.z)")

		// Test that constants are reasonable (not empty, not placeholder values)
		assert.NotEqual(t, "0.0.0", PklVersion, "PklVersion should not be placeholder")
		assert.NotEqual(t, "0.0.0", SchemaVersion, "SchemaVersion should not be placeholder")
		assert.NotEqual(t, "0.0.0", DefaultOllamaImageTag, "DefaultOllamaImageTag should not be placeholder")
		assert.NotEqual(t, "latest", DefaultOllamaImageTag, "DefaultOllamaImageTag should be a specific version, not 'latest'")

		// Test that LatestVersionPlaceholder is actually "latest"
		assert.Equal(t, "latest", LatestVersionPlaceholder, "LatestVersionPlaceholder should be 'latest'")
		assert.Equal(t, "latest", LatestTag, "LatestTag should be 'latest'")
	})

	t.Run("CrossPackageUsage", func(t *testing.T) {
		// Verify that the constants can be accessed and used as expected
		// This demonstrates that the centralization is working properly

		// Test that we can construct common patterns using these constants
		dockerImageRef := "ollama/ollama:" + DefaultOllamaImageTag
		assert.Contains(t, dockerImageRef, "ollama/ollama:")
		assert.Contains(t, dockerImageRef, DefaultOllamaImageTag)

		schemaPackageRef := "package://schema.kdeps.com/core@" + SchemaVersion + "#/Workflow.pkl"
		assert.Contains(t, schemaPackageRef, "package://schema.kdeps.com/core@")
		assert.Contains(t, schemaPackageRef, SchemaVersion)
		assert.Contains(t, schemaPackageRef, "#/Workflow.pkl")

		// Test that all constants are unique (no accidental duplicates)
		versions := []string{PklVersion, AnacondaVersion, SchemaVersion, DefaultOllamaImageTag}
		uniqueVersions := make(map[string]bool)
		for _, v := range versions {
			if v != LatestVersionPlaceholder { // "latest" is expected to be reused
				if uniqueVersions[v] {
					t.Errorf("Duplicate version found: %s", v)
				}
				uniqueVersions[v] = true
			}
		}
	})
}
