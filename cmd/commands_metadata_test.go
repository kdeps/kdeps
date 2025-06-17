package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

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
