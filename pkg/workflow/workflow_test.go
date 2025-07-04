package workflow

import (
	"context"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
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
		// Create a temporary file with valid PKL content
		tmpFile := t.TempDir() + "/valid.pkl"
		validContent := `amends "package://schema.kdeps.com/core@0.2.30#/Workflow.pkl"

name = "testworkflow"
version = "1.0.0"
description = "Test workflow"
targetActionID = "testaction"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new {
        path = "/api/v1/test"
        methods {
          "POST"
        }
      }
    }
    cors {
      enableCORS = true
      allowOrigins {
        "http://localhost:8080"
      }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models {
      "llama3.2:1b"
    }
    ollamaImageTag = "0.8.0"
  }
}`
		err := os.WriteFile(tmpFile, []byte(validContent), 0o644)
		require.NoError(t, err)

		wf, err := LoadWorkflow(ctx, tmpFile, logger)
		assert.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Equal(t, "testworkflow", wf.GetName())
		assert.Equal(t, "1.0.0", wf.GetVersion())
		assert.Equal(t, "Test workflow", wf.GetDescription())
		assert.Equal(t, "testaction", wf.GetTargetActionID())
	})
}
