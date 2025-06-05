package data

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestPopulateDataFileRegistry(t *testing.T) {
	// Create a memory filesystem for testing
	fs := afero.NewMemMapFs()

	t.Run("EmptyDirectory", func(t *testing.T) {
		// Test with an empty directory
		registry, err := PopulateDataFileRegistry(fs, "/test")
		assert.NoError(t, err)
		assert.NotNil(t, registry)
		assert.Empty(t, *registry)
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		// Test with a non-existent directory
		registry, err := PopulateDataFileRegistry(fs, "/nonexistent")
		assert.NoError(t, err)
		assert.NotNil(t, registry)
		assert.Empty(t, *registry)
	})

	t.Run("ValidFileStructure", func(t *testing.T) {
		// Create a test directory structure
		fs.MkdirAll("/test/agent1/v1.0.0", 0o755)
		fs.MkdirAll("/test/agent2/v2.0.0", 0o755)

		// Create some test files
		afero.WriteFile(fs, "/test/agent1/v1.0.0/config.json", []byte("{}"), 0o644)
		afero.WriteFile(fs, "/test/agent1/v1.0.0/data.txt", []byte("data"), 0o644)
		afero.WriteFile(fs, "/test/agent2/v2.0.0/config.json", []byte("{}"), 0o644)

		// Test the registry population
		registry, err := PopulateDataFileRegistry(fs, "/test")
		assert.NoError(t, err)
		assert.NotNil(t, registry)

		// Verify the registry contents
		assert.Contains(t, *registry, "agent1/v1.0.0")
		assert.Contains(t, *registry, "agent2/v2.0.0")

		// Check agent1 files
		agent1Files := (*registry)["agent1/v1.0.0"]
		assert.Equal(t, "/test/agent1/v1.0.0/config.json", agent1Files["config.json"])
		assert.Equal(t, "/test/agent1/v1.0.0/data.txt", agent1Files["data.txt"])

		// Check agent2 files
		agent2Files := (*registry)["agent2/v2.0.0"]
		assert.Equal(t, "/test/agent2/v2.0.0/config.json", agent2Files["config.json"])
	})

	t.Run("InvalidFileStructure", func(t *testing.T) {
		// Create a test directory with invalid structure
		fs.MkdirAll("/test/invalid", 0o755)
		afero.WriteFile(fs, "/test/invalid/file.txt", []byte("data"), 0o644)

		// Test the registry population
		registry, err := PopulateDataFileRegistry(fs, "/test")
		assert.NoError(t, err)
		assert.NotNil(t, registry)

		// Verify that invalid structure is not included
		assert.NotContains(t, *registry, "invalid")
	})

	t.Run("NestedDirectories", func(t *testing.T) {
		// Create a test directory with nested structure
		fs.MkdirAll("/test/agent3/v3.0.0/nested/deep", 0o755)
		afero.WriteFile(fs, "/test/agent3/v3.0.0/nested/deep/file.txt", []byte("data"), 0o644)

		// Test the registry population
		registry, err := PopulateDataFileRegistry(fs, "/test")
		assert.NoError(t, err)
		assert.NotNil(t, registry)

		// Verify the nested file is included with correct path
		agent3Files := (*registry)["agent3/v3.0.0"]
		assert.Equal(t, "/test/agent3/v3.0.0/nested/deep/file.txt", agent3Files["nested/deep/file.txt"])
	})
}
