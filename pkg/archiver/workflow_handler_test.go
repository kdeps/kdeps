package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWorkflow implements the minimal subset of the Workflow interface we need for testing.
type testWorkflow struct{}

func (m testWorkflow) GetAgentID() string               { return "test-agent" }
func (m testWorkflow) GetVersion() string               { return "1.0.0" }
func (m testWorkflow) GetDescription() string           { return "" }
func (m testWorkflow) GetWebsite() *string              { return nil }
func (m testWorkflow) GetAuthors() *[]string            { return nil }
func (m testWorkflow) GetDocumentation() *string        { return nil }
func (m testWorkflow) GetRepository() *string           { return nil }
func (m testWorkflow) GetHeroImage() *string            { return nil }
func (m testWorkflow) GetAgentIcon() *string            { return nil }
func (m testWorkflow) GetTargetActionID() string        { return "test-action" }
func (m testWorkflow) GetWorkflows() []string           { return nil }
func (m testWorkflow) GetSettings() pklProject.Settings { return pklProject.Settings{} }

func TestCompileProjectDoesNotModifyOriginalFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	projectDir := "/test-project"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a test project directory
	require.NoError(t, fs.MkdirAll(projectDir, 0o755))
	require.NoError(t, fs.MkdirAll(filepath.Join(projectDir, "resources"), 0o755))

	// Create a workflow file
	wfContent := `amends "package://schema.kdeps.com/core@0.4.0-dev#/Workflow.pkl"

Name = "test-agent"
Version = "1.0.0"
TargetActionID = "test-action"
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "workflow.pkl"), []byte(wfContent), 0o644))

	// Create a resource file
	resourceContent := `amends "package://schema.kdeps.com/core@0.4.0-dev#/Resource.pkl"

ActionID = "test-action"

Run {
  Exec {
    Command = "echo hello"
  }
}
`
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "resources", "test.pkl"), []byte(resourceContent), 0o644))

	// Create a test file that should not be modified
	testFileContent := "original content"
	testFilePath := filepath.Join(projectDir, "test.txt")
	require.NoError(t, afero.WriteFile(fs, testFilePath, []byte(testFileContent), 0o644))

	// Create a mock workflow
	wf := testWorkflow{}

	// Call CompileProject (this will fail due to missing Pkl binary, but we can test the file protection)
	CompileProject(fs, ctx, wf, kdepsDir, projectDir, env, logger)

	// The compilation will fail due to missing Pkl binary, but that's expected
	// The important thing is that our original files were not modified

	// Verify that the original test file was not modified
	content, err := afero.ReadFile(fs, testFilePath)
	require.NoError(t, err)
	assert.Equal(t, testFileContent, string(content), "Original project file should not be modified")

	// Verify that the original workflow file was not modified
	originalWfContent, err := afero.ReadFile(fs, filepath.Join(projectDir, "workflow.pkl"))
	require.NoError(t, err)
	assert.Equal(t, wfContent, string(originalWfContent), "Original workflow file should not be modified")

	// Verify that the original resource file was not modified
	originalResourceContent, err := afero.ReadFile(fs, filepath.Join(projectDir, "resources", "test.pkl"))
	require.NoError(t, err)
	assert.Equal(t, resourceContent, string(originalResourceContent), "Original resource file should not be modified")
}
