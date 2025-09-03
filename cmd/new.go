package cmd

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewAgentCommand creates the 'new' command and passes the necessary dependencies.
func NewAgentCommand(ctx context.Context, fs afero.Fs, _ string, logger *logging.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new [agentName]",
		Aliases: []string{"n"},
		Short:   "Create a new AI agent",
		Args:    cobra.ExactArgs(1), // Require exactly one argument (agentName)
		RunE: func(_ *cobra.Command, args []string) error {
			return executeNewCommand(ctx, fs, logger, args[0])
		},
	}

	return cmd
}

func executeNewCommand(ctx context.Context, fs afero.Fs, logger *logging.Logger, agentName string) error {
	mainDir := agentName

	if err := createAgentDirectory(fs, mainDir); err != nil {
		return err
	}

	if err := generateAgentWorkflow(ctx, fs, logger, mainDir, agentName); err != nil {
		return err
	}

	return generateAgentResources(ctx, fs, logger, mainDir, agentName)
}

func createAgentDirectory(fs afero.Fs, mainDir string) error {
	if err := fs.MkdirAll(mainDir, 0o755); err != nil {
		return fmt.Errorf("failed to create main directory: %w", err)
	}
	return nil
}

func generateAgentWorkflow(ctx context.Context, fs afero.Fs, logger *logging.Logger, mainDir, agentName string) error {
	if err := template.GenerateWorkflowFile(ctx, fs, logger, mainDir, agentName); err != nil {
		return fmt.Errorf("failed to generate workflow file: %w", err)
	}
	return nil
}

func generateAgentResources(ctx context.Context, fs afero.Fs, logger *logging.Logger, mainDir, agentName string) error {
	if err := template.GenerateResourceFiles(ctx, fs, logger, mainDir, agentName); err != nil {
		return fmt.Errorf("failed to generate resource files: %w", err)
	}
	return nil
}
