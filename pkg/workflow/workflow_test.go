package workflow

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	assets "github.com/kdeps/schema/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWorkflow(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := LoadWorkflow(ctx, "nonexistent.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("InvalidWorkflowFile", func(t *testing.T) {
		// Create a temporary file with invalid PKL content
		tmpFile := t.TempDir() + "/invalid.pkl"
		err := os.WriteFile(tmpFile, []byte("invalid pkl content"), 0o644)
		require.NoError(t, err)

		_, err = LoadWorkflow(ctx, tmpFile, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidWorkflowFile", func(t *testing.T) {
		// Setup PKL workspace with embedded schema files
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup()

		// Create a temporary file with valid PKL content using schema assets
		tmpFile := t.TempDir() + "/valid.pkl"
		validContent := fmt.Sprintf(`amends "%s"

AgentID = "testworkflow"
Version = "1.0.0"
Description = "Test workflow"
TargetActionID = "testaction"
Settings {
  APIServerMode = false
  WebServerMode = false
  RateLimitMax = 100
  Environment = "dev"
}`, workspace.GetImportPath("Workflow.pkl"))
		err = os.WriteFile(tmpFile, []byte(validContent), 0o644)
		require.NoError(t, err)

		wf, err := LoadWorkflow(ctx, tmpFile, logger)
		assert.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Equal(t, "testworkflow", wf.GetAgentID())
		assert.Equal(t, "1.0.0", wf.GetVersion())
		assert.NotNil(t, wf.GetDescription())
		assert.Equal(t, "Test workflow", *wf.GetDescription())
		assert.Equal(t, "testaction", wf.GetTargetActionID())
	})
}

func TestWorkflowWithSchemaAssets(t *testing.T) {
	t.Run("ValidateWorkflowAgainstSchemaAssets", func(t *testing.T) {
		// Setup PKL workspace with all schema files
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup()

		// Get the actual Workflow.pkl schema content
		workflowSchema, err := assets.GetPKLFileAsString("Workflow.pkl")
		require.NoError(t, err)
		assert.NotEmpty(t, workflowSchema)

		// Verify the schema contains expected v0.4.0 properties
		assert.Contains(t, workflowSchema, "AgentID: String")
		assert.Contains(t, workflowSchema, "Description: String")
		assert.Contains(t, workflowSchema, "Website: Uri?")
		assert.Contains(t, workflowSchema, "Authors: Listing<String>?")
		assert.Contains(t, workflowSchema, "Documentation: Uri?")
		assert.Contains(t, workflowSchema, "Repository: Uri?")
		assert.Contains(t, workflowSchema, "HeroImage: String?")
		assert.Contains(t, workflowSchema, "AgentIcon: String?")
		assert.Contains(t, workflowSchema, "Version: String")
		assert.Contains(t, workflowSchema, "TargetActionID: String")
		assert.Contains(t, workflowSchema, "Workflows: Listing")
		assert.Contains(t, workflowSchema, "Settings: Project.Settings")

		t.Logf("Workflow schema validation successful")
		t.Logf("Schema contains %d characters", len(workflowSchema))
	})

	t.Run("ValidateProjectSettingsSchema", func(t *testing.T) {
		// Get the Project.pkl schema which contains Settings definition
		projectSchema, err := assets.GetPKLFileAsString("Project.pkl")
		require.NoError(t, err)
		assert.NotEmpty(t, projectSchema)

		// Verify it contains the new v0.4.0 Settings properties
		assert.Contains(t, projectSchema, "RateLimitMax: Int? = 100")
		assert.Contains(t, projectSchema, "Environment: BuildEnv? = \"dev\"")
		assert.Contains(t, projectSchema, "APIServerMode: Boolean? = false")
		assert.Contains(t, projectSchema, "WebServerMode: Boolean? = false")
		assert.Contains(t, projectSchema, "AgentSettings: Docker.DockerSettings")

		t.Logf("Project schema validation successful")
	})

	t.Run("ValidateResourceSchema", func(t *testing.T) {
		// Get the Resource.pkl schema
		resourceSchema, err := assets.GetPKLFileAsString("Resource.pkl")
		require.NoError(t, err)
		assert.NotEmpty(t, resourceSchema)

		// Verify it contains the new v0.4.0 Resource properties
		assert.Contains(t, resourceSchema, "ActionID: String")
		assert.Contains(t, resourceSchema, "Name: String")
		assert.Contains(t, resourceSchema, "Description: String")
		assert.Contains(t, resourceSchema, "Category: String")
		assert.Contains(t, resourceSchema, "PostflightCheck: ValidationCheck?")
		assert.Contains(t, resourceSchema, "AllowedHeaders: Listing<String>?")
		assert.Contains(t, resourceSchema, "AllowedParams: Listing<String>?")
		assert.Contains(t, resourceSchema, "APIResponse: APIServerResponse?")

		// Verify ValidationCheck class with retry properties
		assert.Contains(t, resourceSchema, "Retry: Boolean? = false")
		assert.Contains(t, resourceSchema, "RetryTimes: Int? = 3")

		t.Logf("Resource schema validation successful")
	})

	t.Run("CreateWorkflowWithAssetsImport", func(t *testing.T) {
		// Setup PKL workspace
		workspace, err := assets.SetupPKLWorkspaceInTmpDir()
		require.NoError(t, err)
		defer workspace.Cleanup()

		// Create a test workflow file that imports from the workspace
		tmpFile := t.TempDir() + "/assets_workflow.pkl"
		workflowContent := fmt.Sprintf(`import "%s" as Workflow

myWorkflow = new Workflow {
  AgentID = "assets-test"
  Description = "Testing with assets"
  Version = "1.0.0"
  TargetActionID = "test-action"
  Workflows {}
  Settings = new Workflow.Project.Settings {
    APIServerMode = true
    RateLimitMax = 50
    Environment = "dev"
    AgentSettings = new Workflow.Docker.DockerSettings {
      InstallAnaconda = false
    }
  }
}`, workspace.GetImportPath("Workflow.pkl"))

		err = os.WriteFile(tmpFile, []byte(workflowContent), 0o644)
		require.NoError(t, err)

		t.Logf("Created test workflow with assets import at: %s", tmpFile)
		t.Logf("Workspace directory: %s", workspace.Directory)
		t.Logf("Import path: %s", workspace.GetImportPath("Workflow.pkl"))

		// Note: We can't actually evaluate this with LoadWorkflow since it expects
		// the amends format, but this demonstrates how to use assets for PKL imports
	})
}
