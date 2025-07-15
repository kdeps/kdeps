package version_test

import (
	"regexp"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionVariables(t *testing.T) {
	// Test that Version has a default value
	vi := version.GetVersionInfo()
	assert.Equal(t, "dev", vi.Version)

	// Test that Commit has a default value
	assert.Equal(t, "", vi.Commit)

	// Test that we can modify the variables
	version.SetVersionInfo("1.0.0", "abc123")
	vi2 := version.GetVersionInfo()
	assert.Equal(t, "1.0.0", vi2.Version)
	assert.Equal(t, "abc123", vi2.Commit)
	// Restore
	version.SetVersionInfo("dev", "")
	vi3 := version.GetVersionInfo()
	assert.Equal(t, "dev", vi3.Version)
	assert.Equal(t, "", vi3.Commit)
}

func TestVersion(t *testing.T) {
	// Test case 1: Check if version string is not empty
	vi := version.GetVersionInfo()
	if vi.Version == "" {
		t.Errorf("Version string is empty, expected a non-empty version")
	}
	t.Log("Version string test passed")
}

func TestVersionDefaults(t *testing.T) {
	vi := version.GetVersionInfo()
	require.Equal(t, "dev", vi.Version)
	require.Equal(t, "", vi.Commit)
}

func TestDefaultVersionValues(t *testing.T) {
	// Test that all default version constants are not empty
	assert.NotEmpty(t, version.DefaultSchemaVersion, "DefaultSchemaVersion should not be empty")
	assert.NotEmpty(t, version.DefaultAnacondaVersion, "DefaultAnacondaVersion should not be empty")
	assert.NotEmpty(t, version.DefaultPklVersion, "DefaultPklVersion should not be empty")
	assert.NotEmpty(t, version.DefaultOllamaImageTag, "DefaultOllamaImageTag should not be empty")
	assert.NotEmpty(t, version.MinimumSchemaVersion, "MinimumSchemaVersion should not be empty")

	// Test that they follow expected version format patterns
	semverPattern := `^\d+\.\d+\.\d+$`
	dateVersionPattern := `^\d{4}\.\d{2}-\d+$`

	// Schema version should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), version.DefaultSchemaVersion, "DefaultSchemaVersion should follow semantic versioning")
	assert.Regexp(t, regexp.MustCompile(semverPattern), version.MinimumSchemaVersion, "MinimumSchemaVersion should follow semantic versioning")

	// Anaconda version should follow date-based versioning
	assert.Regexp(t, regexp.MustCompile(dateVersionPattern), version.DefaultAnacondaVersion, "DefaultAnacondaVersion should follow date-based versioning")

	// PKL version should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), version.DefaultPklVersion, "DefaultPklVersion should follow semantic versioning")

	// Ollama image tag should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), version.DefaultOllamaImageTag, "DefaultOllamaImageTag should follow semantic versioning")

	// Default schema version should be >= minimum using utils.CompareVersions
	cmp, err := utils.CompareVersions(version.DefaultSchemaVersion, version.MinimumSchemaVersion)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, cmp, 0, "DefaultSchemaVersion should be >= MinimumSchemaVersion")
}

func TestOverrideVersionValues(t *testing.T) {
	version.SetVersionInfo("1.2.3", "abc123")
	vi := version.GetVersionInfo()
	if vi.Version != "1.2.3" {
		t.Errorf("override failed for Version, got %s", vi.Version)
	}
	if vi.Commit != "abc123" {
		t.Errorf("override failed for Commit, got %s", vi.Commit)
	}
	// Restore
	version.SetVersionInfo("dev", "")
}

func TestVersionVars(t *testing.T) {
	vi := version.GetVersionInfo()
	if vi.Version == "" {
		t.Fatalf("Version should not be empty")
	}
	// Commit may be empty in dev builds but accessing it should not panic.
	_ = vi.Commit
}
