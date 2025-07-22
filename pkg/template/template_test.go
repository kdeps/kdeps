package template_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/kdeps/kdeps/pkg/version"
	assets "github.com/kdeps/schema/assets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package variable mutex for safe reassignment
var editPklMutex sync.Mutex

// Helper function to safely save and restore EditPkl variable
func saveAndRestoreEditPkl(_ *testing.T, newValue texteditor.EditPklFunc) func() {
	editPklMutex.Lock()
	original := texteditor.EditPkl
	texteditor.EditPkl = newValue
	return func() {
		texteditor.EditPkl = original
		editPklMutex.Unlock()
	}
}

func withTestState(_ *testing.T, fn func()) {
	origEditPkl := texteditor.EditPkl
	defer func() { texteditor.EditPkl = origEditPkl }()
	fn()
}

func setNonInteractive(t *testing.T) func() {
	t.Helper()
	oldValue := os.Getenv("NON_INTERACTIVE")
	t.Setenv("NON_INTERACTIVE", "1")
	return func() {
		t.Setenv("NON_INTERACTIVE", oldValue)
	}
}

func TestValidateAgentName(t *testing.T) {
	// Test case 1: Valid agent name
	err := template.ValidateAgentName("test-agent")
	if err != nil {
		t.Errorf("Expected no error for valid agent name, got: %v", err)
	}

	// Test case 2: Empty agent name
	err = template.ValidateAgentName("")
	if err == nil {
		t.Error("Expected error for empty agent name, got nil")
	}

	// Test case 3: Agent name with spaces
	err = template.ValidateAgentName("test agent")
	if err == nil {
		t.Error("Expected error for agent name with spaces, got nil")
	}

	t.Log("validateAgentName tests passed")
}

func TestCreateDirectoryNew(t *testing.T) {
	// Test case: Create directory with in-memory FS
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test", "dir")
	err := template.CreateDirectory(fs, logger, path)
	if err != nil {
		t.Errorf("Expected no error creating directory, got: %v", err)
	}
	// Check if directory exists
	exists, err := afero.DirExists(fs, path)
	if err != nil {
		t.Errorf("Error checking directory existence: %v", err)
	}
	if !exists {
		t.Error("Expected directory to exist, but it does not")
	}
	t.Log("createDirectory test passed")
}

func TestCreateFileNew(t *testing.T) {
	// Test case: Create file with in-memory FS
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test", "file.txt")
	content := "test content"
	err := template.CreateFile(fs, logger, path, content)
	if err != nil {
		t.Errorf("Expected no error creating file, got: %v", err)
	}
	// Check if file exists and content is correct
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Errorf("Error reading file: %v", err)
	}
	if string(data) != content {
		t.Errorf("Expected file content to be '%s', got '%s'", content, string(data))
	}
	t.Log("createFile test passed")
}

func TestPromptForAgentName_NonInteractive(t *testing.T) {
	// Test case: Non-interactive mode should return default name
	t.Setenv("NON_INTERACTIVE", "1")
	name, err := template.PromptForAgentName()
	if err != nil {
		t.Errorf("Expected no error in non-interactive mode, got: %v", err)
	}
	if name != "test-agent" {
		t.Errorf("Expected default name 'test-agent', got '%s'", name)
	}
	t.Log("promptForAgentName non-interactive test passed")
}

func TestCreateDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("CreateValidDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/directory")
		err := template.CreateDirectory(fs, logger, path)

		require.NoError(t, err)
		exists, err := afero.DirExists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("CreateNestedDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/nested/deep/directory")
		err := template.CreateDirectory(fs, logger, path)

		require.NoError(t, err)
		exists, err := afero.DirExists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("CreateExistingDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/existing")
		err := fs.MkdirAll(path, 0o755)
		require.NoError(t, err)

		err = template.CreateDirectory(fs, logger, path)
		assert.NoError(t, err)
	})

	t.Run("CreateDirectoryWithError", func(t *testing.T) {
		// Use a read-only filesystem to force an error
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		path := filepath.Join(tempDir, "test/readonly")
		err := template.CreateDirectory(readOnlyFs, logger, path)

		assert.Error(t, err)
	})
}

func TestCreateFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("CreateValidFile", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/file.txt")
		content := "test content"

		err := template.CreateFile(fs, logger, path, content)

		require.NoError(t, err)
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists)

		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("CreateFileInNestedDirectory", func(t *testing.T) {
		// Create directory first
		dir := filepath.Join(tempDir, "test/nested/dir")
		err := fs.MkdirAll(dir, 0o755)
		require.NoError(t, err)

		path := filepath.Join(dir, "file.txt")
		content := "nested file content"

		err = template.CreateFile(fs, logger, path, content)

		require.NoError(t, err)
		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("OverwriteExistingFile", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/overwrite.txt")
		originalContent := "original content"
		newContent := "new content"

		// Create original file
		err := afero.WriteFile(fs, path, []byte(originalContent), 0o644)
		require.NoError(t, err)

		// Overwrite with new content
		err = template.CreateFile(fs, logger, path, newContent)

		require.NoError(t, err)
		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Equal(t, newContent, string(data))
	})
}

func TestLoadTemplate(t *testing.T) {
	data := map[string]string{
		"Header": "test-header",
		"Name":   "test-name",
	}

	content, err := template.LoadTemplate("workflow.pkl", data)
	if err != nil {
		t.Fatalf("loadTemplate() error = %v", err)
	}

	if !strings.Contains(content, "test-header") {
		t.Errorf("Template content does not contain header: %s", content)
	}
	if !strings.Contains(content, "test-name") {
		t.Errorf("Template content does not contain name: %s", content)
	}
}

func TestTemplateLoadingEdgeCases(t *testing.T) {
	t.Run("TemplateWithEmptyData", func(t *testing.T) {
		templatePath := "templates/workflow.pkl"
		data := map[string]string{}

		content, err := template.LoadTemplate(templatePath, data)

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		// Verify that the template still loads even with empty data
		assert.Contains(t, content, "AgentID =")
		assert.Contains(t, content, "Description =")
	})

	t.Run("TemplateWithMissingVariables", func(t *testing.T) {
		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header",
			// Name is missing
		}

		content, err := template.LoadTemplate(templatePath, data)

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		// Verify that the template still loads but with empty variables
		assert.Contains(t, content, "test header")
		assert.Contains(t, content, "AgentID =")
	})

	t.Run("TemplateWithSpecialCharacters", func(t *testing.T) {
		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header with special chars: !@#$%^&*()",
			"Name":   "test-agent_with.special@chars",
		}

		content, err := template.LoadTemplate(templatePath, data)

		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "test header with special chars: !@#$%^&*()")
		assert.Contains(t, content, "test-agent_with.special@chars")
	})
}

func TestGenerateWorkflowFile(t *testing.T) {
	data := map[string]string{
		"Header": "test-header",
		"Name":   "test-name",
	}

	content, err := template.LoadTemplate("workflow.pkl", data)
	if err != nil {
		t.Fatalf("loadTemplate() error = %v", err)
	}

	if !strings.Contains(content, "test-header") {
		t.Errorf("Template content does not contain header: %s", content)
	}
	if !strings.Contains(content, "test-name") {
		t.Errorf("Template content does not contain name: %s", content)
	}
}

func TestGenerateResourceFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	mainDir := "test-agent"
	name := "test-agent"

	err := template.GenerateResourceFiles(ctx, fs, logger, mainDir, name)
	if err != nil {
		t.Fatalf("GenerateResourceFiles() error = %v", err)
	}

	// Verify resource files were created
	resourceDir := filepath.Join(mainDir, "resources")
	files, err := afero.ReadDir(fs, resourceDir)
	if err != nil {
		t.Fatalf("Error reading resource directory: %v", err)
	}

	// Check that we have the expected number of files
	expectedFiles := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	require.Len(t, files, len(expectedFiles), "Unexpected number of resource files")

	// Check each expected file exists
	for _, expectedFile := range expectedFiles {
		exists, err := afero.Exists(fs, filepath.Join(resourceDir, expectedFile))
		require.NoError(t, err)
		assert.True(t, exists, "Expected file %s does not exist", expectedFile)
	}
}

func TestGenerateSpecificAgentFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	mainDir := "test-agent"
	name := "client"

	err := template.GenerateSpecificAgentFile(ctx, fs, logger, mainDir, name)
	if err != nil {
		t.Fatalf("GenerateSpecificAgentFile() error = %v", err)
	}

	// Verify the file was created
	exists, err := afero.Exists(fs, filepath.Join(mainDir, "resources", name+".pkl"))
	if err != nil {
		t.Fatalf("Error checking file existence: %v", err)
	}
	if !exists {
		t.Error("Expected client.pkl file to be created")
	}

	// Read the file content
	content, err := afero.ReadFile(fs, filepath.Join(mainDir, "resources", name+".pkl"))
	if err != nil {
		t.Fatalf("Error reading generated file: %v", err)
	}

	// Check if the content contains the agent name
	if !strings.Contains(string(content), name) {
		t.Errorf("Generated file does not contain agent name: %s", content)
	}
}

func TestGenerateAgent(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	name := "test-agent"

	// First, generate the workflow file
	err := template.GenerateWorkflowFile(ctx, fs, logger, name, name)
	if err != nil {
		t.Fatalf("GenerateWorkflowFile() error = %v", err)
	}

	// Then generate resource files
	err = template.GenerateResourceFiles(ctx, fs, logger, name, name)
	if err != nil {
		t.Fatalf("GenerateResourceFiles() error = %v", err)
	}

	// Verify workflow file was created
	exists, err := afero.Exists(fs, filepath.Join(name, "workflow.pkl"))
	if err != nil {
		t.Fatalf("Error checking workflow file existence: %v", err)
	}
	if !exists {
		t.Error("Expected workflow.pkl file to be created")
	}

	// Verify resource files were created
	resourceDir := filepath.Join(name, "resources")
	files, err := afero.ReadDir(fs, resourceDir)
	if err != nil {
		t.Fatalf("Error reading resource directory: %v", err)
	}

	// Check that we have the expected number of files
	expectedFiles := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	require.Len(t, files, len(expectedFiles), "Unexpected number of resource files")

	// Check each expected file exists
	for _, expectedFile := range expectedFiles {
		exists, err := afero.Exists(fs, filepath.Join(resourceDir, expectedFile))
		require.NoError(t, err)
		assert.True(t, exists, "Expected file %s does not exist", expectedFile)
	}
}

func TestPrintWithDots(_ *testing.T) {
	template.PrintWithDots("test")
}

func TestSchemaVersionInTemplates(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("WorkflowTemplateWithSchemaVersion", func(t *testing.T) {
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)
		defer fs.RemoveAll(tempDir)

		err = template.GenerateWorkflowFile(ctx, fs, logger, tempDir, "testAgent")
		require.NoError(t, err)

		content, err := afero.ReadFile(fs, filepath.Join(tempDir, "workflow.pkl"))
		require.NoError(t, err)

		// Verify that the schema version is included in the template
		assert.Contains(t, string(content), fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.Version(ctx)))
	})

	t.Run("ResourceTemplateWithSchemaVersion", func(t *testing.T) {
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)
		defer fs.RemoveAll(tempDir)

		err = template.GenerateResourceFiles(ctx, fs, logger, tempDir, "testAgent")
		require.NoError(t, err)

		// Check all generated resource files
		files, err := afero.ReadDir(fs, filepath.Join(tempDir, "resources"))
		require.NoError(t, err)

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".pkl" {
				continue
			}

			content, err := afero.ReadFile(fs, filepath.Join(tempDir, "resources", file.Name()))
			require.NoError(t, err)

			// Verify that the schema version is included in each template
			assert.Contains(t, string(content), fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.Version(ctx)))
		}
	})
}

