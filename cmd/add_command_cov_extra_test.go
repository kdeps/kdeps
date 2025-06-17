package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestNewAddCommand_RunE ensures the command is wired correctly – we expect an
// error because the provided package path doesn't exist, but the purpose of
// the test is simply to execute the RunE handler to mark lines as covered.
func TestNewAddCommand_RunE(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewAddCommand(fs, ctx, "/kdeps", logger)

	// Supply non-existent path so that ExtractPackage fails and RunE returns
	// an error. Success isn't required – only execution.
	if err := cmd.RunE(cmd, []string{"/does/not/exist.kdeps"}); err == nil {
		t.Fatalf("expected error from RunE due to missing package file")
	}
}
