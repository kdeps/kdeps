package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExecute struct {
	command     string
	args        []string
	env         []string
	shouldError bool
	stdout      string
	stderr      string
}

func (m *mockExecute) Execute(ctx context.Context) (struct {
	Stdout string
	Stderr string
}, error,
) {
	if m.shouldError {
		return struct {
			Stdout string
			Stderr string
		}{}, errors.New("mock execution error")
	}
	return struct {
		Stdout string
		Stderr string
	}{
		Stdout: m.stdout,
		Stderr: m.stderr,
	}, nil
}

func setupTestResolver(t *testing.T) *DependencyResolver {
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	// Create necessary directories
	err := fs.MkdirAll("/tmp", 0o755)
	require.NoError(t, err)
	err = fs.MkdirAll("/files", 0o755)
	require.NoError(t, err)

	return &DependencyResolver{
		Fs:                fs,
		Logger:            logger,
		Context:           ctx,
		FilesDir:          "/files",
		RequestID:         "test-request",
		AnacondaInstalled: false,
	}
}

func TestHandlePython(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("SuccessfulExecution", func(t *testing.T) {
		t.Parallel()
		pythonBlock := &python.ResourcePython{
			Script: "print('Hello, World!')",
		}

		err := dr.HandlePython("test-action", pythonBlock)
		assert.NoError(t, err)
	})

	t.Run("DecodeError", func(t *testing.T) {
		t.Parallel()
		pythonBlock := &python.ResourcePython{
			Script: "invalid base64",
		}

		err := dr.HandlePython("test-action", pythonBlock)
		assert.NoError(t, err)
	})
}

func TestDecodePythonBlock(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("ValidBase64Script", func(t *testing.T) {
		t.Parallel()
		encodedScript := "cHJpbnQoJ0hlbGxvLCBXb3JsZCEnKQ==" // "print('Hello, World!')"
		pythonBlock := &python.ResourcePython{
			Script: encodedScript,
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
		assert.Equal(t, "print('Hello, World!')", pythonBlock.Script)
	})

	t.Run("ValidBase64Env", func(t *testing.T) {
		t.Parallel()
		env := map[string]string{
			"TEST_KEY": "dGVzdF92YWx1ZQ==", // "test_value"
		}
		pythonBlock := &python.ResourcePython{
			Script: "print('test')",
			Env:    &env,
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", (*pythonBlock.Env)["TEST_KEY"])
	})

	t.Run("InvalidBase64Script", func(t *testing.T) {
		t.Parallel()
		pythonBlock := &python.ResourcePython{
			Script: "invalid base64",
		}

		err := dr.decodePythonBlock(pythonBlock)
		assert.NoError(t, err)
	})
}

func TestWritePythonStdoutToFile(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("ValidStdout", func(t *testing.T) {
		t.Parallel()
		encodedStdout := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!"
		resourceID := "test-resource-valid"

		filePath, err := dr.WritePythonStdoutToFile(resourceID, &encodedStdout)
		assert.NoError(t, err)
		assert.NotEmpty(t, filePath)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, filePath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "Hello, World!")
	})

	t.Run("NilStdout", func(t *testing.T) {
		t.Parallel()
		filePath, err := dr.WritePythonStdoutToFile("test-resource-nil", nil)
		assert.NoError(t, err)
		assert.Empty(t, filePath)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		t.Parallel()
		invalidStdout := "invalid base64"
		_, err := dr.WritePythonStdoutToFile("test-resource-invalid", &invalidStdout)
		assert.NoError(t, err)
	})
}

func TestFormatPythonEnv(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("ValidEnv", func(t *testing.T) {
		t.Parallel()
		env := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		formatted := dr.formatPythonEnv(&env)
		assert.Len(t, formatted, 2)
		assert.Contains(t, formatted, "KEY1=value1")
		assert.Contains(t, formatted, "KEY2=value2")
	})

	t.Run("NilEnv", func(t *testing.T) {
		t.Parallel()
		formatted := dr.formatPythonEnv(nil)
		assert.Empty(t, formatted)
	})

	t.Run("EmptyEnv", func(t *testing.T) {
		t.Parallel()
		env := map[string]string{}
		formatted := dr.formatPythonEnv(&env)
		assert.Empty(t, formatted)
	})
}

func TestCreatePythonTempFile(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("ValidScript", func(t *testing.T) {
		t.Parallel()
		script := "print('test')"

		file, err := dr.createPythonTempFile(script)
		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, file.Name())
		assert.NoError(t, err)
		assert.Equal(t, script, string(content))

		// Cleanup
		dr.cleanupTempFile(file.Name())
	})

	t.Run("EmptyScript", func(t *testing.T) {
		t.Parallel()
		file, err := dr.createPythonTempFile("")
		assert.NoError(t, err)
		assert.NotNil(t, file)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, file.Name())
		assert.NoError(t, err)
		assert.Empty(t, string(content))

		// Cleanup
		dr.cleanupTempFile(file.Name())
	})
}

func TestCleanupTempFile(t *testing.T) {
	t.Parallel()
	dr := setupTestResolver(t)

	t.Run("ExistingFile", func(t *testing.T) {
		t.Parallel()
		// Create a temporary file
		file, err := dr.Fs.Create("/tmp/test-file.txt")
		require.NoError(t, err)
		file.Close()

		// Cleanup the file
		dr.cleanupTempFile("/tmp/test-file.txt")

		// Verify file is deleted
		exists, err := afero.Exists(dr.Fs, "/tmp/test-file.txt")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		t.Parallel()
		// Attempt to cleanup non-existent file
		dr.cleanupTempFile("/tmp/non-existent.txt")
		// Should not panic or error
	})
}
