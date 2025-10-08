package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/assets"
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
		// Copy schema assets to temp directory for offline testing
		schemaDir, err := assets.CopyAssetsToTempDirWithConversion()
		require.NoError(t, err)
		defer os.RemoveAll(schemaDir)

		// Create a temporary file with valid PKL content using local schema
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "valid.pkl")
		validContent := `amends "` + filepath.Join(schemaDir, "Workflow.pkl") + `"

AgentID = "testworkflow"
Version = "1.0.0"
Description = "Test workflow"
TargetActionID = "testaction"
Settings {
  APIServerMode = true
  APIServer {
    HostIP = "127.0.0.1"
    PortNum = 3000
    Routes {
      new {
        Path = "/api/v1/test"
        Methods {
          "POST"
        }
      }
    }
    CORS {
      EnableCORS = true
      AllowOrigins {
        "http://localhost:8080"
      }
    }
  }
  AgentSettings {
    Timezone = "Etc/UTC"
    Models {
      "llama3.2:1b"
    }
    OllamaImageTag = "0.8.0"
  }
}`
		err = os.WriteFile(tmpFile, []byte(validContent), 0o644)
		require.NoError(t, err)

		wf, err := LoadWorkflow(ctx, tmpFile, logger)
		assert.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Equal(t, "testworkflow", wf.GetAgentID())
		assert.Equal(t, "1.0.0", wf.GetVersion())
		assert.Equal(t, "Test workflow", wf.GetDescription())
		assert.Equal(t, "testaction", wf.GetTargetActionID())
	})
}
