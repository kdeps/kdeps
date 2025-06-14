package cmd

import (
	"context"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestNewScaffoldCommand_ListResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)

	// Just ensure it completes without panic when no resource names are supplied.
	cmd.Run(cmd, []string{"myagent"})
}

func TestNewScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	cmd.Run(cmd, []string{"agent", "unknown"}) // should handle gracefully without panic
}

func TestNewScaffoldCommand_GenerateFile(t *testing.T) {
	_ = os.Setenv("NON_INTERACTIVE", "1") // speed

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)

	cmd.Run(cmd, []string{"agentx", "client"})

	// Verify generated file exists
	if ok, _ := afero.Exists(fs, "agentx/resources/client.pkl"); !ok {
		t.Fatalf("expected generated client.pkl file not found")
	}
}