func TestFileGenerationEdgeCases(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	tests := []struct {
		name          string
		agentName     string
		baseDir       string
		expectedError bool
	}{
		{
			name:          "EmptyAgentName",
			agentName:     "",
			baseDir:       "",
			expectedError: true,
		},
		{
			name:          "SpacesInAgentName",
			agentName:     "invalid name",
			baseDir:       "",
			expectedError: true,
		},
		{
			name:          "ValidWithBaseDir",
			agentName:     "test-agent",
			baseDir:       "base",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First, generate the workflow file
			err := template.GenerateWorkflowFile(ctx, fs, logger, filepath.Join(tt.baseDir, tt.agentName), tt.agentName)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Then generate resource files
			err = template.GenerateResourceFiles(ctx, fs, logger, filepath.Join(tt.baseDir, tt.agentName), tt.agentName)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// For valid cases, verify the files were created in the correct location
			basePath := filepath.Join(tt.baseDir, tt.agentName)
			exists, err := afero.Exists(fs, filepath.Join(basePath, "workflow.pkl"))
			require.NoError(t, err)
			assert.True(t, exists)

			// Check resource directory
			resourceDir := filepath.Join(basePath, "resources")
			exists, err = afero.Exists(fs, resourceDir)
			require.NoError(t, err)
			assert.True(t, exists)
		})
	}
}

func TestCreateDirectoryEdgeCases(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("CreateDirectoryWithInvalidPath", func(t *testing.T) {
		path := ""
		err := template.CreateDirectory(fs, logger, path)
		assert.Error(t, err, "Expected error for empty path")
	})

	t.Run("CreateDirectoryWithReadOnlyParent", func(t *testing.T) {
		// Simulate a read-only parent directory by using a read-only FS
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		path := filepath.Join(tempDir, "test/readonly/child")
		err := template.CreateDirectory(readOnlyFs, logger, path)
		assert.Error(t, err, "Expected error when parent directory is read-only")
	})
}

func TestCreateFileEdgeCases(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("CreateFileWithInvalidPath", func(t *testing.T) {
		path := ""
		content := "test content"
		err := template.CreateFile(fs, logger, path, content)
		assert.Error(t, err, "Expected error for empty path")
	})

	t.Run("CreateFileInNonExistentDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "nonexistent/dir/file.txt")
		content := "test content"
		err := template.CreateFile(fs, logger, path, content)
		require.NoError(t, err, "Expected no error, should create parent directories")
		exists, err := afero.Exists(fs, path)
		require.NoError(t, err)
		assert.True(t, exists, "File should exist")
	})

	t.Run("CreateFileWithEmptyContent", func(t *testing.T) {
		path := filepath.Join(tempDir, "empty.txt")
		content := ""
		err := template.CreateFile(fs, logger, path, content)
		require.NoError(t, err, "Expected no error for empty content")
		data, err := afero.ReadFile(fs, path)
		require.NoError(t, err)
		assert.Empty(t, string(data), "File content should be empty")
	})
}

