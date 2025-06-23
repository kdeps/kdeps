package cmd_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewAddCommandFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	assert.Equal(t, "install [package]", cmd.Use)
	assert.Equal(t, []string{"i"}, cmd.Aliases)
	assert.Equal(t, "Install an AI agent locally", cmd.Short)
	assert.Equal(t, "$ kdeps install ./myAgent.kdeps", cmd.Example)
}

func TestNewAddCommandExecution(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	err := fs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create test package file
	agentKdepsPath := filepath.Join(testDir, "agent.kdeps")
	err = afero.WriteFile(fs, agentKdepsPath, []byte("test package"), 0o644)
	assert.NoError(t, err)

	// Test error case - no arguments
	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package file
	cmd = NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{filepath.Join(testDir, "nonexistent.kdeps")})
	err = cmd.Execute()
	assert.Error(t, err)

	// Test error case - invalid package content
	cmd = NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{agentKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err)
}

func TestNewAddCommandValidPackage(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Create test directory
	testDir := filepath.Join("/test")
	validAgentDir := filepath.Join(testDir, "valid-agent")
	err := fs.MkdirAll(validAgentDir, 0o755)
	assert.NoError(t, err)

	// Create test package file with valid structure
	workflowPath := filepath.Join(validAgentDir, "workflow.pkl")
	err = afero.WriteFile(fs, workflowPath, []byte("name: test\nversion: 1.0.0"), 0o644)
	assert.NoError(t, err)

	// Create resources directory and add required resources
	resourcesDir := filepath.Join(validAgentDir, "resources")
	err = fs.MkdirAll(resourcesDir, 0o755)
	assert.NoError(t, err)

	// Create all required resource files
	requiredResources := []string{"client.pkl", "exec.pkl", "llm.pkl", "python.pkl", "response.pkl"}
	for _, resource := range requiredResources {
		resourcePath := filepath.Join(resourcesDir, resource)
		err = afero.WriteFile(fs, resourcePath, []byte("resource content"), 0o644)
		assert.NoError(t, err)
	}

	validKdepsPath := filepath.Join(testDir, "valid-agent.kdeps")
	err = afero.WriteFile(fs, validKdepsPath, []byte("valid package"), 0o644)
	assert.NoError(t, err)

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{validKdepsPath})
	err = cmd.Execute()
	assert.Error(t, err) // Should fail due to invalid package format, but in a different way
}

// TestNewAddCommand_RunE ensures the command is wired correctly – we expect an
// error because the provided package path doesn't exist, but the purpose of
// the test is simply to execute the RunE handler to mark lines as covered.
func TestNewAddCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, "/kdeps", logger)

	// Supply non-existent path so that ExtractPackage fails and RunE returns
	// an error. Success isn't required – only execution.
	if err := cmd.RunE(cmd, []string{"/does/not/exist.kdeps"}); err == nil {
		t.Fatalf("expected error from RunE due to missing package file")
	}
}

// TestNewAddCommand_ErrorPath ensures RunE returns an error when ExtractPackage fails.
func TestNewAddCommand_ErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cmd := NewAddCommand(fs, ctx, "/tmp/kdeps", logging.NewTestLogger())
	cmd.SetArgs([]string{"nonexistent.kdeps"})

	err := cmd.Execute()
	assert.Error(t, err, "expected error when package file does not exist")
}

func TestNewAddCommand_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	cmd := NewAddCommand(fs, ctx, "/tmp/kdeps", logging.NewTestLogger())

	assert.Equal(t, "install [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "Install")

	// missing arg should error
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

// TestNewAddCommandRunE executes the command with a dummy package path. We
// only assert that an error is returned (because the underlying extractor will
// fail with the in-memory filesystem). The objective is to exercise the command
// wiring rather than validate its behaviour.
func TestNewAddCommandRunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewAddCommand(fs, context.Background(), "/kdeps", logging.NewTestLogger())

	if err := cmd.RunE(cmd, []string{"dummy.kdeps"}); err == nil {
		t.Fatalf("expected error due to missing package file, got nil")
	}
}

func TestNewAddCommand_Structure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/test/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)

	if cmd.Use != "install [package]" {
		t.Errorf("expected Use 'install [package]', got %q", cmd.Use)
	}
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "i" {
		t.Errorf("expected Aliases ['i'], got %v", cmd.Aliases)
	}
	if cmd.Short != "Install an AI agent locally" {
		t.Errorf("expected Short 'Install an AI agent locally', got %q", cmd.Short)
	}
	if cmd.Example != "$ kdeps install ./myAgent.kdeps" {
		t.Errorf("expected Example, got %q", cmd.Example)
	}
}

func TestNewAddCommand_ArgsValidation(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/test/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)

	// No arguments should error
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error when no arguments provided")
	}

	// One argument should pass
	err = cmd.Args(cmd, []string{"test.kdeps"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewAddCommand_RunE_ErrorPath(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/test/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	cmd.SetArgs([]string{"nonexistent.kdeps"})
	err := cmd.Execute()
	if err == nil {
		t.Error("expected error for nonexistent package file")
	}
}

// TestNewAddCommand_Success tests the success path with mocked ExtractPackage
func TestNewAddCommand_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Store original function
	originalExtractPackageFn := cmd.ExtractPackageFn

	// Restore original function after test
	defer func() {
		cmd.ExtractPackageFn = originalExtractPackageFn
	}()

	// Mock ExtractPackage for success path
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{Workflow: "test-workflow"}, nil
	}

	// Capture the printed message
	originalPrintlnFn := cmd.PrintlnFn
	defer func() {
		cmd.PrintlnFn = originalPrintlnFn
	}()

	var printedMessage string
	cmd.PrintlnFn = func(a ...interface{}) (n int, err error) {
		printedMessage = fmt.Sprintln(a...)
		return len(printedMessage), nil
	}

	// Test the success path
	addCmd := cmd.NewAddCommand(fs, ctx, kdepsDir, logger)
	err := addCmd.RunE(addCmd, []string{"test.kdeps"})

	assert.NoError(t, err)
	assert.Contains(t, printedMessage, "AI agent installed locally: test.kdeps")
}

// TestNewAddCommand_ExtractPackageError tests error handling when ExtractPackage fails
func TestNewAddCommand_ExtractPackageError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Store original function
	originalExtractPackageFn := cmd.ExtractPackageFn

	// Restore original function after test
	defer func() {
		cmd.ExtractPackageFn = originalExtractPackageFn
	}()

	// Mock ExtractPackage for error path
	cmd.ExtractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, fmt.Errorf("extract package error")
	}

	// Test the error path
	addCmd := cmd.NewAddCommand(fs, ctx, kdepsDir, logger)
	err := addCmd.RunE(addCmd, []string{"test.kdeps"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "extract package error")
}
