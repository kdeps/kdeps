package cmd_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	kdSchema "github.com/kdeps/schema/gen/kdeps"
	kdepsschema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// helper to execute a Cobra command and return the error.
func execCommand(c *cobra.Command, args ...string) error {
	c.SetArgs(args)
	return c.Execute()
}

func TestCommandConstructors_NoArgsError(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dir := t.TempDir()
	logger := logging.NewTestLogger()

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"add", NewAddCommand(fs, ctx, dir, logger)},
		{"build", NewBuildCommand(fs, ctx, dir, nil, logger)},
		{"run", NewRunCommand(fs, ctx, dir, nil, logger)},
	}

	for _, tt := range tests {
		if err := execCommand(tt.cmd); err == nil {
			t.Errorf("%s: expected error for missing args, got nil", tt.name)
		}
	}
}

func TestNewAgentCommand_Metadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dir := t.TempDir()
	logger := logging.NewTestLogger()

	c := NewAgentCommand(fs, ctx, dir, logger)
	if c.Use != "new [agentName]" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if len(c.Aliases) == 0 || c.Aliases[0] != "n" {
		t.Errorf("expected alias 'n', got %v", c.Aliases)
	}

	// Execute with missing arg to ensure validation triggers.
	if err := execCommand(c); err == nil {
		t.Fatal("expected error for missing agentName arg")
	}
}

func TestBuildAndRunCommands_RunEErrorFast(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	dir := t.TempDir()
	logger := logging.NewTestLogger()

	nonExist := "nonexistent.kdeps"

	buildCmd := NewBuildCommand(fs, ctx, dir, nil, logger)
	if err := execCommand(buildCmd, nonExist); err == nil {
		t.Errorf("BuildCommand expected error for missing file, got nil")
	}

	runCmd := NewRunCommand(fs, ctx, dir, nil, logger)
	if err := execCommand(runCmd, nonExist); err == nil {
		t.Errorf("RunCommand expected error for missing file, got nil")
	}
}

func TestNewBuildAndRunCommands_Basic(t *testing.T) {
	logger := logging.NewTestLogger()
	fs := afero.NewOsFs()
	ctx := context.Background()
	kdepsDir := t.TempDir()

	sysCfg := &kdeps.Kdeps{}

	buildCmd := NewBuildCommand(fs, ctx, kdepsDir, sysCfg, logger)
	require.Equal(t, "build [package]", buildCmd.Use)
	require.Len(t, buildCmd.Aliases, 1)

	// Invoke RunE directly with a non-existent file; we expect an error but no panic.
	err := buildCmd.RunE(buildCmd, []string{"missing.kdeps"})
	require.Error(t, err)

	runCmd := NewRunCommand(fs, ctx, kdepsDir, sysCfg, logger)
	require.Equal(t, "run [package]", runCmd.Use)
	require.Len(t, runCmd.Aliases, 1)

	err = runCmd.RunE(runCmd, []string{"missing.kdeps"})
	require.Error(t, err)
}

// TestNewBuildCommandRunE ensures calling the RunE function returns an error
// when provided a non-existent package, exercising the early ExtractPackage
// error path while covering the constructor's code.
func TestNewBuildCommandRunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewBuildCommand(fs, context.Background(), "/kdeps", &kdepsschema.Kdeps{}, logging.NewTestLogger())

	if err := cmd.RunE(cmd, []string{"missing.kdeps"}); err == nil {
		t.Fatalf("expected error due to missing package file, got nil")
	}
}

// TestNewPackageCommandRunE similarly exercises the early failure path.
func TestNewPackageCommandRunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewPackageCommand(fs, context.Background(), "/kdeps", nil, logging.NewTestLogger())

	if err := cmd.RunE(cmd, []string{"/nonexistent/agent"}); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// TestNewRunCommandRunE covers the run constructor.
func TestNewRunCommandRunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewRunCommand(fs, context.Background(), "/kdeps", &kdepsschema.Kdeps{}, logging.NewTestLogger())

	if err := cmd.RunE(cmd, []string{"missing.kdeps"}); err == nil {
		t.Fatalf("expected error due to missing package file, got nil")
	}
}

// TestNewScaffoldCommandRunE2 simply instantiates the command to cover the
// constructor's statements.
func TestNewScaffoldCommandRunE2(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewScaffoldCommand(fs, context.Background(), logging.NewTestLogger())

	if cmd == nil {
		t.Fatalf("expected command instance, got nil")
	}
}