func TestMain(m *testing.M) {
	// Set non-interactive mode
	os.Setenv("NON_INTERACTIVE", "1")

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// validateAgentName should reject empty, whitespace, and names with spaces, and accept valid names.
func TestValidateAgentNameExtra(t *testing.T) {
	require.Error(t, template.ValidateAgentName(""))
	require.Error(t, template.ValidateAgentName("   "))
	require.Error(t, template.ValidateAgentName("bad name"))
	require.NoError(t, template.ValidateAgentName("goodName"))
}

// promptForAgentName should return default in non-interactive mode.
func TestPromptForAgentNameNonInteractiveExtra(t *testing.T) {
	t.Setenv("NON_INTERACTIVE", "1")

	name, err := template.PromptForAgentName()
	require.NoError(t, err)
	require.Equal(t, "test-agent", name)
}

// TestCreateDirectoryAndFile verifies createDirectory and createFile behavior.
func TestCreateDirectoryAndFileExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	t.Setenv("NON_INTERACTIVE", "1")

	// Test createDirectory
	err := template.CreateDirectory(fs, logger, "dir/subdir")
	require.NoError(t, err)
	exists, err := afero.DirExists(fs, "dir/subdir")
	require.NoError(t, err)
	require.True(t, exists)

	// Test createFile
	err = template.CreateFile(fs, logger, "dir/subdir/file.txt", "content")
	require.NoError(t, err)
	data, err := afero.ReadFile(fs, "dir/subdir/file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("content"), data)
}

// TestLoadTemplateFromDisk verifies loadTemplate reads from TEMPLATE_DIR when set.
func TestLoadTemplateFromDiskExtra(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("TEMPLATE_DIR", tmpDir)
	defer t.Setenv("TEMPLATE_DIR", os.Getenv("TEMPLATE_DIR"))

	// Write a simple template file
	templateName := "foo.tmpl"
	content := "Hello {{.Name}}"
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, templateName), []byte(content), 0o644))

	out, err := template.LoadTemplate(templateName, map[string]string{"Name": "Bob"})
	require.NoError(t, err)
	require.Equal(t, "Hello Bob", out)
}

// TestGenerateWorkflowFile covers error for invalid name and success path.
func TestGenerateWorkflowFileExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	t.Setenv("NON_INTERACTIVE", "1")
	defer t.Setenv("NON_INTERACTIVE", os.Getenv("NON_INTERACTIVE"))

	// Invalid name should return error
	err := template.GenerateWorkflowFile(context.Background(), fs, logger, "outdir", "bad name")
	require.Error(t, err)

	// Setup disk template
	tmpDir := t.TempDir()
	t.Setenv("TEMPLATE_DIR", tmpDir)
	defer t.Setenv("TEMPLATE_DIR", os.Getenv("TEMPLATE_DIR"))
	tmplPath := filepath.Join(tmpDir, "workflow.pkl")
	require.NoError(t, os.WriteFile(tmplPath, []byte("X:{{.Name}}"), 0o644))

	// Successful generation
	mainDir := "agentdir"
	err = template.GenerateWorkflowFile(context.Background(), fs, logger, mainDir, "Agent")
	require.NoError(t, err)
	output, err := afero.ReadFile(fs, filepath.Join(mainDir, "workflow.pkl"))
	require.NoError(t, err)
	require.Equal(t, "X:Agent", string(output))
}

// TestGenerateResourceFiles covers error for invalid name and success path.
func TestGenerateResourceFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	t.Setenv("NON_INTERACTIVE", "1")
	defer t.Setenv("NON_INTERACTIVE", os.Getenv("NON_INTERACTIVE"))

	// Invalid name
	err := template.GenerateResourceFiles(context.Background(), fs, logger, "outdir", "bad name")
	require.Error(t, err)

	// Setup disk templates directory matching embedded FS
	tmpDir := t.TempDir()
	t.Setenv("TEMPLATE_DIR", tmpDir)
	defer t.Setenv("TEMPLATE_DIR", os.Getenv("TEMPLATE_DIR"))
	// Create .pkl template files for each embedded resource (skip workflow.pkl)
	templateFiles := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, name := range templateFiles {
		path := filepath.Join(tmpDir, name)
		content := fmt.Sprintf("CONTENT:%s:{{.Name}}", name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}

	mainDir := "agentdir2"
	err = template.GenerateResourceFiles(context.Background(), fs, logger, mainDir, "Agent")
	require.NoError(t, err)

	// client.pkl should be created with expected content
	clientPath := filepath.Join(mainDir, "resources", "client.pkl")
	output, err := afero.ReadFile(fs, clientPath)
	require.NoError(t, err)
	require.Equal(t, "CONTENT:client.pkl:Agent", string(output))
	// workflow.pkl should be skipped
	exists, err := afero.Exists(fs, filepath.Join(mainDir, "resources", "workflow.pkl"))
	require.NoError(t, err)
	require.False(t, exists)
}

