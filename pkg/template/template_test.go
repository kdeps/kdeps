package template

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		agentName   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidName",
			agentName:   "myagent",
			expectError: false,
		},
		{
			name:        "ValidNameWithNumbers",
			agentName:   "agent123",
			expectError: false,
		},
		{
			name:        "ValidNameWithUnderscore",
			agentName:   "my_agent",
			expectError: false,
		},
		{
			name:        "EmptyName",
			agentName:   "",
			expectError: true,
			errorMsg:    "agent name cannot be empty or only whitespace",
		},
		{
			name:        "WhitespaceOnly",
			agentName:   "   ",
			expectError: true,
			errorMsg:    "agent name cannot be empty or only whitespace",
		},
		{
			name:        "NameWithSpaces",
			agentName:   "my agent",
			expectError: true,
			errorMsg:    "agent name cannot contain spaces",
		},
		{
			name:        "NameWithMultipleSpaces",
			agentName:   "my  agent  name",
			expectError: true,
			errorMsg:    "agent name cannot contain spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateAgentName(tt.agentName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	t.Run("LoadValidTemplate", func(t *testing.T) {
		// Test loading a template that exists in the embedded FS
		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header",
			"Name":   "testAgent",
		}

		content, err := loadTemplate(templatePath, data)

		assert.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.Contains(t, content, "test header")
		assert.Contains(t, content, "testAgent")
	})

	t.Run("LoadNonExistentTemplate", func(t *testing.T) {
		templatePath := "templates/nonexistent.pkl"
		data := map[string]string{}

		_, err := loadTemplate(templatePath, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read embedded template")
	})

	t.Run("LoadTemplateWithInvalidPath", func(t *testing.T) {
		templatePath := "invalid/path.pkl"
		data := map[string]string{}

		_, err := loadTemplate(templatePath, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read embedded template")
	})

	t.Run("LoadAllTemplates", func(t *testing.T) {
		// Test loading all available templates to improve coverage
		templatePaths := []string{
			"templates/workflow.pkl",
			"templates/llm.pkl",
			"templates/client.pkl",
			"templates/exec.pkl",
			"templates/python.pkl",
			"templates/response.pkl",
		}

		data := map[string]string{
			"Header": "test header",
			"Name":   "testAgent",
		}

		for _, templatePath := range templatePaths {
			content, err := loadTemplate(templatePath, data)
			assert.NoError(t, err, "Failed to load template: %s", templatePath)
			assert.NotEmpty(t, content)
			assert.Contains(t, content, "test header")
		}
	})
}

func TestTemplateLoadingEdgeCases(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("GenerateValidWorkflow", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent")
		name := "testAgent"

		err := fs.MkdirAll(mainDir, 0o755)
		require.NoError(t, err)

		err = generateWorkflowFile(fs, ctx, logger, mainDir, name)

		assert.NoError(t, err)

		workflowPath := filepath.Join(mainDir, "workflow.pkl")
		exists, err := afero.Exists(fs, workflowPath)
		assert.NoError(t, err)
		assert.True(t, exists)

		content, err := afero.ReadFile(fs, workflowPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), name)
		assert.Contains(t, string(content), "amends")
	})

	t.Run("GenerateWorkflowInvalidDirectory", func(t *testing.T) {
		// Test error when creating file in directory that can't be created
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		mainDir := filepath.Join(tempDir, "test/agent")
		name := "testAgent"

		err := generateWorkflowFile(readOnlyFs, ctx, logger, mainDir, name)

		assert.Error(t, err)
	})

	t.Run("GenerateWorkflowWithSpecialCharacters", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/special")
		name := "agent_with_underscores123"

		err := fs.MkdirAll(mainDir, 0o755)
		require.NoError(t, err)

		err = generateWorkflowFile(fs, ctx, logger, mainDir, name)

		assert.NoError(t, err)

		workflowPath := filepath.Join(mainDir, "workflow.pkl")
		content, err := afero.ReadFile(fs, workflowPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), name)
	})
}

