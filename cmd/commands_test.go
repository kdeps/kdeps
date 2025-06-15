package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
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
