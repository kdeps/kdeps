package cmd_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stdoutMutex sync.Mutex

func saveAndRestoreStdout(t *testing.T, newStdout *os.File) func() {
	stdoutMutex.Lock()
	original := os.Stdout
	os.Stdout = newStdout
	return func() {
		os.Stdout = original
		stdoutMutex.Unlock()
	}
}

// captureOutput redirects stdout to a buffer and returns a restore func along
// with the buffer pointer.
func captureOutput() (*bytes.Buffer, func()) {
	stdoutMutex.Lock()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buf := &bytes.Buffer{}
	done := make(chan struct{})

	go func() {
		_, _ = io.Copy(buf, r)
		close(done)
	}()

	restore := func() {
		w.Close()
		<-done
		os.Stdout = old
		stdoutMutex.Unlock()
	}
	return buf, restore
}

func TestNewScaffoldCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)
	assert.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
	assert.Equal(t, "Scaffold specific files for an agent", cmd.Short)
	assert.Contains(t, cmd.Long, "Available resources:")
}

func TestNewScaffoldCommandNoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	cmd := NewScaffoldCommand(ctx, fs, logger)
	cmd.SetArgs([]string{testAgentDir})
	err = cmd.Execute()
	require.NoError(t, err)
}

func TestNewScaffoldCommandValidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	validResources := []string{"client", "exec", "llm", "python", "response"}

	for _, resource := range validResources {
		cmd := NewScaffoldCommand(ctx, fs, logger)
		cmd.SetArgs([]string{testAgentDir, resource})
		err := cmd.Execute()
		require.NoError(t, err)

		// Verify file was created
		filePath := filepath.Join(testAgentDir, "resources", resource+".pkl")
		exists, err := afero.Exists(fs, filePath)
		require.NoError(t, err)
		assert.True(t, exists, "File %s should exist", filePath)
	}
}

func TestNewScaffoldCommandInvalidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	cmd := NewScaffoldCommand(ctx, fs, logger)
	cmd.SetArgs([]string{testAgentDir, "invalid-resource"})
	err = cmd.Execute()
	require.NoError(t, err) // Command doesn't return error for invalid resources

	// Verify file was not created
	filePath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err := afero.Exists(fs, filePath)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandMultipleResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	require.NoError(t, err)

	cmd := NewScaffoldCommand(ctx, fs, logger)
	cmd.SetArgs([]string{testAgentDir, "client", "exec", "invalid-resource"})
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify valid files were created
	clientPath := filepath.Join(testAgentDir, "resources", "client.pkl")
	exists, err := afero.Exists(fs, clientPath)
	require.NoError(t, err)
	assert.True(t, exists, "File %s should exist", clientPath)

	execPath := filepath.Join(testAgentDir, "resources", "exec.pkl")
	exists, err = afero.Exists(fs, execPath)
	require.NoError(t, err)
	assert.True(t, exists, "File %s should exist", execPath)

	// Verify invalid file was not created
	invalidPath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err = afero.Exists(fs, invalidPath)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandNoArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)
	err := cmd.Execute()
	require.Error(t, err) // Should fail due to missing required argument
}

func TestNewScaffoldCommand_ListResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)

	// Just ensure it completes without panic when no resource names are supplied.
	cmd.Run(cmd, []string{"myagent"})
}

func TestNewScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)
	cmd.Run(cmd, []string{"agent", "unknown"}) // should handle gracefully without panic
}

func TestNewScaffoldCommand_GenerateFile(t *testing.T) {
	t.Setenv("NON_INTERACTIVE", "1") // speed

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)

	cmd.Run(cmd, []string{"agentx", "client"})

	// Verify generated file exists
	if ok, _ := afero.Exists(fs, "agentx/resources/client.pkl"); !ok {
		t.Fatalf("expected generated client.pkl file not found")
	}
}

// TestScaffoldCommand_Happy creates two valid resources and asserts files are
// written under the expected paths.
func TestScaffoldCommand_Happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)

	agent := "myagent"
	args := []string{agent, "client", "exec"}

	// Capture output just in case (not strictly needed but keeps test quiet).
	_, restore := captureOutput()
	defer restore()

	cmd.Run(cmd, args)

	// Verify generated files exist.
	expected := []string{
		agent + "/resources/client.pkl",
		agent + "/resources/exec.pkl",
	}
	for _, path := range expected {
		if ok, _ := afero.Exists(fs, path); !ok {
			t.Fatalf("expected file %s to exist", path)
		}
	}

	_ = schema.Version(ctx)
}

// TestScaffoldCommand_InvalidResource ensures invalid names are reported and
// not created.
func TestScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(ctx, fs, logger)
	agent := "badagent"

	buf, restore := captureOutput()
	defer restore()

	cmd.Run(cmd, []string{agent, "bogus"})

	// The bogus file should not be created.
	if ok, _ := afero.Exists(fs, agent+"/resources/bogus.pkl"); ok {
		t.Fatalf("unexpected file created for invalid resource")
	}

	_ = buf // output not asserted; just ensuring no panic

	_ = schema.Version(ctx)
}