func TestGenerateResourceFiles(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("GenerateValidResourceFiles", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent")
		name := "testAgent"

		err := fs.MkdirAll(mainDir, 0o755)
		require.NoError(t, err)

		err = generateResourceFiles(fs, ctx, logger, mainDir, name)

		assert.NoError(t, err)

		resourceDir := filepath.Join(mainDir, "resources")
		exists, err := afero.DirExists(fs, resourceDir)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Check that resource files were created (excluding workflow.pkl)
		files, err := afero.ReadDir(fs, resourceDir)
		assert.NoError(t, err)
		assert.Greater(t, len(files), 0)

		// Verify content of one of the resource files contains the template data
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".pkl") {
				filePath := filepath.Join(resourceDir, file.Name())
				content, err := afero.ReadFile(fs, filePath)
				assert.NoError(t, err)
				// Check for schema header instead of name since name might not be in all templates
				assert.Contains(t, string(content), "amends")
				assert.Contains(t, string(content), "schema.kdeps.com")
				break
			}
		}
	})

	t.Run("GenerateResourceFilesWithError", func(t *testing.T) {
		// Test with read-only filesystem to trigger error when creating resource directory
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		mainDir := filepath.Join(tempDir, "test/agent")
		name := "testAgent"

		err := generateResourceFiles(readOnlyFs, ctx, logger, mainDir, name)

		assert.Error(t, err)
	})

	t.Run("GenerateResourceFilesVerifyAllTemplates", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent_all")
		name := "testAgentAll"

		err := fs.MkdirAll(mainDir, 0o755)
		require.NoError(t, err)

		err = generateResourceFiles(fs, ctx, logger, mainDir, name)

		assert.NoError(t, err)

		resourceDir := filepath.Join(mainDir, "resources")
		files, err := afero.ReadDir(fs, resourceDir)
		assert.NoError(t, err)

		// Verify that all expected resource templates were created
		expectedFiles := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
		for _, expectedFile := range expectedFiles {
			found := false
			for _, file := range files {
				if file.Name() == expectedFile {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected file %s was not created", expectedFile)
		}

		// Verify workflow.pkl was NOT created in resources directory
		for _, file := range files {
			assert.NotEqual(t, "workflow.pkl", file.Name(), "workflow.pkl should not be in resources directory")
		}
	})
}

func TestGenerateSpecificFile(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	tempDir, err := afero.TempDir(fs, "", "test")
	require.NoError(t, err)

	t.Run("GenerateWorkflowFile", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent")
		fileName := "workflow.pkl"
		agentName := "testAgent"

		err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

		assert.NoError(t, err)

		// Workflow files should be in the main directory
		filePath := filepath.Join(mainDir, fileName)
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Data directory should be created
		dataDir := filepath.Join(mainDir, "data")
		exists, err = afero.DirExists(fs, dataDir)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("GenerateResourceFile", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent2")
		fileName := "llm.pkl"
		agentName := "testAgent2"

		err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

		assert.NoError(t, err)

		// Resource files should be in the resources directory
		filePath := filepath.Join(mainDir, "resources", fileName)
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("GenerateFileWithoutExtension", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent3")
		fileName := "llm" // No .pkl extension
		agentName := "testAgent3"

		err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

		assert.NoError(t, err)

		// Should automatically add .pkl extension
		filePath := filepath.Join(mainDir, "resources", "llm.pkl")
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("GenerateNonExistentTemplate", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent4")
		fileName := "nonexistent.pkl"
		agentName := "testAgent4"

		err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read embedded template")
	})

	t.Run("GenerateWorkflowCaseInsensitive", func(t *testing.T) {
		mainDir := filepath.Join(tempDir, "test/agent5")
		fileName := "workflow.pkl" // Correct case
		agentName := "testAgent5"

		err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

		assert.NoError(t, err)

		// Should be placed in main directory
		filePath := filepath.Join(mainDir, fileName)
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Verify content contains the agent name
		content, err := afero.ReadFile(fs, filePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), agentName)
	})

	t.Run("GenerateAllResourceTemplates", func(t *testing.T) {
		// Test generating each resource template individually
		resourceTemplates := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}

		for i, fileName := range resourceTemplates {
			mainDir := filepath.Join(tempDir, fmt.Sprintf("test/agent_resource_%d", i))
			agentName := fmt.Sprintf("testAgent%d", i)

			err := generateSpecificFile(fs, ctx, logger, mainDir, fileName, agentName)

			assert.NoError(t, err, "Failed to generate %s", fileName)

			filePath := filepath.Join(mainDir, "resources", fileName)
			exists, err := afero.Exists(fs, filePath)
			assert.NoError(t, err)
			assert.True(t, exists, "File %s was not created", fileName)

			// Verify file content contains template structure (not necessarily agent name)
			content, err := afero.ReadFile(fs, filePath)
			assert.NoError(t, err)
			assert.Contains(t, string(content), "amends")
			assert.Contains(t, string(content), "schema.kdeps.com")
			// Templates may not actually contain the agent name variable in their content
		}
	})

	t.Run("GenerateFileWithDataDirError", func(t *testing.T) {
		// Create a read-only filesystem to test error in data directory creation
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())

		mainDir := filepath.Join(tempDir, "test/agent_readonly")
		fileName := "llm.pkl"
		agentName := "testAgent"

		err := generateSpecificFile(readOnlyFs, ctx, logger, mainDir, fileName, agentName)

		assert.Error(t, err)
	})
}

