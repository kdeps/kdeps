package version

import "testing"

func TestVersionVariables(t *testing.T) {
	// Test that Version is not empty
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Test that Version is either "dev" or a valid version string
	if Version != "dev" {
		// If not "dev", it should be a valid version string
		// This is a basic check - you might want to add more specific version format validation
		if len(Version) < 3 { // At least "x.y" format
			t.Errorf("Version should be either 'dev' or a valid version string, got: %s", Version)
		}
	}

	// Commit can be empty in development, but should be a valid git commit hash when set
	if Commit != "" {
		if len(Commit) < 7 { // Git commit hashes are at least 7 characters
			t.Errorf("Commit should be a valid git commit hash when set, got: %s", Commit)
		}
	}
}