func TestValidateAgentNameSimple(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"", true},
		{"foo bar", true},
		{"valid", false},
	}

	for _, c := range cases {
		err := template.ValidateAgentName(c.name)
		if c.wantErr && err == nil {
			t.Fatalf("expected error for %q, got nil", c.name)
		}
		if !c.wantErr && err != nil {
			t.Fatalf("unexpected error for %q: %v", c.name, err)
		}
	}
}

func TestLoadTemplateEmbeddedBasic(t *testing.T) {
	data := map[string]string{
		"Header": "header-line",
		"Name":   "myagent",
	}
	out, err := template.LoadTemplate("workflow.pkl", data)
	if err != nil {
		t.Fatalf("loadTemplate error: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected non-empty output")
	}
	if !contains(out, "header-line") || !contains(out, "myagent") {
		t.Fatalf("output does not contain expected replacements: %s", out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (contains(s[1:], substr) || s[:len(substr)] == substr))
}

func TestGenerateAgentEndToEndExtra(t *testing.T) {
	// Ensure non-interactive to avoid slow sleeps.
	old := os.Getenv("NON_INTERACTIVE")
	t.Setenv("NON_INTERACTIVE", "1")
	defer t.Setenv("NON_INTERACTIVE", old)

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	baseDir := "/tmp"
	agentName := "client" // corresponds to existing embedded template client.pkl

	if err := template.GenerateAgent(ctx, fs, logger, baseDir, agentName); err != nil {
		t.Fatalf("GenerateAgent error: %v", err)
	}

	// Verify that workflow file was created
	wfPath := baseDir + "/" + agentName + "/workflow.pkl"
	if ok, _ := afero.Exists(fs, wfPath); !ok {
		t.Fatalf("expected workflow.pkl to exist at %s", wfPath)
	}

	// Verify that at least one resource file exists
	resPath := baseDir + "/" + agentName + "/resources/client.pkl"
	if ok, _ := afero.Exists(fs, resPath); !ok {
		t.Fatalf("expected resource file %s to exist", resPath)
	}
}

// TestPromptForAgentNameNonInteractive verifies that the helper returns the fixed
// value when NON_INTERACTIVE is set, without awaiting user input.
func TestPromptForAgentNameNonInteractive(t *testing.T) {
	orig := os.Getenv("NON_INTERACTIVE")
	t.Setenv("NON_INTERACTIVE", "1")
	defer t.Setenv("NON_INTERACTIVE", orig)

	name, err := template.PromptForAgentName()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "test-agent" {
		t.Errorf("expected 'test-agent', got %q", name)
	}
}

// TestGenerateAgentBasic creates an agent in a mem-fs and ensures that the core files
// are generated without touching the real filesystem.
func TestGenerateAgentBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	baseDir := "/workspace"
	agentName := "client"

	if err := template.GenerateAgent(ctx, fs, logger, baseDir, agentName); err != nil {
		t.Fatalf("GenerateAgent failed: %v", err)
	}

	// Expected files
	expects := []string{
		filepath.Join(baseDir, agentName, "workflow.pkl"),
		filepath.Join(baseDir, agentName, "resources", "client.pkl"),
		filepath.Join(baseDir, agentName, "resources", "exec.pkl"),
	}

	for _, path := range expects {
		exists, err := afero.Exists(fs, path)
		if err != nil {
			t.Fatalf("error checking %s: %v", path, err)
		}
		if !exists {
			t.Errorf("expected file %s to be generated", path)
		}
	}
}

