package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// validateAgentName should reject empty, whitespace, and names with spaces, and accept valid names.
func TestValidateAgentNameExtra(t *testing.T) {
	require.Error(t, validateAgentName(""))
	require.Error(t, validateAgentName("   "))
	require.Error(t, validateAgentName("bad name"))
	require.NoError(t, validateAgentName("goodName"))
}

// promptForAgentName should return default in non-interactive mode.
func TestPromptForAgentNameNonInteractiveExtra(t *testing.T) {
	os.Setenv("NON_INTERACTIVE", "1")
	defer os.Unsetenv("NON_INTERACTIVE")

	name, err := promptForAgentName()
	require.NoError(t, err)
	require.Equal(t, "test-agent", name)
}

// TestCreateDirectoryAndFile verifies createDirectory and createFile behavior.
func TestCreateDirectoryAndFileExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	os.Setenv("NON_INTERACTIVE", "1")
	defer os.Unsetenv("NON_INTERACTIVE")

	// Test createDirectory
	err := createDirectory(fs, logger, "dir/subdir")
	require.NoError(t, err)
	exists, err := afero.DirExists(fs, "dir/subdir")
	require.NoError(t, err)
	require.True(t, exists)

	// Test createFile
	err = createFile(fs, logger, "dir/subdir/file.txt", "content")
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, "dir/subdir/file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("content"), data)
}

// TestLoadTemplateFromDisk verifies loadTemplate reads from TEMPLATE_DIR when set.
func TestLoadTemplateFromDiskExtra(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("TEMPLATE_DIR", tmpDir)
	defer os.Unsetenv("TEMPLATE_DIR")

	// Write a simple template file
	templateName := "foo.tmpl"
	content := "Hello {{.Name}}"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, templateName), []byte(content), 0644))

	out, err := loadTemplate(templateName, map[string]string{"Name": "Bob"})
	require.NoError(t, err)
	require.Equal(t, "Hello Bob", out)
}

// TestGenerateWorkflowFile covers error for invalid name and success path.
func TestGenerateWorkflowFileExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	os.Setenv("NON_INTERACTIVE", "1")
	defer os.Unsetenv("NON_INTERACTIVE")

	// Invalid name should return error
	err := GenerateWorkflowFile(fs, context.Background(), logger, "outdir", "bad name")
	require.Error(t, err)

	// Setup disk template
	tmpDir := t.TempDir()
	os.Setenv("TEMPLATE_DIR", tmpDir)
	defer os.Unsetenv("TEMPLATE_DIR")
	tmplPath := filepath.Join(tmpDir, "workflow.pkl")
	require.NoError(t, os.WriteFile(tmplPath, []byte("X:{{.Name}}"), 0644))

	// Successful generation
	mainDir := "agentdir"
	err = GenerateWorkflowFile(fs, context.Background(), logger, mainDir, "Agent")
	require.NoError(t, err)
	output, err := afero.ReadFile(fs, filepath.Join(mainDir, "workflow.pkl"))
	require.NoError(t, err)
	require.Equal(t, "X:Agent", string(output))
}

// TestGenerateResourceFiles covers error for invalid name and success path.
func TestGenerateResourceFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	os.Setenv("NON_INTERACTIVE", "1")
	defer os.Unsetenv("NON_INTERACTIVE")

	// Invalid name
	err := GenerateResourceFiles(fs, context.Background(), logger, "outdir", "bad name")
	require.Error(t, err)

	// Setup disk templates directory matching embedded FS
	tmpDir := t.TempDir()
	os.Setenv("TEMPLATE_DIR", tmpDir)
	defer os.Unsetenv("TEMPLATE_DIR")
	// Create .pkl template files for each embedded resource (skip workflow.pkl)
	templateFiles := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, name := range templateFiles {
		path := filepath.Join(tmpDir, name)
		content := fmt.Sprintf("CONTENT:%s:{{.Name}}", name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	}

	mainDir := "agentdir2"
	err = GenerateResourceFiles(fs, context.Background(), logger, mainDir, "Agent")
	require.NoError(t, err)

	// client.pkl should be created with expected content
	clientPath := filepath.Join(mainDir, "resources", "client.pkl")
	output, err := afero.ReadFile(fs, clientPath)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("CONTENT:client.pkl:Agent"), string(output))
	// workflow.pkl should be skipped
	exists, err := afero.Exists(fs, filepath.Join(mainDir, "resources", "workflow.pkl"))
	require.NoError(t, err)
	require.False(t, exists)
}
