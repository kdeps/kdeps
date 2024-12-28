package cmd

import (
	"fmt"
	"kdeps/pkg/template"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewScaffoldCommand creates the 'scaffold' subcommand for generating specific agent files
func NewScaffoldCommand(fs afero.Fs, logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold [agentName] [fileName]",
		Short: "Scaffold a specific file for an agent",
		Args:  cobra.MaximumNArgs(2), // Allow up to two arguments (agentName, fileName)
		Run: func(cmd *cobra.Command, args []string) {
			var agentName, fileName string
			if len(args) > 0 {
				agentName = args[0]
			}
			if len(args) > 1 {
				fileName = args[1]
			}

			if err := template.GenerateSpecificAgentFile(fs, logger, agentName, fileName); err != nil {
				logger.Error("Error scaffolding file: ", err)
				fmt.Println("Error:", err)
			}
		},
	}
}