func TestGenerateDockerfileFromTemplate(t *testing.T) {
	templateData := map[string]interface{}{
		"ImageVersion":     "latest",
		"SchemaVersion":    "1.0.0",
		"HostIP":           "127.0.0.1",
		"OllamaPortNum":    "11434",
		"KdepsHost":        "127.0.0.1:3000",
		"ArgsSection":      "ARG TEST=value",
		"EnvsSection":      "ENV CUSTOM=test",
		"PkgSection":       "RUN apt-get install -y git",
		"PythonPkgSection": "RUN pip install numpy",
		"CondaPkgSection":  "RUN conda install pytorch",
		"AnacondaVersion":  version.DefaultAnacondaVersion,
		"PklVersion":       version.DefaultPklVersion,
		"KdepsVersion":     version.DefaultKdepsInstallVersion,
		"Timezone":         "UTC",
		"ExposedPort":      "8080",
		"Environment":      "dev",
		"AgentName":        "test-agent",
		"InstallAnaconda":  true,
		"DevBuildMode":     false,
		"ApiServerMode":    true,
	}

	content, err := template.GenerateDockerfileFromTemplate(templateData)
	require.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify key components are present
	assert.Contains(t, content, "FROM ollama/ollama:latest")
	assert.Contains(t, content, "ENV SCHEMA_VERSION=1.0.0")
	assert.Contains(t, content, "ENV OLLAMA_HOST=127.0.0.1:11434")
	assert.Contains(t, content, "ENV KDEPS_HOST=127.0.0.1:3000")
	assert.Contains(t, content, "ARG TEST=value")
	assert.Contains(t, content, "ENV CUSTOM=test")
	assert.Contains(t, content, "RUN apt-get install -y git")
	assert.Contains(t, content, "RUN pip install numpy")
	assert.Contains(t, content, "RUN conda install pytorch")
	assert.Contains(t, content, "ENV TZ=UTC")
	assert.Contains(t, content, "EXPOSE 8080")
	assert.Contains(t, content, "ENTRYPOINT [\"/bin/kdeps\"]")

	// Verify conditional logic
	assert.Contains(t, content, "anaconda-linux-"+version.DefaultAnacondaVersion)           // Anaconda should be installed
	assert.Contains(t, content, "curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps") // Production mode
}

func TestGenerateDockerfileFromTemplate_DevMode(t *testing.T) {
	templateData := map[string]interface{}{
		"ImageVersion":     "latest",
		"SchemaVersion":    "1.0.0",
		"HostIP":           "127.0.0.1",
		"OllamaPortNum":    "11434",
		"KdepsHost":        "127.0.0.1:3000",
		"ArgsSection":      "",
		"EnvsSection":      "",
		"PkgSection":       "",
		"PythonPkgSection": "",
		"CondaPkgSection":  "",
		"AnacondaVersion":  version.DefaultAnacondaVersion,
		"PklVersion":       version.DefaultPklVersion,
		"KdepsVersion":     version.DefaultKdepsInstallVersion,
		"Timezone":         "UTC",
		"ExposedPort":      "",
		"Environment":      "dev",
		"AgentName":        "test-agent",
		"InstallAnaconda":  false,
		"DevBuildMode":     true,
		"ApiServerMode":    false,
	}

	content, err := template.GenerateDockerfileFromTemplate(templateData)
	require.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify dev mode specific content
	assert.Contains(t, content, "RUN cp /cache/kdeps /bin/kdeps")
	assert.Contains(t, content, "RUN chmod a+x /bin/kdeps")

	// Verify API server mode is off
	assert.NotContains(t, content, "EXPOSE")

	// Verify Anaconda is not installed
	assert.NotContains(t, content, "anaconda-linux")
}

