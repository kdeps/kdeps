package cmd_test

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	coreWorkflowAmends = "amends \"package://schema.kdeps.com/core@%s#/Workflow.pkl\""
	coreResourceImport = "import \"package://schema.kdeps.com/core@%s#/Resource.pkl\""
)

func TestUpgradeCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	upgradeCmd := cmd.UpgradeCommand(ctx, fs, "/tmp", logger)

	assert.Contains(t, upgradeCmd.Use, "upgrade")
	assert.NotEmpty(t, upgradeCmd.Short)
	assert.NotEmpty(t, upgradeCmd.Long)
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
			name:           "upgrade workflow amends",
			content:        fmt.Sprintf(coreWorkflowAmends+"\nAgentID = \"test\"", "0.3.1"),
			targetVersion:  "0.4.6",
			expectedChange: true,
			expectedResult: fmt.Sprintf(coreWorkflowAmends+"\nAgentID = \"test\"", "0.4.6"),
		},
		{
			name:           "upgrade resource import",
			content:        fmt.Sprintf(coreResourceImport+"\nAgentID = \"test\"", "0.3.1"),
			targetVersion:  "0.4.6",
			expectedChange: true,
			expectedResult: fmt.Sprintf(coreResourceImport+"\nAgentID = \"test\"", "0.4.6"),
		},
		{
			name:           "already at target version",
			content:        fmt.Sprintf(coreWorkflowAmends+"\nAgentID = \"test\"", "0.4.6"),
			targetVersion:  "0.4.6",
			expectedChange: false,
			expectedResult: fmt.Sprintf(coreWorkflowAmends+"\nAgentID = \"test\"", "0.4.6"),
		},
		{
			name:           "multiple version references",
			content:        fmt.Sprintf(coreWorkflowAmends+"\n"+coreResourceImport+"\nAgentID = \"test\"", "0.3.1", "0.3.1"),
			targetVersion:  "0.4.6",
			expectedChange: true,
			expectedResult: fmt.Sprintf(coreWorkflowAmends+"\n"+coreResourceImport+"\nAgentID = \"test\"", "0.4.6", "0.4.6"),
		},
		{
			name: "no schema references",
			content: `AgentID = "test"
Version = "1.0.0"`,
			targetVersion:  "0.4.6",
			expectedChange: false,
			expectedResult: `AgentID = "test"
Version = "1.0.0"`,
		},
		{
			name:           "duplicate amends lines",
			content:        fmt.Sprintf(coreWorkflowAmends+"\n"+coreWorkflowAmends+"\nAgentID = \"test\"", "0.3.1", "0.3.1"),
			targetVersion:  "0.4.6",
			expectedChange: true,
			expectedResult: fmt.Sprintf(coreWorkflowAmends+"\n"+coreWorkflowAmends+"\nAgentID = \"test\"", "0.4.6", "0.4.6"),
		},
		{
			name:           "duplicate import lines",
			content:        fmt.Sprintf(coreResourceImport+"\n"+coreResourceImport+"\nAgentID = \"test\"", "0.3.1", "0.3.1"),
			targetVersion:  "0.4.6",
			expectedChange: true,
			expectedResult: fmt.Sprintf(coreResourceImport+"\n"+coreResourceImport+"\nAgentID = \"test\"", "0.4.6", "0.4.6"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add debug output for failing test
			if tt.name == "upgrade workflow amends" {
				t.Logf("Input content: %q", tt.content)
				t.Logf("Target version: %s", tt.targetVersion)
			}

			result, changed, err := cmd.UpgradeSchemaVersionInContent(tt.content, tt.targetVersion, logger)
			require.NoError(t, err)

			// Add debug output for failing test
			if tt.name == "upgrade workflow amends" {
				t.Logf("Result: %q", result)
				t.Logf("Changed: %v", changed)
			}

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
		err := cmd.UpgradeSchemaVersions(context.Background(), fs, testDir, "0.4.6", true, logger)
		require.NoError(t, err)

		// Files should not be modified in dry run
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.3.1")
		assert.NotContains(t, string(content), "0.4.6")
	})

	t.Run("actual upgrade", func(t *testing.T) {
		err := cmd.UpgradeSchemaVersions(context.Background(), fs, testDir, "0.4.6", false, logger)
		require.NoError(t, err)

		// Check workflow.pkl was updated
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.4.6")
		assert.NotContains(t, string(content), "0.3.1")

		// Check resource file was updated
		content, err = afero.ReadFile(fs, filepath.Join(testDir, "resources", "test.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.4.6")
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

	upgradeCmd := cmd.UpgradeCommand(ctx, fs, "/tmp", logger)

	t.Run("invalid target version", func(t *testing.T) {
		upgradeCmd.SetArgs([]string{"--version", "invalid", "."})
		err := upgradeCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid target version")
	})

	t.Run("version below minimum", func(t *testing.T) {
		upgradeCmd.SetArgs([]string{"--version", "0.1.0", "."})
		err := upgradeCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum supported version")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		upgradeCmd := cmd.UpgradeCommand(ctx, fs, "/tmp", logger)
		upgradeCmd.SetArgs([]string{"/nonexistent"})
		err := upgradeCmd.Execute()
		require.Error(t, err)
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
	upgradeCmd := cmd.UpgradeCommand(ctx, fs, "/tmp", logger)
	upgradeCmd.SetArgs([]string{"--version", "0.4.6", testDir})

	err := upgradeCmd.Execute()
	require.NoError(t, err)

	// Verify file was updated
	updatedContent, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
	require.NoError(t, err)
	assert.Contains(t, string(updatedContent), "0.4.6")
	assert.NotContains(t, string(updatedContent), "0.3.1")
}

func TestRegexPattern(t *testing.T) {
	content := `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
AgentID = "test"`

	pattern := `(amends\s+"package://schema\.kdeps\.com/core@)([^"#]+)(#/[^"]+")`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(content, -1)

	t.Logf("Content: %q", content)
	t.Logf("Pattern: %s", pattern)
	t.Logf("Matches: %+v", matches)

	require.Len(t, matches, 1, "Should find exactly one match")
	require.Len(t, matches[0], 4, "Match should have 4 groups")

	currentVersion := matches[0][2]
	require.Equal(t, "0.3.1", currentVersion, "Should extract version 0.3.1")

	// Test replacement
	oldRef := matches[0][1] + currentVersion + matches[0][3]
	newRef := matches[0][1] + "0.4.6" + matches[0][3]

	expected := `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"
AgentID = "test"`

	result := strings.ReplaceAll(content, oldRef, newRef)
	require.Equal(t, expected, result, "Replacement should work correctly")
}

func TestUpgradeFunctionDirect(t *testing.T) {
	content := `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"
AgentID = "test"`

	t.Logf("Input content: %q", content)

	// Create a logger that will show debug output
	logger := logging.NewTestLogger()

	result, changed, err := cmd.UpgradeSchemaVersionInContent(content, "0.4.6", logger)

	t.Logf("Result: %q", result)
	t.Logf("Changed: %v", changed)
	t.Logf("Error: %v", err)

	require.NoError(t, err)
	require.True(t, changed, "Should have changed")
	require.Contains(t, result, "0.4.6", "Result should contain new version")
	require.NotContains(t, result, "0.3.1", "Result should not contain old version")
}

func TestUpgradeOllamaTagVersionInContent(t *testing.T) {
	logger := logging.NewTestLogger()

	tests := []struct {
		name           string
		content        string
		targetTag      string
		expectedChange bool
		expectedResult string
	}{
		{
			name: "upgrade Ollama tag version",
			content: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
        OllamaTagVersion = "0.9.2"
    }
}`,
			targetTag:      "0.9.6",
			expectedChange: true,
			expectedResult: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
        OllamaTagVersion = "0.9.6"
    }
}`,
		},
		{
			name: "already at target Ollama tag",
			content: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
        OllamaTagVersion = "0.9.6"
    }
}`,
			targetTag:      "0.9.6",
			expectedChange: false,
			expectedResult: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
        OllamaTagVersion = "0.9.6"
    }
}`,
		},
		{
			name: "no Ollama tag references",
			content: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

AgentID = "test"
Version = "1.0.0"`,
			targetTag:      "0.9.6",
			expectedChange: false,
			expectedResult: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

AgentID = "test"
Version = "1.0.0"`,
		},
		{
			name: "multiple Ollama tag references",
			content: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
                OllamaTagVersion = "0.9.2"
        // Another reference for testing
        // OllamaTagVersion = "0.9.2"
                        OllamaTagVersion = "0.9.2"
    }
}`,
			targetTag:      "0.9.6",
			expectedChange: true,
			expectedResult: `amends "package://schema.kdeps.com/core@0.4.6#/Workflow.pkl"

Settings {
    AgentSettings {
                OllamaTagVersion = "0.9.6"
        // Another reference for testing
        // OllamaTagVersion = "0.9.6"
                        OllamaTagVersion = "0.9.6"
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed, err := cmd.UpgradeOllamaTagVersionInContent(tt.content, tt.targetTag, logger)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedChange, changed, "Change flag should match expected")
			assert.Equal(t, tt.expectedResult, result, "Result should match expected content")
		})
	}
}

