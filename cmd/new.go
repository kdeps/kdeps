package cmd

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewAgentCommand creates the 'new' command and passes the necessary dependencies.
func NewAgentCommand(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [agentName]",
		Aliases: []string{"n"},
		Short:   "Create a new AI agent",
		Args:    cobra.ExactArgs(1), // Require exactly one argument (agentName)
		RunE: func(cmd *cobra.Command, args []string) error {
			agentName := args[0]

			// Create the main directory under baseDir
			mainDir := agentName
			if err := fs.MkdirAll(mainDir, 0o755); err != nil {
				return fmt.Errorf("failed to create main directory: %w", err)
			}

			// Generate workflow file
			if err := GenerateWorkflowFileFn(fs, ctx, logger, mainDir, agentName); err != nil {
				return fmt.Errorf("failed to generate workflow file: %w", err)
			}

			// Generate resource files
			if err := GenerateResourceFilesFn(fs, ctx, logger, mainDir, agentName); err != nil {
				return fmt.Errorf("failed to generate resource files: %w", err)
			}

			return nil
		},
	}

	return cmd
}