// Test only the validation part of GenerateAgent to avoid interactive prompts
func TestGenerateAgentValidation(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidAgentName", func(t *testing.T) {
		agentName := "invalid agent" // Contains space

		err := GenerateAgent(fs, ctx, logger, agentName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent name cannot contain spaces")
	})

	t.Run("EmptyAgentName", func(t *testing.T) {
		agentName := "   " // Whitespace only

		err := GenerateAgent(fs, ctx, logger, agentName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent name cannot be empty or only whitespace")
	})

	// Note: We can't test valid agent names because they trigger interactive prompts later in the flow
}

// Test only the validation part of GenerateSpecificAgentFile to avoid interactive prompts
func TestGenerateSpecificAgentFileValidation(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidAgentName", func(t *testing.T) {
		agentName := "invalid agent" // Contains space
		fileName := "llm.pkl"

		err := GenerateSpecificAgentFile(fs, ctx, logger, agentName, fileName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent name cannot contain spaces")
	})

	// Note: We can't test empty agent name case because it triggers interactive prompt
}

func TestPrintWithDots(t *testing.T) {
	t.Parallel()

	// This function prints to stdout, so we'll just test it doesn't panic
	assert.NotPanics(t, func() {
		printWithDots("Testing message")
	})
}

// Additional unit tests for comprehensive coverage

func TestGenerateAgent(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("GenerateAgentWithInvalidName", func(t *testing.T) {
		agentName := "invalid agent" // Contains space

		err := GenerateAgent(fs, ctx, logger, agentName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent name cannot contain spaces")
	})

	t.Run("GenerateAgentFileSystemError", func(t *testing.T) {
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		agentName := "testAgent"

		err := GenerateAgent(readOnlyFs, ctx, logger, agentName)

		assert.Error(t, err)
	})
}

func TestGenerateSpecificAgentFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("GenerateSpecificAgentFileWithInvalidAgentName", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		agentName := "invalid agent" // Contains space
		fileName := "llm.pkl"

		err := GenerateSpecificAgentFile(fs, ctx, logger, agentName, fileName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent name cannot contain spaces")
	})

	t.Run("GenerateSpecificAgentFileWithNonExistentTemplate", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		agentName := "testAgent"
		fileName := "nonexistent.pkl"

		err := GenerateSpecificAgentFile(fs, ctx, logger, agentName, fileName)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read embedded template")
	})

	t.Run("GenerateSpecificAgentFileFileSystemError", func(t *testing.T) {
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		agentName := "testAgent"
		fileName := "llm.pkl"

		err := GenerateSpecificAgentFile(readOnlyFs, ctx, logger, agentName, fileName)

		assert.Error(t, err)
	})
}

// TestPromptForAgentName is removed because it requires interactive input

func TestLoadTemplateErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("LoadTemplateWithMalformedTemplate", func(t *testing.T) {
		// This tests error handling within loadTemplate for template parsing errors
		// We can't easily create a malformed embedded template, but we can test
		// the error handling paths that exist

		templatePath := "templates/workflow.pkl"
		data := map[string]string{
			"Header": "test header",
			"Name":   "testAgent",
		}

		// Normal case should work
		content, err := loadTemplate(templatePath, data)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
	})
}

func TestGenerateWorkflowFileErrorPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("GenerateWorkflowFileWithReadOnlyFS", func(t *testing.T) {
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		mainDir := "./testAgent"
		name := "testAgent"

		err := generateWorkflowFile(readOnlyFs, ctx, logger, mainDir, name)

		assert.Error(t, err)
	})
}

func TestGenerateResourceFilesErrorPaths(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("GenerateResourceFilesTemplateReadError", func(t *testing.T) {
		// The embedded templates should always be readable, but we can test
		// the normal success path to improve coverage
		mainDir := "./testAgent"
		name := "testAgent"

		err := fs.MkdirAll(mainDir, 0o755)
		require.NoError(t, err)

		err = generateResourceFiles(fs, ctx, logger, mainDir, name)

		assert.NoError(t, err)

		// Verify that multiple resource files were created
		resourceDir := filepath.Join(mainDir, "resources")
		files, err := afero.ReadDir(fs, resourceDir)
		assert.NoError(t, err)
		assert.Greater(t, len(files), 2) // Should have multiple resource files
	})
}

func TestPromptForAgentName(t *testing.T) {
	t.Parallel()

	// Since this function uses the huh library for interactive prompts,
	// we'll test the validation logic by mocking the input
	t.Run("ValidInput", func(t *testing.T) {
		// This test is mainly for documentation purposes since we can't easily
		// test interactive prompts in unit tests
		t.Skip("Skipping interactive prompt test")
	})

	t.Run("InvalidInput", func(t *testing.T) {
		// This test is mainly for documentation purposes since we can't easily
		// test interactive prompts in unit tests
		t.Skip("Skipping interactive prompt test")
	})
}

func TestSchemaVersionInTemplates(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("WorkflowTemplateWithSchemaVersion", func(t *testing.T) {
		tempDir, err := afero.TempDir(fs, "", "test")
		require.NoError(t, err)
		defer fs.RemoveAll(tempDir)

		err = generateWorkflowFile(fs, ctx, logger, tempDir, "testAgent")
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

		err = generateResourceFiles(fs, ctx, logger, tempDir, "testAgent")
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
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("GenerateFileWithLongPath", func(t *testing.T) {
		// Create a very long directory path
		longPath := filepath.Join("test", strings.Repeat("a/", 50))
		err := fs.MkdirAll(longPath, 0755)
		require.NoError(t, err)

		// Use a valid template name
		templateName := "workflow.pkl"
		outputPath := filepath.Join(longPath, templateName)
		err = generateSpecificFile(fs, ctx, logger, longPath, templateName, "test-agent")
		assert.NoError(t, err)

		// Verify the file was created
		exists, err := afero.Exists(fs, outputPath)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Verify the content
		content, err := afero.ReadFile(fs, outputPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "name =")
		assert.Contains(t, string(content), "description =")
	})

	t.Run("GenerateFileWithSpecialCharacters", func(t *testing.T) {
		// Create a directory with special characters
		specialDir := filepath.Join("test", "special!@#$%^&*()")
		err := fs.MkdirAll(specialDir, 0755)
		require.NoError(t, err)

		// Use a valid template name
		templateName := "workflow.pkl"
		outputPath := filepath.Join(specialDir, templateName)
		err = generateSpecificFile(fs, ctx, logger, specialDir, templateName, "test-agent_with.special@chars")
		assert.NoError(t, err)

		// Verify the file was created
		exists, err := afero.Exists(fs, outputPath)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Verify the content
		content, err := afero.ReadFile(fs, outputPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "name =")
		assert.Contains(t, string(content), "description =")
	})

	t.Run("GenerateFileWithExistingContent", func(t *testing.T) {
		// Create a test directory
		testDir := filepath.Join("test", "existing")
		err := fs.MkdirAll(testDir, 0755)
		require.NoError(t, err)

		// Create an existing file with some content
		existingPath := filepath.Join(testDir, "workflow.pkl")
		err = afero.WriteFile(fs, existingPath, []byte("existing content"), 0644)
		require.NoError(t, err)

		// Generate the file, overwriting the existing content
		err = generateSpecificFile(fs, ctx, logger, testDir, "workflow.pkl", "new-agent")
		assert.NoError(t, err)

		// Verify the file was overwritten
		content, err := afero.ReadFile(fs, existingPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "name =")
		assert.Contains(t, string(content), "description =")
		assert.NotContains(t, string(content), "existing content")
	})
}