func TestUpgradeSchemaAndOllamaIntegration(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create test directory structure
	testDir := "/test-agent"
	require.NoError(t, fs.MkdirAll(testDir, 0o755))

	// Create test workflow file with both old schema and old Ollama tag
	workflowContent := `amends "package://schema.kdeps.com/core@0.3.1#/Workflow.pkl"

AgentID = "test-agent"
Version = "1.0.0"
TargetActionID = "responseResource"

Settings {
    AgentSettings {
        Models {
            "llama3.2:1b"
        }
        OllamaTagVersion = "0.9.2"
    }
}`

	require.NoError(t, afero.WriteFile(fs, filepath.Join(testDir, "workflow.pkl"), []byte(workflowContent), 0o644))

	t.Run("dry run upgrade both schema and Ollama", func(t *testing.T) {
		err := cmd.UpgradeSchemaVersions(context.Background(), fs, testDir, "0.4.6", true, logger)
		require.NoError(t, err)

		// Files should not be modified in dry run
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "0.3.1", "Should still contain old schema version")
		assert.Contains(t, string(content), "0.9.2", "Should still contain old Ollama tag")
		assert.NotContains(t, string(content), "0.4.6", "Should not contain new schema version")
		assert.NotContains(t, string(content), "0.9.6", "Should not contain new Ollama tag")
	})

	t.Run("actual upgrade both schema and Ollama", func(t *testing.T) {
		err := cmd.UpgradeSchemaVersions(context.Background(), fs, testDir, "0.4.6", false, logger)
		require.NoError(t, err)

		// Check workflow.pkl was updated for both schema and Ollama
		content, err := afero.ReadFile(fs, filepath.Join(testDir, "workflow.pkl"))
		require.NoError(t, err)

		// Schema version should be upgraded
		assert.Contains(t, string(content), "0.4.6", "Should contain new schema version")
		assert.NotContains(t, string(content), "0.3.1", "Should not contain old schema version")

		// Ollama tag should be upgraded
		assert.Contains(t, string(content), "0.9.6", "Should contain new Ollama tag")
		assert.NotContains(t, string(content), "0.9.2", "Should not contain old Ollama tag")

		// Other content should remain unchanged
		assert.Contains(t, string(content), "test-agent", "Should preserve AgentID")
		assert.Contains(t, string(content), "responseResource", "Should preserve TargetActionID")
		assert.Contains(t, string(content), "llama3.2:1b", "Should preserve model")
	})
}
