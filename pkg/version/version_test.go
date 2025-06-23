package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/kdeps/kdeps/pkg/version"
)

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
		// Verify that version constants are actually being used in the codebase
		// by checking they have reasonable values
		assert.Equal(t, "0.28.1", PklVersion, "PklVersion should match expected value")
		assert.Equal(t, "2024.10-1", AnacondaVersion, "AnacondaVersion should match expected value")
		assert.Equal(t, "0.2.30", SchemaVersion, "SchemaVersion should match expected value")
		assert.Equal(t, "0.6.8", DefaultOllamaImageTag, "DefaultOllamaImageTag should match expected value")
	})
}
