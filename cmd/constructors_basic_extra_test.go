package cmd_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCommandConstructors simply ensures that constructing each top-level Cobra command
// does not panic and returns a non-nil *cobra.Command. This executes the constructor
// logic which improves coverage of the cmd package without executing the command
// handlers themselves (which may require heavy runtime dependencies).
func TestCommandConstructors(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.TODO()
	logger := logging.NewTestLogger()

	tests := []struct {
		name string
		fn   func() interface{}
	}{
		{name: "Add", fn: func() interface{} { return cmd.NewAddCommand(fs, ctx, "", logger) }},
		{name: "Build", fn: func() interface{} { return cmd.NewBuildCommand(fs, ctx, "", nil, logger) }},
		{name: "Package", fn: func() interface{} { return cmd.NewPackageCommand(fs, ctx, "", nil, logger) }},
		{name: "Run", fn: func() interface{} { return cmd.NewRunCommand(fs, ctx, "", nil, logger) }},
		{name: "Scaffold", fn: func() interface{} { return cmd.NewScaffoldCommand(fs, ctx, logger) }},
		{name: "Agent", fn: func() interface{} { return cmd.NewAgentCommand(fs, ctx, "", logger) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("constructor %s panicked: %v", tc.name, r)
				}
			}()

			if cmdVal := tc.fn(); cmdVal == nil {
				t.Fatalf("constructor %s returned nil", tc.name)
			}
		})
	}
}
