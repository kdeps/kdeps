package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"

	kdepsschema "github.com/kdeps/schema/gen/kdeps"
)

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
