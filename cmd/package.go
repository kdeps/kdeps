package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/workflow"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// Define styles using lipgloss.
var (
	primaryStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// NewPackageCommand creates the 'package' command and passes the necessary dependencies.
func NewPackageCommand(fs afero.Fs, ctx context.Context, kdepsDir string, env *environment.Environment, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "package [agent-dir]",
		Aliases: []string{"p"},
		Example: "$ kdeps package ./myAgent/",
		Short:   "Package an AI agent to .kdeps file",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentDir := args[0]

			// Find the workflow file associated with the agent directory
			wfFile, err := archiver.FindWorkflowFile(fs, agentDir, logger)
			if err != nil {
				return fmt.Errorf("%s: %w", errorStyle.Render("Error finding workflow file"), err)
			}

			// Load the workflow
			wf, err := workflow.LoadWorkflow(ctx, wfFile, logger)
			if err != nil {
				return fmt.Errorf("%s: %w", errorStyle.Render("Error loading workflow"), err)
			}

			// Compile the project
			_, _, err = archiver.CompileProject(fs, ctx, wf, kdepsDir, agentDir, env, logger)
			if err != nil {
				return fmt.Errorf("%s: %w", errorStyle.Render("Error compiling project"), err)
			}

			// Print success message
			fmt.Println(successStyle.Render("AI agent packaged successfully:"), primaryStyle.Render(agentDir))
			return nil
		},
	}
}
