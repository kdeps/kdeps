package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
