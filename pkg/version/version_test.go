package version_test

import (
	"regexp"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/kdeps/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package variable mutexes for safe reassignment
var (
	versionMutex sync.Mutex
	commitMutex  sync.Mutex
)

// Helper functions to safely save and restore package variables
func saveAndRestoreVersion(_ *testing.T, newValue string) func() {
	versionMutex.Lock()
	original := version.Version
	version.Version = newValue
	return func() {
		version.Version = original
		versionMutex.Unlock()
	}
}

func saveAndRestoreCommit(_ *testing.T, newValue string) func() {
	commitMutex.Lock()
	original := version.Commit
	version.Commit = newValue
	return func() {
		version.Commit = original
		commitMutex.Unlock()
	}
}

// testVersionState manages test state changes for version package
type testVersionState struct {
	origVersion string
	origCommit  string
}

func newTestVersionState() *testVersionState {
	return &testVersionState{
		origVersion: version.Version,
		origCommit:  version.Commit,
	}
}

func (ts *testVersionState) restore() {
	version.Version = ts.origVersion
	version.Commit = ts.origCommit
}

func withVersionTestState(t *testing.T, fn func()) {
	versionMutex.Lock()
	defer versionMutex.Unlock()

	state := newTestVersionState()
	defer state.restore()

	fn()
}

func TestVersionVariables(t *testing.T) {
	// Test that Version has a default value
	assert.Equal(t, "dev", version.Version)

	// Test that Commit has a default value
	assert.Equal(t, "", version.Commit)

	// Test that we can modify the variables
	withVersionTestState(t, func() {
		version.Version = "1.0.0"
		version.Commit = "abc123"

		assert.Equal(t, "1.0.0", version.Version)
		assert.Equal(t, "abc123", version.Commit)
	})

	// Restore original values
	assert.Equal(t, "dev", version.Version)
	assert.Equal(t, "", version.Commit)
}

func TestVersion(t *testing.T) {
	// Test case 1: Check if version string is not empty
	if version.Version == "" {
		t.Errorf("Version string is empty, expected a non-empty version")
	}
	t.Log("Version string test passed")
}

func TestVersionDefaults(t *testing.T) {
	require.Equal(t, "dev", version.Version)
	require.Equal(t, "", version.Commit)
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
	withVersionTestState(t, func() {
		version.Version = "1.2.3"
		version.Commit = "abc123"

		if version.Version != "1.2.3" {
			t.Errorf("override failed for Version, got %s", version.Version)
		}
		if version.Commit != "abc123" {
			t.Errorf("override failed for Commit, got %s", version.Commit)
		}
	})
}

func TestVersionVars(t *testing.T) {
	if version.Version == "" {
		t.Fatalf("Version should not be empty")
	}
	// Commit may be empty in dev builds but accessing it should not panic.
	_ = version.Commit
}
