package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestNewBuildCommandMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewBuildCommand(fs, context.Background(), "/kdeps", nil, logging.NewTestLogger())

	if cmd.Use != "build [package]" {
		t.Fatalf("unexpected Use: %s", cmd.Use)
	}
	if len(cmd.Aliases) == 0 || cmd.Aliases[0] != "b" {
		t.Fatalf("expected alias 'b'")
	}
	if cmd.Short == "" {
		t.Fatalf("Short description should not be empty")
	}
}
