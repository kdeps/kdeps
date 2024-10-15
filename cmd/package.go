package cmd

import (
	"context"
	"fmt"
	"kdeps/pkg/archiver"
	"kdeps/pkg/environment"
	"kdeps/pkg/workflow"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewPackageCommand creates the 'package' command and passes the necessary dependencies
func NewPackageCommand(fs afero.Fs, ctx context.Context, kdepsDir string, env *environment.Environment, logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "package [agent-dir]",
		Aliases: []string{"p"},
		Example: "$ kdeps package ./myAgent/",
		Short:   "Package an AI agent to .kdeps file",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentDir := args[0]
			// Add your packaging logic here
			wfFile, err := archiver.FindWorkflowFile(fs, agentDir, logger)
			if err != nil {
				return err
			}
			wf, err := workflow.LoadWorkflow(ctx, wfFile, logger)
			if err != nil {
				return err
			}
			_, _, err = archiver.CompileProject(fs, ctx, wf, kdepsDir, agentDir, env, logger)
			if err != nil {
				return err
			}
			fmt.Println("AI agent packaged:", agentDir)
			return nil
		},
	}
}
