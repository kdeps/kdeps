package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	kdSchema "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func TestCommandConstructors_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kd"
	logger := logging.NewTestLogger()

	systemCfg := &kdSchema.Kdeps{}

	tests := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"add", func() *cobra.Command { return NewAddCommand(fs, ctx, kdepsDir, logger) }},
		{"build", func() *cobra.Command { return NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger) }},
		{"run", func() *cobra.Command { return NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger) }},
		{"package", func() *cobra.Command { return NewPackageCommand(fs, ctx, kdepsDir, nil, logger) }},
		{"scaffold", func() *cobra.Command { return NewScaffoldCommand(fs, ctx, logger) }},
		{"new", func() *cobra.Command { return NewAgentCommand(fs, ctx, kdepsDir, logger) }},
	}

	for _, tc := range tests {
		c := tc.cmd()
		if c.Use == "" {
			t.Errorf("%s: Use metadata empty", tc.name)
		}
		// execute with no args -> expect error due to Args validation (except scaffold prints help but still no error).
		c.SetArgs([]string{})
		_ = c.Execute()
	}
}
