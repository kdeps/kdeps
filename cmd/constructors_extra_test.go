package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

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
	systemCfg := &kdeps.Kdeps{}
	cmd := NewBuildCommand(fs, context.Background(), "/tmp/kdeps", systemCfg, logging.NewTestLogger())

	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}

	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "b" {
		t.Fatalf("expected alias 'b', got %v", cmd.Aliases)
	}
}
