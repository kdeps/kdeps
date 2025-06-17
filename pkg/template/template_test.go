package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Save the original EditPkl function
var originalEditPkl = texteditor.EditPkl

func setNonInteractive(t *testing.T) func() {
	t.Helper()
	oldValue := os.Getenv("NON_INTERACTIVE")
	os.Setenv("NON_INTERACTIVE", "1")
	return func() {
		os.Setenv("NON_INTERACTIVE", oldValue)
	}
}

func TestValidateAgentName(t *testing.T) {
	// Test case 1: Valid agent name
	err := validateAgentName("test-agent")
	if err != nil {
		t.Errorf("Expected no error for valid agent name, got: %v", err)
	}

	// Test case 2: Empty agent name
	err = validateAgentName("")
	if err == nil {
		t.Error("Expected error for empty agent name, got nil")
	}

	// Test case 3: Agent name with spaces
	err = validateAgentName("test agent")
	if err == nil {
		t.Error("Expected error for agent name with spaces, got nil")
	}

	t.Log("validateAgentName tests passed")
}

func TestCreateDirectoryNew(t *testing.T) {
	// Test case: Create directory with in-memory FS
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	path := "/test/dir"
	err := createDirectory(fs, logger, path)
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
	path := "/test/file.txt"
	content := "test content"
	err := createFile(fs, logger, path, content)
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
	os.Setenv("NON_INTERACTIVE", "1")
	defer os.Unsetenv("NON_INTERACTIVE")
	name, err := promptForAgentName()
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
		err := createDirectory(fs, logger, path)

		assert.NoError(t, err)
		exists, err := afero.DirExists(fs, path)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("CreateNestedDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/nested/deep/directory")
		err := createDirectory(fs, logger, path)

		assert.NoError(t, err)
		exists, err := afero.DirExists(fs, path)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("CreateExistingDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "test/existing")
		err := fs.MkdirAll(path, 0o755)
		require.NoError(t, err)

		err = createDirectory(fs, logger, path)
		assert.NoError(t, err)
	})

	t.Run("CreateDirectoryWithError", func(t *testing.T) {
		// Use a read-only filesystem to force an error
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		path := filepath.Join(tempDir, "test/readonly")
		err := createDirectory(readOnlyFs, logger, path)

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

		err := createFile(fs, logger, path, content)

		assert.NoError(t, err)
		exists, err := afero.Exists(fs, path)
		assert.NoError(t, err)
		assert.True(t, exists)

		data, err := afero.ReadFile(fs, path)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("CreateFileInNestedDirectory", func(t *testing.T) {
		// Create directory first
		dir := filepath.Join(tempDir, "test/nested/dir")
		err := fs.MkdirAll(dir, 0o755)
		require.NoError(t, err)

		path := filepath.Join(dir, "file.txt")
		content := "nested file content"

		err = createFile(fs, logger, path, content)

		assert.NoError(t, err)
		data, err := afero.ReadFile(fs, path)
		assert.NoError(t, err)
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
		err = createFile(fs, logger, path, newContent)

		assert.NoError(t, err)
		data, err := afero.ReadFile(fs, path)
		assert.NoError(t, err)
		assert.Equal(t, newContent, string(data))
	})
}

func TestLoadTemplate(t *testing.T) {
	data := map[string]string{
		"Header": "test-header",
		"Name":   "test-name",
	}

	content, err := loadTemplate("workflow.pkl", data)
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

		content, err := loadTemplate(templatePath, data)

		assert.NoError(t, err)
		assert.NotEmpty(t, content)
		// Verify that the template still loads even with empty data
		assert.Contains(t, content, "name =")
		assert.Contains(t, content, "description =")
	})

	t.Run("TemplateWithMissingVariables", func(t *testing.T) {
		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header",
			// Name is missing
		}

		content, err := loadTemplate(templatePath, data)

		assert.NoError(t, err)
		assert.NotEmpty(t, content)
		// Verify that the template still loads but with empty variables
		assert.Contains(t, content, "test header")
		assert.Contains(t, content, "name =")
	})

	t.Run("TemplateWithSpecialCharacters", func(t *testing.T) {
		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header with special chars: !@#$%^&*()",
			"Name":   "test-agent_with.special@chars",
		}

		content, err := loadTemplate(templatePath, data)

		assert.NoError(t, err)
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

	content, err := loadTemplate("workflow.pkl", data)
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

	err := GenerateResourceFiles(fs, ctx, logger, mainDir, name)
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
	assert.Equal(t, len(expectedFiles), len(files), "Unexpected number of resource files")

	// Check each expected file exists
	for _, expectedFile := range expectedFiles {
		exists, err := afero.Exists(fs, filepath.Join(resourceDir, expectedFile))
		assert.NoError(t, err)
		assert.True(t, exists, "Expected file %s does not exist", expectedFile)
	}
}

func TestGenerateSpecificAgentFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	mainDir := "test-agent"
	name := "client"

	err := GenerateSpecificAgentFile(fs, ctx, logger, mainDir, name)
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
	err := GenerateWorkflowFile(fs, ctx, logger, name, name)
	if err != nil {
		t.Fatalf("GenerateWorkflowFile() error = %v", err)
	}

	// Then generate resource files
	err = GenerateResourceFiles(fs, ctx, logger, name, name)
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
	assert.Equal(t, len(expectedFiles), len(files), "Unexpected number of resource files")

	// Check each expected file exists
	for _, expectedFile := range expectedFiles {
		exists, err := afero.Exists(fs, filepath.Join(resourceDir, expectedFile))
		assert.NoError(t, err)
		assert.True(t, exists, "Expected file %s does not exist", expectedFile)
	}
}

func TestPrintWithDots(t *testing.T) {
	printWithDots("test")
}

func TestSchemaVersionInTemplates(t *testing.T) {

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("WorkflowTemplateWithSchemaVersion", func(t *testing.T) {
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)
		defer fs.RemoveAll(tempDir)

		err = GenerateWorkflowFile(fs, ctx, logger, tempDir, "testAgent")
		require.NoError(t, err)

		content, err := afero.ReadFile(fs, filepath.Join(tempDir, "workflow.pkl"))
		require.NoError(t, err)

		// Verify that the schema version is included in the template
		assert.Contains(t, string(content), fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"`, schema.SchemaVersion(ctx)))
	})

	t.Run("ResourceTemplateWithSchemaVersion", func(t *testing.T) {
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)
		defer fs.RemoveAll(tempDir)

		err = GenerateResourceFiles(fs, ctx, logger, tempDir, "testAgent")
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
			assert.Contains(t, string(content), fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Resource.pkl"`, schema.SchemaVersion(ctx)))
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
			err := GenerateWorkflowFile(fs, ctx, logger, filepath.Join(tt.baseDir, tt.agentName), tt.agentName)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Then generate resource files
			err = GenerateResourceFiles(fs, ctx, logger, filepath.Join(tt.baseDir, tt.agentName), tt.agentName)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// For valid cases, verify the files were created in the correct location
			basePath := filepath.Join(tt.baseDir, tt.agentName)
			exists, err := afero.Exists(fs, filepath.Join(basePath, "workflow.pkl"))
			assert.NoError(t, err)
			assert.True(t, exists)

			// Check resource directory
			resourceDir := filepath.Join(basePath, "resources")
			exists, err = afero.Exists(fs, resourceDir)
			assert.NoError(t, err)
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
		err := createDirectory(fs, logger, path)
		assert.Error(t, err, "Expected error for empty path")
	})

	t.Run("CreateDirectoryWithReadOnlyParent", func(t *testing.T) {
		// Simulate a read-only parent directory by using a read-only FS
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		path := filepath.Join(tempDir, "test/readonly/child")
		err := createDirectory(readOnlyFs, logger, path)
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
		err := createFile(fs, logger, path, content)
		assert.Error(t, err, "Expected error for empty path")
	})

	t.Run("CreateFileInNonExistentDirectory", func(t *testing.T) {
		path := filepath.Join(tempDir, "nonexistent/dir/file.txt")
		content := "test content"
		err := createFile(fs, logger, path, content)
		assert.NoError(t, err, "Expected no error, should create parent directories")
		exists, err := afero.Exists(fs, path)
		assert.NoError(t, err)
		assert.True(t, exists, "File should exist")
	})

	t.Run("CreateFileWithEmptyContent", func(t *testing.T) {
		path := filepath.Join(tempDir, "empty.txt")
		content := ""
		err := createFile(fs, logger, path, content)
		assert.NoError(t, err, "Expected no error for empty content")
		data, err := afero.ReadFile(fs, path)
		assert.NoError(t, err)
		assert.Equal(t, "", string(data), "File content should be empty")
	})
}

func TestMain(m *testing.M) {
	// Save the original EditPkl function
	originalEditPkl := texteditor.EditPkl
	// Replace with mock for testing
	texteditor.EditPkl = texteditor.MockEditPkl
	// Set non-interactive mode
	os.Setenv("NON_INTERACTIVE", "1")

	// Run tests
	code := m.Run()

	// Restore original function
	texteditor.EditPkl = originalEditPkl

	os.Exit(code)
}
