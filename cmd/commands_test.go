package cmd_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
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
		{"add", cmd.NewAddCommand(fs, ctx, dir, logger)},
		{"build", cmd.NewBuildCommand(fs, ctx, dir, nil, logger)},
		{"run", cmd.NewRunCommand(fs, ctx, dir, nil, logger)},
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

	c := cmd.NewAgentCommand(fs, ctx, dir, logger)
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

	buildCmd := cmd.NewBuildCommand(fs, ctx, dir, nil, logger)
	if err := execCommand(buildCmd, nonExist); err == nil {
		t.Errorf("BuildCommand expected error for missing file, got nil")
	}

	runCmd := cmd.NewRunCommand(fs, ctx, dir, nil, logger)
	if err := execCommand(runCmd, nonExist); err == nil {
		t.Errorf("RunCommand expected error for missing file, got nil")
	}
}
