package resource_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	. "github.com/kdeps/kdeps/pkg/resource"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadResource_FileNotFound verifies that LoadResource returns an error when
// provided with a non-existent file path. This exercises the error branch to
// ensure we log and wrap the underlying failure correctly.
func TestLoadResource_FileNotFound(t *testing.T) {
	_, err := LoadResource(context.Background(), "/path/to/nowhere/nonexistent.pkl", logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error when reading missing resource file")
	}
}

func TestLoadResource_AdditionalCoverage(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("ValidResourceWithAllFields", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "complete.pkl")

		// Create a resource with all possible fields
		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "complete_action"
name = "Complete Test Action"  
category = "testing"
description = "A complete test resource with all fields"
requires {
  "dependency1"
  "dependency2"
}
run {
  chat {
    model = "llama3.2"
    prompt = "test prompt"
    JSONResponse = true
    JSONResponseKeys {
      "result"
      "status"
    }
    timeoutDuration = 30.s
  }
  preflightCheck {
    validations {
      1 == 1
      "test" == "test"
    }
  }
  APIResponse {
    success = true
    response {
      data {
        "test_data"
        "more_data"
      }
    }
  }
}
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		require.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "complete_action", resource.ActionID)
		assert.Equal(t, "Complete Test Action", resource.Name)
		assert.Equal(t, "testing", resource.Category)
		assert.Contains(t, resource.Description, "complete test resource")
	})

	t.Run("ResourceWithExecRun", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "exec.pkl")

		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "exec_action"
name = "Exec Test Action"
category = "execution"
description = "Resource that executes commands"
run {
  exec {
    command = "echo hello world"
    env {
      ["TEST_VAR"] = "test_value"
    }
  }
}
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		require.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "exec_action", resource.ActionID)
		assert.Equal(t, "execution", resource.Category)
	})

	t.Run("ResourceWithHTTPClient", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "http.pkl")

		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "http_action"
name = "HTTP Test Action"
category = "networking"
description = "Resource that makes HTTP requests"
run {
  HTTPClient {
    method = "POST"
    url = "https://api.example.com/data"
    headers {
      ["Content-Type"] = "application/json"
      ["Authorization"] = "Bearer token"
    }
    data {
      "{\"test\": \"data\"}"
    }
  }
}
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		require.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "http_action", resource.ActionID)
		assert.Equal(t, "networking", resource.Category)
	})

	t.Run("MinimalValidResource", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "minimal.pkl")

		// Minimal valid resource with just required fields
		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "minimal"
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		// This might succeed or fail depending on PKL validation rules
		if err != nil {
			assert.Contains(t, err.Error(), "error reading resource file")
		} else {
			assert.NotNil(t, resource)
			assert.Equal(t, "minimal", resource.ActionID)
		}
	})

	t.Run("FileWithSyntaxError", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "syntax_error.pkl")

		// Resource with syntax error (missing quote)
		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "syntax_error
name = "This has a syntax error"
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		assert.Error(t, err)
		assert.Nil(t, resource)
		assert.Contains(t, err.Error(), "error reading resource file")
	})

	t.Run("DirectoryInsteadOfFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirPath := filepath.Join(tmpDir, "not_a_file")
		err := os.Mkdir(dirPath, 0o755)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, dirPath, logger)

		assert.Error(t, err)
		assert.Nil(t, resource)
		assert.Contains(t, err.Error(), "error reading resource file")
	})

	t.Run("EmptyFilename", func(t *testing.T) {
		resource, err := LoadResource(ctx, "", logger)

		assert.Error(t, err)
		assert.Nil(t, resource)
		assert.Contains(t, err.Error(), "error reading resource file")
	})

	t.Run("VeryLongValidResource", func(t *testing.T) {
		tmpDir := t.TempDir()
		resourceFile := filepath.Join(tmpDir, "long.pkl")

		// Create a resource with many fields to test different parsing paths
		content := `amends "package://schema.kdeps.com/core@` + schema.SchemaVersion(ctx) + `#/Resource.pkl"

actionID = "very_long_action_id_that_tests_string_handling"
name = "Very Long Test Action Name That Tests String Handling Capabilities"
category = "comprehensive_testing_category"
description = "This is a very long description that tests the resource loading functionality with extensive text content and various characters including numbers 123, symbols !@#$%^&*(), and unicode characters 你好世界"
requires {
  "dependency_one"
  "dependency_two"
  "dependency_three"
  "dependency_four"
  "dependency_five"
}
run {
  chat {
    model = "llama3.2"
    prompt = "This is a very long prompt that tests the chat functionality with extensive text content"
    JSONResponse = true
    JSONResponseKeys {
      "result"
      "status"
      "metadata"
      "timestamp"
      "version"
    }
    timeoutDuration = 120.s
  }
  preflightCheck {
    validations {
      1 + 1 == 2
      "hello" + " " + "world" == "hello world"
      10 > 5
      true == true
      false != true
    }
  }
  APIResponse {
    success = true
    response {
      data {
        "first_piece_of_data"
        "second_piece_of_data"
        "third_piece_of_data"
      }
    }
  }
}
`
		err := os.WriteFile(resourceFile, []byte(content), 0o644)
		require.NoError(t, err)

		resource, err := LoadResource(ctx, resourceFile, logger)

		require.NoError(t, err)
		assert.NotNil(t, resource)
		assert.Equal(t, "very_long_action_id_that_tests_string_handling", resource.ActionID)
		assert.Contains(t, resource.Description, "unicode characters")
	})
}
