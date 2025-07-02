package cmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpgradeCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := UpgradeCommand(fs, ctx, "/tmp", logger)

	assert.Contains(t, cmd.Use, "upgrade")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestUpgradeSchemaVersionInContent(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name           string
		content        string
		targetVersion  string
		expectedChange bool
		expectedResult string
	}{
		{
			name: "upgrade workflow amends",
			content: `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
AgentID = "test"`,
			targetVersion:  "0.3.2",
			expectedChange: true,
			expectedResult: `amends "package://schema.kdeps.com/core@0.3.2#/Workflow.pkl"
AgentID = "test"`,
		},
		{
			name: "upgrade resource import",
			content: `import "package://schema.kdeps.com/core@0.3.1#/Resource.pkl"
AgentID = "test"`,
			targetVersion:  "0.3.2",
			expectedChange: true,
			expectedResult: `import "package://schema.kdeps.com/core@0.3.2#/Resource.pkl"
AgentID = "test"`,
		},
		{
			name: "already at target version",
			content: `amends "package://schema.kdeps.com/core@0.3.2#/Workflow.pkl"
AgentID = "test"`,
			targetVersion:  "0.3.2",
			expectedChange: false,
			expectedResult: `amends "package://schema.kdeps.com/core@0.3.2#/Workflow.pkl"
AgentID = "test"`,
		},
		{
			name: "multiple version references",
			content: `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
import "package://schema.kdeps.com/core@0.3.1#/Resource.pkl"
AgentID = "test"`,
			targetVersion:  "0.3.2",
			expectedChange: true,
			expectedResult: `amends "package://schema.kdeps.com/core@0.3.2#/Workflow.pkl"
import "package://schema.kdeps.com/core@0.3.2#/Resource.pkl"
AgentID = "test"`,
		},
		{
			name: "no schema references",
			content: `AgentID = "test"
Version = "1.0.0"`,
			targetVersion:  "0.3.2",
			expectedChange: false,
			expectedResult: `AgentID = "test"
Version = "1.0.0"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed, err := upgradeSchemaVersionInContent(tt.content, tt.targetVersion, logger)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedChange, changed)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestUpgradeSchemaVersions(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create test directory structure
	testDir := "/test-agent"
	require.NoError(t, fs.MkdirAll(testDir, 0o755))
	require.NoError(t, fs.MkdirAll(filepath.Join(testDir, "resources"), 0o755))

	// Create test files
	workflowContent := `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
AgentID = "test-agent"
Version = "1.0.0"`

	resourceContent := `import "package://schema.kdeps.com/core@0.3.1#/Resource.pkl"
AgentID = "testResource"`

	nonPklContent := `{
  "name": "package.json",
  "version": "1.0.0"
}`

	require.NoError(t, afero.WriteFile(fs, filepath.Join(testDir, "workflow.pkl"), []byte(workflowContent), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(testDir, "resources", "test.pkl"), []byte(resourceContent), 0o644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(testDir, "package.json"), []byte(nonPklContent), 0o644))

	t.Run("dry run upgrade", func(t *testing.T) {
		err := upgradeSchemaVersions(fs, testDir, "0.3.2", true, logger)
		require.NoError(t, err)

		// Files should not be modified in dry run
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.3.1")
	})

	t.Run("actual upgrade", func(t *testing.T) {
		err := upgradeSchemaVersions(fs, testDir, "0.3.2", false, logger)
		require.NoError(t, err)

		// Check workflow.pkl was updated
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.3.2")
		assert.NotContains(t, string(content), "0.3.1")

		// Check resource file was updated
		content, err = afero.ReadFile(fs, filepath.Join(testDir, "resources", "test.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.3.2")
		assert.NotContains(t, string(content), "0.3.1")

		// Check non-pkl file was not modified
		content, err = afero.ReadFile(fs, filepath.Join(testDir, "package.json"))
		require.NoError(t, err)
		assert.Equal(t, nonPklContent, string(content))
	})
}

func TestUpgradeCommandValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := UpgradeCommand(fs, ctx, "/tmp", logger)

	t.Run("invalid target version", func(t *testing.T) {
		cmd.SetArgs([]string{"--version", "invalid", "."})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid target version")
	})

	t.Run("version below minimum", func(t *testing.T) {
		cmd.SetArgs([]string{"--version", "0.1.0", "."})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum supported version")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		cmd := UpgradeCommand(fs, ctx, "/tmp", logger)
		cmd.SetArgs([]string{"/nonexistent"})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory does not exist")
	})
}

func TestUpgradeCommandIntegration(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory with pkl files
	testDir := "/test-upgrade"
	require.NoError(t, fs.MkdirAll(testDir, 0o755))

	content := `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
AgentID = "test"
Version = "1.0.0"`

	require.NoError(t, afero.WriteFile(fs, filepath.Join(testDir, "workflow.pkl"), []byte(content), 0o644))

	// Test upgrade command
	cmd := UpgradeCommand(fs, ctx, "/tmp", logger)
	cmd.SetArgs([]string{"--version", "0.3.2", testDir})

	err := cmd.Execute()
	require.NoError(t, err)

	// Verify file was updated
	updatedContent, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
	require.NoError(t, err)
	assert.Contains(t, string(updatedContent), "0.3.2")
	assert.NotContains(t, string(updatedContent), "0.3.1")
}
