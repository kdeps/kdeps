package cmd_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewRootCommand_Structure(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	assert.Equal(t, "kdeps", rootCmd.Use)
	assert.Equal(t, "Multi-model AI agent framework.", rootCmd.Short)
	assert.Contains(t, rootCmd.Long, "Kdeps is a multi-model AI agent framework")
	assert.NotEmpty(t, rootCmd.Version)
}

func TestNewRootCommand_Subcommands(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Check that all expected subcommands are present
	expectedCommands := []string{"new", "scaffold", "install", "package", "build", "run"}
	actualCommands := make([]string, 0, len(rootCmd.Commands()))
	for _, subcmd := range rootCmd.Commands() {
		actualCommands = append(actualCommands, subcmd.Name())
	}

	for _, expected := range expectedCommands {
		assert.Contains(t, actualCommands, expected, "Expected subcommand %s not found", expected)
	}
}

func TestNewRootCommand_PersistentFlags(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Check that the --latest flag is present
	latestFlag := rootCmd.PersistentFlags().Lookup("latest")
	assert.NotNil(t, latestFlag, "Expected --latest flag not found")
	assert.Equal(t, "l", latestFlag.Shorthand)
	assert.Equal(t, "false", latestFlag.DefValue)
}

func TestNewRootCommand_SubcommandInjection(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	// Mock subcommand functions to return test commands
	origNewAgent := cmd.NewAgentCommandFn
	origNewScaffold := cmd.NewScaffoldCommandFn
	origNewAdd := cmd.NewAddCommandFn
	origNewPackage := cmd.NewPackageCommandFn
	origNewBuild := cmd.NewBuildCommandFn
	origNewRun := cmd.NewRunCommandFn

	defer func() {
		cmd.NewAgentCommandFn = origNewAgent
		cmd.NewScaffoldCommandFn = origNewScaffold
		cmd.NewAddCommandFn = origNewAdd
		cmd.NewPackageCommandFn = origNewPackage
		cmd.NewBuildCommandFn = origNewBuild
		cmd.NewRunCommandFn = origNewRun
	}()

	// Create mock commands
	mockAgentCmd := &cobra.Command{Use: "mock-agent"}
	mockScaffoldCmd := &cobra.Command{Use: "mock-scaffold"}
	mockAddCmd := &cobra.Command{Use: "mock-add"}
	mockPackageCmd := &cobra.Command{Use: "mock-package"}
	mockBuildCmd := &cobra.Command{Use: "mock-build"}
	mockRunCmd := &cobra.Command{Use: "mock-run"}

	// Set mock functions
	cmd.NewAgentCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
		return mockAgentCmd
	}
	cmd.NewScaffoldCommandFn = func(fs afero.Fs, ctx context.Context, logger *logging.Logger) *cobra.Command {
		return mockScaffoldCmd
	}
	cmd.NewAddCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
		return mockAddCmd
	}
	cmd.NewPackageCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return mockPackageCmd
	}
	cmd.NewBuildCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
		return mockBuildCmd
	}
	cmd.NewRunCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
		return mockRunCmd
	}

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Verify that the mock commands are added
	subcommandNames := make([]string, 0, len(rootCmd.Commands()))
	for _, subcmd := range rootCmd.Commands() {
		subcommandNames = append(subcommandNames, subcmd.Use)
	}

	assert.Contains(t, subcommandNames, "mock-agent")
	assert.Contains(t, subcommandNames, "mock-scaffold")
	assert.Contains(t, subcommandNames, "mock-add")
	assert.Contains(t, subcommandNames, "mock-package")
	assert.Contains(t, subcommandNames, "mock-build")
	assert.Contains(t, subcommandNames, "mock-run")
}

func TestNewRootCommand_CommandSortingDisabled(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	// Reset command sorting to default
	cobra.EnableCommandSorting = true

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Verify that command sorting is disabled
	assert.False(t, cobra.EnableCommandSorting)

	// Verify that the root command was created successfully
	assert.NotNil(t, rootCmd)
}

func TestNewRootCommand_HelpText(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Test help text generation - check that template is not empty
	helpTemplate := rootCmd.HelpTemplate()
	assert.NotEmpty(t, helpTemplate)

	// Test actual help output by capturing it
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	assert.NoError(t, err)

	helpOutput := buf.String()
	assert.Contains(t, helpOutput, "Usage:")
	assert.Contains(t, helpOutput, "kdeps")

	// Test that the command has a valid help template
	assert.NotNil(t, rootCmd)
}

func TestNewRootCommand_Execute(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	env := &environment.Environment{}
	logger := logging.NewTestSafeLogger()

	rootCmd := cmd.NewRootCommand(fs, ctx, kdepsDir, systemCfg, env, logger)

	// Test that root command can be executed without arguments (should show help)
	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	// Should not error when no subcommand is provided
	assert.NoError(t, err)
}

func TestNewAgentCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAgentCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "new [agentName]", cmd.Use)
}

func TestNewScaffoldCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
}

func TestNewAddCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "install [package]", cmd.Use)
}

func TestNewPackageCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	cmd := NewPackageCommand(fs, ctx, kdepsDir, env, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "package [agent-dir]", cmd.Use)
}

func TestNewBuildCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "build [package]", cmd.Use)
}

func TestNewRunCommand(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "run [package]", cmd.Use)
}

func TestNewRootCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{}
	cmd := NewRootCommand(fs, context.Background(), "/kdeps", nil, env, logging.NewTestLogger())
	if cmd.Use != "kdeps" {
		t.Fatalf("expected root command name kdeps, got %s", cmd.Use)
	}
	if cmd.Version == "" {
		t.Fatalf("version string should be set")
	}
	if len(cmd.Commands()) == 0 {
		t.Fatalf("expected subcommands attached")
	}
}
