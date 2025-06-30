package version

import (
	"regexp"
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
	// Test that all default version constants are not empty
	assert.NotEmpty(t, DefaultSchemaVersion, "DefaultSchemaVersion should not be empty")
	assert.NotEmpty(t, DefaultAnacondaVersion, "DefaultAnacondaVersion should not be empty")
	assert.NotEmpty(t, DefaultPklVersion, "DefaultPklVersion should not be empty")
	assert.NotEmpty(t, DefaultOllamaImageTag, "DefaultOllamaImageTag should not be empty")
	assert.NotEmpty(t, MinimumSchemaVersion, "MinimumSchemaVersion should not be empty")

	// Test that they follow expected version format patterns
	semverPattern := `^\d+\.\d+\.\d+$`
	dateVersionPattern := `^\d{4}\.\d{2}-\d+$`
	
	// Schema version should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), DefaultSchemaVersion, "DefaultSchemaVersion should follow semantic versioning")
	assert.Regexp(t, regexp.MustCompile(semverPattern), MinimumSchemaVersion, "MinimumSchemaVersion should follow semantic versioning")
	
	// Anaconda version should follow date-based versioning
	assert.Regexp(t, regexp.MustCompile(dateVersionPattern), DefaultAnacondaVersion, "DefaultAnacondaVersion should follow date-based versioning")
	
	// PKL version should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), DefaultPklVersion, "DefaultPklVersion should follow semantic versioning")
	
	// Ollama image tag should follow semantic versioning
	assert.Regexp(t, regexp.MustCompile(semverPattern), DefaultOllamaImageTag, "DefaultOllamaImageTag should follow semantic versioning")
	
	// Default schema version should be >= minimum
	cmp, err := CompareVersions(DefaultSchemaVersion, MinimumSchemaVersion)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, cmp, 0, "DefaultSchemaVersion should be >= MinimumSchemaVersion")
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

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
		hasError bool
	}{
		{"equal versions", "1.2.3", "1.2.3", 0, false},
		{"v1 greater major", "2.0.0", "1.9.9", 1, false},
		{"v1 less major", "1.0.0", "2.0.0", -1, false},
		{"v1 greater minor", "1.2.0", "1.1.9", 1, false},
		{"v1 less minor", "1.1.0", "1.2.0", -1, false},
		{"v1 greater patch", "1.2.3", "1.2.2", 1, false},
		{"v1 less patch", "1.2.2", "1.2.3", -1, false},
		{"invalid v1", "1.2", "1.2.3", 0, true},
		{"invalid v2", "1.2.3", "1.2", 0, true},
		{"non-numeric v1", "1.a.3", "1.2.3", 0, true},
		{"non-numeric v2", "1.2.3", "1.b.3", 0, true},
		{"negative version", "-1.2.3", "1.2.3", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareVersions(tt.v1, tt.v2)
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateSchemaVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		hasError bool
	}{
		{"valid current version", DefaultSchemaVersion, false},
		{"valid minimum version", MinimumSchemaVersion, false},
		{"valid higher version", "1.0.0", false},
		{"below minimum", "0.1.9", true},
		{"empty version", "", true},
		{"invalid format", "1.2", true},
		{"non-numeric", "1.a.3", true},
		{"negative version", "-1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchemaVersion(tt.version)
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsSchemaVersionSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		supported bool
	}{
		{"current version", DefaultSchemaVersion, true},
		{"minimum version", MinimumSchemaVersion, true},
		{"higher version", "1.0.0", true},
		{"below minimum", "0.1.9", false},
		{"empty version", "", false},
		{"invalid format", "1.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSchemaVersionSupported(tt.version)
			assert.Equal(t, tt.supported, result)
		})
	}
}
