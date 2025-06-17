package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

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
