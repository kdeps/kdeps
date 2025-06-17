package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestNewAddCommandRunE executes the command with a dummy package path. We
// only assert that an error is returned (because the underlying extractor will
// fail with the in-memory filesystem). The objective is to exercise the command
// wiring rather than validate its behaviour.
func TestNewAddCommandRunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := NewAddCommand(fs, context.Background(), "/kdeps", logging.NewTestLogger())

	if err := cmd.RunE(cmd, []string{"dummy.kdeps"}); err == nil {
		t.Fatalf("expected error due to missing package file, got nil")
	}
}
