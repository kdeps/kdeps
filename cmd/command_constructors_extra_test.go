package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

// helper returns common deps for command constructors.
func testDeps() (afero.Fs, context.Context, string, *logging.Logger) {
	return afero.NewMemMapFs(), context.Background(), "/tmp/kdeps", logging.NewTestLogger()
}

func TestNewAddCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewAddCommand(fs, ctx, dir, logger)
	if cmd.Use != "install [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// RunE with a non-existent file to exercise error path but cover closure.
	if err := cmd.RunE(cmd, []string{"/no/file.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewBuildCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewBuildCommand(fs, ctx, dir, &kdCfg.Kdeps{}, logger)
	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewAgentCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewAgentCommand(fs, ctx, dir, logger)
	if cmd.Use != "new [agentName]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// Provide invalid args to hit error path.
	if err := cmd.RunE(cmd, []string{""}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewPackageCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewPackageCommand(fs, ctx, dir, &environment.Environment{}, logger)
	if cmd.Use != "package [agent-dir]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"/nonexistent"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewRunCommandConstructor(t *testing.T) {
	fs, ctx, dir, logger := testDeps()
	cmd := NewRunCommand(fs, ctx, dir, &kdCfg.Kdeps{}, logger)
	if cmd.Use != "run [package]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	if err := cmd.RunE(cmd, []string{"nonexistent.kdeps"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewScaffoldCommandConstructor(t *testing.T) {
	fs, _, _, logger := testDeps()
	cmd := NewScaffoldCommand(fs, context.Background(), logger)
	if cmd.Use != "scaffold [agentName] [fileNames...]" {
		t.Fatalf("unexpected Use field: %s", cmd.Use)
	}

	// args missing triggers help path, fast.
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected error")
	}
}
