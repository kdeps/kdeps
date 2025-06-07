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
func NewAgentCommand(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
	newCmd := &cobra.Command{
		Use:     "new [agentName]",
		Aliases: []string{"n"},
		Short:   "Create a new AI agent",
		Args:    cobra.MaximumNArgs(1), // Allow at most one argument (agentName)
		Run: func(cmd *cobra.Command, args []string) {
			var agentName string
			if len(args) > 0 {
				agentName = args[0]
			}

			// Pass the agentName to GenerateAgent
			if err := template.GenerateAgent(fs, ctx, logger, "", agentName); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	return newCmd
}
