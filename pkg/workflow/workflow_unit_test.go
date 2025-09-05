package workflow

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Unit tests for workflow functions
func TestLoadWorkflowFromFile(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidFile", func(t *testing.T) {
		ctx := context.Background()
		logger := logging.NewTestLogger()

		_, err := loadWorkflowFromFile(ctx, "/nonexistent/file.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("InvalidPKL", func(t *testing.T) {
		// Create a temporary file with invalid PKL content
		tmpDir, err := os.MkdirTemp("", "workflow_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		invalidContent := `invalid pkl content that will cause parsing error`

		workflowFile := filepath.Join(tmpDir, "invalid.pkl")
		err = os.WriteFile(workflowFile, []byte(invalidContent), 0o644)
		require.NoError(t, err)

		_, err = loadWorkflowFromFile(ctx, workflowFile, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})
}

func TestLoadWorkflowFromEmbeddedAssets(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidFile", func(t *testing.T) {
		ctx := context.Background()
		logger := logging.NewTestLogger()

		_, err := loadWorkflowFromEmbeddedAssets(ctx, "/nonexistent/file.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("InvalidPKL", func(t *testing.T) {
		// Create a temporary file with invalid PKL content
		tmpDir, err := os.MkdirTemp("", "workflow_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		invalidContent := `invalid pkl content that will cause parsing error`

		workflowFile := filepath.Join(tmpDir, "invalid.pkl")
		err = os.WriteFile(workflowFile, []byte(invalidContent), 0o644)
		require.NoError(t, err)

		_, err = loadWorkflowFromEmbeddedAssets(ctx, workflowFile, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})
}

func TestLoadWorkflow_EmbeddedAssets(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidFile", func(t *testing.T) {
		_, err := LoadWorkflow(ctx, "/nonexistent/file.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})
}
