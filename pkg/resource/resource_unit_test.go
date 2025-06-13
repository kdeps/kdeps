package resource

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadResource(t *testing.T) {


	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("ValidResourceFile", func(t *testing.T) {
		// Create a temporary file on the real filesystem (PKL needs real files)
		tmpDir, err := os.MkdirTemp("", "resource_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a valid resource file content
		validContent := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "testaction"
name = "Test Action"
category = "test"
description = "Test resource"
run {
  APIResponse {
    success = true
    response {
      data {
        "test"
      }
    }
  }
}
`

		resourceFile := filepath.Join(tmpDir, "test.pkl")
		err = os.WriteFile(resourceFile, []byte(validContent), 0o644)
		require.NoError(t, err)

		// Test LoadResource - this should load the resource successfully
		resource, err := LoadResource(ctx, resourceFile, logger)

		// Should succeed and return a valid resource
		require.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "testaction", resource.ActionID)
		assert.Equal(t, "Test Action", resource.Name)
		assert.Equal(t, "test", resource.Category)
		assert.Equal(t, "Test resource", resource.Description)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		resourceFile := "/nonexistent/file.pkl"

		_, err := LoadResource(ctx, resourceFile, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading resource file")
	})

	t.Run("InvalidResourceFile", func(t *testing.T) {
		// Create a temporary file with invalid content
		tmpDir, err := os.MkdirTemp("", "resource_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create invalid PKL content
		invalidContent := `invalid pkl content that will cause parsing error`

		resourceFile := filepath.Join(tmpDir, "invalid.pkl")
		err = os.WriteFile(resourceFile, []byte(invalidContent), 0o644)
		require.NoError(t, err)

		_, err = LoadResource(ctx, resourceFile, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading resource file")
	})

	t.Run("NilLogger", func(t *testing.T) {
		resourceFile := "/test.pkl"

		// Test with nil logger - should panic
		assert.Panics(t, func() {
			LoadResource(ctx, resourceFile, nil)
		})
	})

	t.Run("EmptyResourceFile", func(t *testing.T) {
		// Create a temporary file with empty content
		tmpDir, err := os.MkdirTemp("", "resource_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		resourceFile := filepath.Join(tmpDir, "empty.pkl")
		err = os.WriteFile(resourceFile, []byte(""), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		// Empty file might actually load successfully or fail - either is acceptable
		// Just ensure it doesn't panic and we get consistent behavior
		if err != nil {
			assert.Contains(t, err.Error(), "error reading resource file")
			assert.Nil(t, resource)
		} else {
			// If it succeeds, we should have a valid resource
			assert.NotNil(t, resource)
		}
	})
}

// Test helper to ensure the logging calls work correctly
func TestLoadResourceLogging(t *testing.T) {


	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("LoggingBehavior", func(t *testing.T) {
		resourceFile := "/nonexistent/file.pkl"

		_, err := LoadResource(ctx, resourceFile, logger)

		// Should log debug and error messages
		assert.Error(t, err)
		// The actual logging verification would require a mock logger
		// but this tests that the function completes without panic
	})

	t.Run("SuccessLogging", func(t *testing.T) {
		// Create a temporary file on the real filesystem
		tmpDir, err := os.MkdirTemp("", "resource_test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Create a valid resource file content
		validContent := `amends "package://schema.kdeps.com/core@0.2.30#/Resource.pkl"

actionID = "testaction"
name = "Test Action"
category = "test"
description = "Test resource"
run {
  APIResponse {
    success = true
    response {
      data {
        "test"
      }
    }
  }
}
`

		resourceFile := filepath.Join(tmpDir, "test.pkl")
		err = os.WriteFile(resourceFile, []byte(validContent), 0o644)
		require.NoError(t, err)

		// This should test the successful debug logging path
		resource, err := LoadResource(ctx, resourceFile, logger)

		assert.NoError(t, err)
		assert.NotNil(t, resource)
	})
}
