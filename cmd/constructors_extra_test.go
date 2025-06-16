package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	kschema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestCommandConstructors verifies each Cobra constructor returns a non-nil *cobra.Command
// with the expected Use string populated. We don't execute the RunE handlers -
// just calling the constructor is enough to cover its statements.
func TestCommandConstructors(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "kdeps_cmd")
	if err != nil {
		t.Fatalf("TempDir error: %v", err)
	}

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Environment needed for NewPackageCommand
	env, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("env error: %v", err)
	}

	// Dummy config object for Build / Run commands
	dummyCfg := &kschema.Kdeps{}

	cases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"add", NewAddCommand(fs, ctx, tmpDir, logger)},
		{"build", NewBuildCommand(fs, ctx, tmpDir, dummyCfg, logger)},
		{"new", NewAgentCommand(fs, ctx, tmpDir, logger)},
		{"package", NewPackageCommand(fs, ctx, tmpDir, env, logger)},
		{"run", NewRunCommand(fs, ctx, tmpDir, dummyCfg, logger)},
		{"scaffold", NewScaffoldCommand(fs, ctx, logger)},
	}

	for _, c := range cases {
		if c.cmd == nil {
			t.Fatalf("%s: constructor returned nil", c.name)
		}
		if c.cmd.Use == "" {
			t.Fatalf("%s: Use string empty", c.name)
		}
	}
}

func TestNewAddCommand_Meta(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewAddCommand(fs, context.Background(), "/tmp/kdeps", logging.NewTestLogger())

	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "i" {
		t.Fatalf("expected alias 'i', got %v", cmd.Aliases)
	}
}

func TestNewBuildCommand_Meta(t *testing.T) {
	fs := afero.NewMemMapFs()
	systemCfg := &kschema.Kdeps{}
	cmd := NewBuildCommand(fs, context.Background(), "/tmp/kdeps", systemCfg, logging.NewTestLogger())

	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "b" {
		t.Fatalf("expected alias 'b', got %v", cmd.Aliases)
	}
}
