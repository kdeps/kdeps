package cmd_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewScaffoldCommand_Structure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	assert.Equal(t, "scaffold [agentName] [fileNames...]", command.Use)
	assert.Equal(t, "Scaffold specific files for an agent", command.Short)
	assert.Contains(t, command.Long, "Scaffold specific files for an agent")
}

func TestNewScaffoldCommand_NoFileNames(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		capturedOutput = append(capturedOutput, a[0].(string))
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with only agent name (no file names)
	command.SetArgs([]string{"test-agent"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify that available resources are shown
	assert.Contains(t, capturedOutput, "Available resources:")
	assert.Contains(t, capturedOutput, "  - client: HTTP client for making API calls")
	assert.Contains(t, capturedOutput, "  - exec: Execute shell commands and scripts")
	assert.Contains(t, capturedOutput, "  - llm: Large Language Model interaction")
	assert.Contains(t, capturedOutput, "  - python: Run Python scripts")
	assert.Contains(t, capturedOutput, "  - response: API response handling")
}

func TestNewScaffoldCommand_ValidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Mock GenerateSpecificAgentFile to succeed
	origGenerateFile := cmd.GenerateSpecificAgentFileFn
	cmd.GenerateSpecificAgentFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName, resourceName string) error {
		return nil
	}
	defer func() { cmd.GenerateSpecificAgentFileFn = origGenerateFile }()

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		if len(a) >= 2 {
			capturedOutput = append(capturedOutput, a[0].(string)+" "+a[1].(string))
		}
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with valid resources
	command.SetArgs([]string{"test-agent", "client", "exec"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify success messages
	assert.Contains(t, capturedOutput, "Successfully scaffolded file: test-agent/resources/client.pkl")
	assert.Contains(t, capturedOutput, "Successfully scaffolded file: test-agent/resources/exec.pkl")
}

func TestNewScaffoldCommand_InvalidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		// Join all arguments to capture the full line
		var parts []string
		for _, arg := range a {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		capturedOutput = append(capturedOutput, strings.Join(parts, " "))
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with invalid resources
	command.SetArgs([]string{"test-agent", "invalid", "also-invalid"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify error messages
	assert.Contains(t, capturedOutput, "\nInvalid resource(s): invalid, also-invalid")
	assert.Contains(t, capturedOutput, "\nAvailable resources:")
}

func TestNewScaffoldCommand_MixedValidAndInvalid(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Mock GenerateSpecificAgentFile to succeed for valid resources
	origGenerateFile := cmd.GenerateSpecificAgentFileFn
	cmd.GenerateSpecificAgentFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName, resourceName string) error {
		return nil
	}
	defer func() { cmd.GenerateSpecificAgentFileFn = origGenerateFile }()

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		// Join all arguments to capture the full line
		var parts []string
		for _, arg := range a {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		capturedOutput = append(capturedOutput, strings.Join(parts, " "))
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with mixed valid and invalid resources
	command.SetArgs([]string{"test-agent", "client", "invalid", "exec"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify success messages for valid resources
	assert.Contains(t, capturedOutput, "Successfully scaffolded file: test-agent/resources/client.pkl")
	assert.Contains(t, capturedOutput, "Successfully scaffolded file: test-agent/resources/exec.pkl")

	// Verify error messages for invalid resources
	assert.Contains(t, capturedOutput, "\nInvalid resource(s): invalid")
}

func TestNewScaffoldCommand_GenerateFileError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Mock GenerateSpecificAgentFile to return error
	origGenerateFile := cmd.GenerateSpecificAgentFileFn
	cmd.GenerateSpecificAgentFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName, resourceName string) error {
		return errors.New("generation failed")
	}
	defer func() { cmd.GenerateSpecificAgentFileFn = origGenerateFile }()

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		// Join all arguments to capture the full line
		var parts []string
		for _, arg := range a {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		capturedOutput = append(capturedOutput, strings.Join(parts, " "))
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with valid resource that fails generation
	command.SetArgs([]string{"test-agent", "client"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify error message
	assert.Contains(t, capturedOutput, "Error: generation failed")
}

func TestNewScaffoldCommand_WithPklExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Mock GenerateSpecificAgentFile to succeed
	origGenerateFile := cmd.GenerateSpecificAgentFileFn
	cmd.GenerateSpecificAgentFileFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger, agentName, resourceName string) error {
		return nil
	}
	defer func() { cmd.GenerateSpecificAgentFileFn = origGenerateFile }()

	// Capture output
	var capturedOutput []string
	origPrintln := cmd.PrintlnFn
	cmd.PrintlnFn = func(a ...interface{}) (int, error) {
		// Join all arguments to capture the full line
		var parts []string
		for _, arg := range a {
			parts = append(parts, fmt.Sprintf("%v", arg))
		}
		capturedOutput = append(capturedOutput, strings.Join(parts, " "))
		return 0, nil
	}
	defer func() { cmd.PrintlnFn = origPrintln }()

	// Test with .pkl extension
	command.SetArgs([]string{"test-agent", "client.pkl"})
	err := command.Execute()
	assert.NoError(t, err)

	// Verify success message (extension should be stripped)
	assert.Contains(t, capturedOutput, "Successfully scaffolded file: test-agent/resources/client.pkl")
}

func TestNewScaffoldCommand_ArgumentValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestSafeLogger()

	command := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Test with no arguments (should fail validation)
	command.SetArgs([]string{})
	err := command.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg")
}

func TestNewScaffoldCommandNoFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	assert.NoError(t, err)

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir})
	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestNewScaffoldCommandValidResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	assert.NoError(t, err)

	validResources := []string{"client", "exec", "llm", "python", "response"}

	for _, resource := range validResources {
		cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
		cmd.SetArgs([]string{testAgentDir, resource})
		err := cmd.Execute()
		assert.NoError(t, err)

		// Verify file was created
		filePath := filepath.Join(testAgentDir, "resources", resource+".pkl")
		exists, err := afero.Exists(fs, filePath)
		assert.NoError(t, err)
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
	assert.NoError(t, err)

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir, "invalid-resource"})
	err = cmd.Execute()
	assert.NoError(t, err) // Command doesn't return error for invalid resources

	// Verify file was not created
	filePath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err := afero.Exists(fs, filePath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandMultipleResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create test directory
	testAgentDir := filepath.Join("test-agent")
	err := fs.MkdirAll(testAgentDir, 0o755)
	assert.NoError(t, err)

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	cmd.SetArgs([]string{testAgentDir, "client", "exec", "invalid-resource"})
	err = cmd.Execute()
	assert.NoError(t, err)

	// Verify valid files were created
	clientPath := filepath.Join(testAgentDir, "resources", "client.pkl")
	exists, err := afero.Exists(fs, clientPath)
	assert.NoError(t, err)
	assert.True(t, exists, "File %s should exist", clientPath)

	execPath := filepath.Join(testAgentDir, "resources", "exec.pkl")
	exists, err = afero.Exists(fs, execPath)
	assert.NoError(t, err)
	assert.True(t, exists, "File %s should exist", execPath)

	// Verify invalid file was not created
	invalidPath := filepath.Join(testAgentDir, "resources", "invalid-resource.pkl")
	exists, err = afero.Exists(fs, invalidPath)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestNewScaffoldCommandNoArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	err := cmd.Execute()
	assert.Error(t, err) // Should fail due to missing required argument
}

func TestNewScaffoldCommand_ListResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)

	// Just ensure it completes without panic when no resource names are supplied.
	cmd.Run(cmd, []string{"myagent"})
}

func TestNewScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	cmd.Run(cmd, []string{"agent", "unknown"}) // should handle gracefully without panic
}

func TestNewScaffoldCommand_GenerateFile(t *testing.T) {
	_ = os.Setenv("NON_INTERACTIVE", "1") // speed

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)

	cmd.Run(cmd, []string{"agentx", "client"})

	// Verify generated file exists
	if ok, _ := afero.Exists(fs, "agentx/resources/client.pkl"); !ok {
		t.Fatalf("expected generated client.pkl file not found")
	}
}

// captureOutput redirects stdout to a buffer and returns a restore func along
// with the buffer pointer.
func captureOutput() (*bytes.Buffer, func()) {
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
	}
	return buf, restore
}

// TestScaffoldCommand_Happy creates two valid resources and asserts files are
// written under the expected paths.
func TestScaffoldCommand_Happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)

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

	_ = schema.SchemaVersion(ctx)
}

// TestScaffoldCommand_InvalidResource ensures invalid names are reported and
// not created.
func TestScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := cmd.NewScaffoldCommand(fs, ctx, logger)
	agent := "badagent"

	buf, restore := captureOutput()
	defer restore()

	cmd.Run(cmd, []string{agent, "bogus"})

	// The bogus file should not be created.
	if ok, _ := afero.Exists(fs, agent+"/resources/bogus.pkl"); ok {
		t.Fatalf("unexpected file created for invalid resource")
	}

	_ = buf // output not asserted; just ensuring no panic

	_ = schema.SchemaVersion(ctx)
}
