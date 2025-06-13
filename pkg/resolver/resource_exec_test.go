package resolver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestExecResolver(t *testing.T) *DependencyResolver {
	tmpDir := t.TempDir()

	fs := afero.NewOsFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	filesDir := filepath.Join(tmpDir, "files")
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(filepath.Join(actionDir, "exec"), 0o755)
	_ = fs.MkdirAll(filesDir, 0o755)

	return &DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}
}

func TestHandleExec(t *testing.T) {

	dr := setupTestExecResolver(t)

	t.Run("SuccessfulExecution", func(t *testing.T) {
	
		execBlock := &exec.ResourceExec{
			Command: "echo 'Hello, World!'",
		}

		err := dr.HandleExec("test-action", execBlock)
		assert.NoError(t, err)
	})

	t.Run("DecodeError", func(t *testing.T) {
	
		execBlock := &exec.ResourceExec{
			Command: "invalid base64",
		}

		err := dr.HandleExec("test-action", execBlock)
		assert.NoError(t, err)
	})
}

func TestDecodeExecBlock(t *testing.T) {

	dr := setupTestExecResolver(t)

	t.Run("ValidBase64Command", func(t *testing.T) {
	
		encodedCommand := "ZWNobyAnSGVsbG8sIFdvcmxkISc=" // "echo 'Hello, World!'"
		execBlock := &exec.ResourceExec{
			Command: encodedCommand,
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
		assert.Equal(t, "echo 'Hello, World!'", execBlock.Command)
	})

	t.Run("ValidBase64Env", func(t *testing.T) {
	
		env := map[string]string{
			"TEST_KEY": "dGVzdF92YWx1ZQ==", // "test_value"
		}
		execBlock := &exec.ResourceExec{
			Command: "echo 'test'",
			Env:     &env,
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
		assert.Equal(t, "test_value", (*execBlock.Env)["TEST_KEY"])
	})

	t.Run("InvalidBase64Command", func(t *testing.T) {
	
		execBlock := &exec.ResourceExec{
			Command: "invalid base64",
		}

		err := dr.decodeExecBlock(execBlock)
		assert.NoError(t, err)
	})
}

func TestWriteStdoutToFile(t *testing.T) {

	dr := setupTestExecResolver(t)

	t.Run("ValidStdout", func(t *testing.T) {
	
		encodedStdout := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!"
		resourceID := "test-resource"

		filePath, err := dr.WriteStdoutToFile(resourceID, &encodedStdout)
		assert.NoError(t, err)
		assert.NotEmpty(t, filePath)

		// Verify file contents
		content, err := afero.ReadFile(dr.Fs, filePath)
		assert.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("NilStdout", func(t *testing.T) {
	
		filePath, err := dr.WriteStdoutToFile("test-resource", nil)
		assert.NoError(t, err)
		assert.Empty(t, filePath)
	})

	t.Run("InvalidBase64", func(t *testing.T) {
	
		invalidStdout := "invalid base64"
		_, err := dr.WriteStdoutToFile("test-resource", &invalidStdout)
		assert.NoError(t, err)
	})
}

// skipIfPKLError skips the test when the provided error is non-nil and indicates
// that the PKL binary / registry is not available in the current CI
// environment. That allows us to exercise all pre-PKL logic while remaining
// green when the external dependency is missing.
func skipIfPKLError(t *testing.T, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "Cannot find module") ||
		strings.Contains(msg, "Received unexpected status code") ||
		strings.Contains(msg, "apple PKL not found") ||
		strings.Contains(msg, "Invalid token") {
		t.Skipf("Skipping test because PKL is unavailable: %v", err)
	}
}

func TestAppendExecEntry(t *testing.T) {


	// Helper to create fresh resolver inside each sub-test
	newResolver := func(t *testing.T) (*DependencyResolver, string) {
		dr := setupTestExecResolver(t)
		pklPath := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
		return dr, pklPath
	}

	t.Run("NewEntry", func(t *testing.T) {
	
		dr, pklPath := newResolver(t)

		initialContent := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Exec.pkl"

resources {
}`, schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initialContent), 0o644))

		newExec := &exec.ResourceExec{
			Command:   "echo 'test'",
			Stdout:    utils.StringPtr("test output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendExecEntry("test-resource", newExec)
		skipIfPKLError(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLError(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test-resource")
		assert.Contains(t, string(content), "ZWNobyAndGVzdCc=")
	})

	t.Run("ExistingEntry", func(t *testing.T) {
	
		dr, pklPath := newResolver(t)

		initialContent := fmt.Sprintf(`extends "package://schema.kdeps.com/core@%s#/Exec.pkl"

resources {
  ["existing-resource"] {
    command = "echo 'old'"
    timestamp = 1234567890.ns
  }
}`, schema.SchemaVersion(dr.Context))
		require.NoError(t, afero.WriteFile(dr.Fs, pklPath, []byte(initialContent), 0o644))

		newExec := &exec.ResourceExec{
			Command:   "echo 'new'",
			Stdout:    utils.StringPtr("new output"),
			Timestamp: &pkl.Duration{Value: float64(time.Now().Unix()), Unit: pkl.Nanosecond},
		}

		err := dr.AppendExecEntry("existing-resource", newExec)
		skipIfPKLError(t, err)
		assert.NoError(t, err)

		content, err := afero.ReadFile(dr.Fs, pklPath)
		skipIfPKLError(t, err)
		require.NoError(t, err)
		assert.Contains(t, string(content), "existing-resource")
		assert.Contains(t, string(content), "ZWNobyAnbmV3Jw==")
		assert.NotContains(t, string(content), "echo 'old'")
	})
}

func TestEncodeExecEnv(t *testing.T) {

	dr := setupTestExecResolver(t)

	t.Run("ValidEnv", func(t *testing.T) {
	
		env := map[string]string{
			"KEY1": "value1",
			"KEY2": "value2",
		}

		encoded := dr.encodeExecEnv(&env)
		assert.NotNil(t, encoded)
		assert.Equal(t, "dmFsdWUx", (*encoded)["KEY1"])
		assert.Equal(t, "dmFsdWUy", (*encoded)["KEY2"])
	})

	t.Run("NilEnv", func(t *testing.T) {
	
		encoded := dr.encodeExecEnv(nil)
		assert.Nil(t, encoded)
	})

	t.Run("EmptyEnv", func(t *testing.T) {
	
		env := map[string]string{}
		encoded := dr.encodeExecEnv(&env)
		assert.NotNil(t, encoded)
		assert.Empty(t, *encoded)
	})
}

func TestEncodeExecOutputs(t *testing.T) {

	dr := setupTestExecResolver(t)

	t.Run("ValidOutputs", func(t *testing.T) {
	
		stdout := "test output"
		stderr := "test error"

		encodedStdout, encodedStderr := dr.encodeExecOutputs(&stderr, &stdout)
		assert.NotNil(t, encodedStdout)
		assert.NotNil(t, encodedStderr)
		assert.Equal(t, "dGVzdCBlcnJvcg==", *encodedStdout)
		assert.Equal(t, "dGVzdCBvdXRwdXQ=", *encodedStderr)
	})

	t.Run("NilOutputs", func(t *testing.T) {
	
		encodedStdout, encodedStderr := dr.encodeExecOutputs(nil, nil)
		assert.Nil(t, encodedStdout)
		assert.Nil(t, encodedStderr)
	})
}