func TestNewAddCommandExtra(t *testing.T) {
	cmd := NewAddCommand(afero.NewMemMapFs(), context.Background(), "kd", logging.NewTestLogger())
	require.Equal(t, "install [package]", cmd.Use)
	require.Equal(t, []string{"i"}, cmd.Aliases)
	require.Equal(t, "Install an AI agent locally", cmd.Short)
	require.Equal(t, "$ kdeps install ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewAgentCommandExtra(t *testing.T) {
	cmd := NewAgentCommand(afero.NewMemMapFs(), context.Background(), "kd", logging.NewTestLogger())
	require.Equal(t, "new [agentName]", cmd.Use)
	require.Equal(t, []string{"n"}, cmd.Aliases)
	require.Equal(t, "Create a new AI agent", cmd.Short)
	require.Error(t, cmd.Args(nil, []string{}))
	require.Error(t, cmd.Args(nil, []string{"a", "b"}))
	require.NoError(t, cmd.Args(nil, []string{"a"}))
}

func TestNewPackageCommandExtra(t *testing.T) {
	env := &environment.Environment{}
	cmd := NewPackageCommand(afero.NewMemMapFs(), context.Background(), "kd", env, logging.NewTestLogger())
	require.Equal(t, "package [agent-dir]", cmd.Use)
	require.Equal(t, []string{"p"}, cmd.Aliases)
	require.Equal(t, "Package an AI agent to .kdeps file", cmd.Short)
	require.Equal(t, "$ kdeps package ./myAgent/", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"dir"}))
}

func TestNewBuildCommandExtra(t *testing.T) {
	cfg := &kdeps.Kdeps{}
	cmd := NewBuildCommand(afero.NewMemMapFs(), context.Background(), "kd", cfg, logging.NewTestLogger())
	require.Equal(t, "build [package]", cmd.Use)
	require.Equal(t, []string{"b"}, cmd.Aliases)
	require.Equal(t, "Build a dockerized AI agent", cmd.Short)
	require.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewRunCommandExtra(t *testing.T) {
	cfg := &kdeps.Kdeps{}
	cmd := NewRunCommand(afero.NewMemMapFs(), context.Background(), "kd", cfg, logging.NewTestLogger())
	require.Equal(t, "run [package]", cmd.Use)
	require.Equal(t, []string{"r"}, cmd.Aliases)
	require.Equal(t, "Build and run a dockerized AI agent container", cmd.Short)
	require.Equal(t, "$ kdeps run ./myAgent.kdeps", cmd.Example)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"pkg"}))
}

func TestNewScaffoldCommandExtra(t *testing.T) {
	cmd := NewScaffoldCommand(afero.NewMemMapFs(), context.Background(), logging.NewTestLogger())
	require.Equal(t, "scaffold [agentName] [fileNames...]", cmd.Use)
	require.Empty(t, cmd.Aliases)
	require.Equal(t, "Scaffold specific files for an agent", cmd.Short)
	require.Error(t, cmd.Args(nil, []string{}))
	require.NoError(t, cmd.Args(nil, []string{"agent"}))
	require.NoError(t, cmd.Args(nil, []string{"agent", "file1"}))
}

func TestCommandConstructors_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kd"
	logger := logging.NewTestLogger()

	systemCfg := &kdSchema.Kdeps{}

	tests := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"add", func() *cobra.Command { return NewAddCommand(fs, ctx, kdepsDir, logger) }},
		{"build", func() *cobra.Command { return NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger) }},
		{"run", func() *cobra.Command { return NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger) }},
		{"package", func() *cobra.Command { return NewPackageCommand(fs, ctx, kdepsDir, nil, logger) }},
		{"scaffold", func() *cobra.Command { return NewScaffoldCommand(fs, ctx, logger) }},
		{"new", func() *cobra.Command { return NewAgentCommand(fs, ctx, kdepsDir, logger) }},
	}

	for _, tc := range tests {
		c := tc.cmd()
		if c.Use == "" {
			t.Errorf("%s: Use metadata empty", tc.name)
		}
		// execute with no args -> expect error due to Args validation (except scaffold prints help but still no error).
		c.SetArgs([]string{})
		_ = c.Execute()
	}
}

func TestNewAddCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewAddCommand(fs, context.Background(), "/kdeps", logging.NewTestLogger())
	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Aliases[0] != "i" {
		t.Fatalf("expected alias 'i'")
	}
	if cmd.Short == "" {
		t.Fatalf("Short description empty")
	}
}

func TestNewRunCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewRunCommand(fs, context.Background(), "/kdeps", nil, logging.NewTestLogger())
	if cmd.Use != "run [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatalf("Short should not be empty")
	}
}

func TestNewPackageAndScaffoldMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{}
	pkgCmd := NewPackageCommand(fs, context.Background(), "/kdeps", env, logging.NewTestLogger())
	if pkgCmd.Use != "package [agent-dir]" {
		t.Fatalf("unexpected package Use: %s", pkgCmd.Use)
	}

	scaffoldCmd := NewScaffoldCommand(fs, context.Background(), logging.NewTestLogger())
	if scaffoldCmd.Use != "scaffold [agentName] [fileNames...]" {
		t.Fatalf("unexpected scaffold Use: %s", scaffoldCmd.Use)
	}
}