func TestTemplateWithSchemaAssets(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("GenerateWorkflowWithAssets", func(t *testing.T) {
		// Setup PKL workspace with embedded schema files
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup() // Important: clean up temp files

		// Verify the workspace has the schema files we expect
		files, err := workspace.ListFiles()
		require.NoError(t, err)
		require.Contains(t, files, "Workflow.pkl")
		require.Contains(t, files, "Resource.pkl")

		// Generate workflow using the workspace
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)

		err = template.GenerateWorkflowFile(ctx, fs, logger, tempDir, "testAgent")
		require.NoError(t, err)

		// Read the generated workflow file
		content, err := afero.ReadFile(fs, filepath.Join(tempDir, "workflow.pkl"))
		require.NoError(t, err)

		// Verify it contains the correct schema reference
		workflowContent := string(content)
		assert.Contains(t, workflowContent, fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.Version(ctx)))
		assert.Contains(t, workflowContent, "AgentID = \"testAgent\"")

		// Get the actual Workflow.pkl content from assets for comparison
		workflowSchema, err := assets.GetPKLFileAsString("Workflow.pkl")
		require.NoError(t, err)
		assert.NotEmpty(t, workflowSchema)
		assert.Contains(t, workflowSchema, "AgentID: String")

		t.Logf("Workspace directory: %s", workspace.Directory)
		t.Logf("Available schema files: %v", files)
	})

	t.Run("ValidateTemplateAgainstSchema", func(t *testing.T) {
		// Setup PKL workspace
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup()

		// Test all resource templates
		resourceTemplates := []string{"exec.pkl", "llm.pkl", "client.pkl", "python.pkl", "response.pkl"}

		for _, templateName := range resourceTemplates {
			t.Run(fmt.Sprintf("Template_%s", templateName), func(t *testing.T) {
				tempDir, err := afero.TempDir(fs, "", "test")
				require.NoError(t, err)

				// Generate the resource file
				err = template.GenerateSpecificAgentFile(ctx, fs, logger, tempDir, strings.TrimSuffix(templateName, ".pkl"))
				require.NoError(t, err)

				// Read the generated file
				content, err := afero.ReadFile(fs, filepath.Join(tempDir, "resources", templateName))
				require.NoError(t, err)

				// Verify it references the correct schema
				resourceContent := string(content)
				assert.Contains(t, resourceContent, fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.Version(ctx)))

				// Verify it has the new v0.4.6 properties
				assert.Contains(t, resourceContent, "PostflightCheck")
				assert.Contains(t, resourceContent, "Retry = false")
				assert.Contains(t, resourceContent, "RetryTimes = 3")

				// Get the actual Resource.pkl schema for validation
				resourceSchema, err := assets.GetPKLFileAsString("Resource.pkl")
				require.NoError(t, err)
				assert.Contains(t, resourceSchema, "PostflightCheck: ValidationCheck?")
			})
		}
	})

	t.Run("ListAllEmbeddedSchemaFiles", func(t *testing.T) {
		// Test that we can access all embedded PKL files
		files, err := assets.ListPKLFiles()
		require.NoError(t, err)
		require.NotEmpty(t, files)

		// Verify expected schema files are present
		expectedFiles := []string{
			"Workflow.pkl",
			"Resource.pkl",
			"LLM.pkl",
			"APIServer.pkl",
			"Project.pkl",
			"Docker.pkl",
		}

		for _, expected := range expectedFiles {
			assert.Contains(t, files, expected, "Expected schema file %s should be available", expected)

			// Test that we can read each file
			content, err := assets.GetPKLFileAsString(expected)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "Schema file %s should have content", expected)
		}

		t.Logf("Available schema files: %v", files)
	})

	t.Run("ValidatePKLFilesIntegrity", func(t *testing.T) {
		// Test the built-in validation function
		err := assets.ValidatePKLFiles()
		require.NoError(t, err, "All expected PKL files should be present in assets")
	})
}
