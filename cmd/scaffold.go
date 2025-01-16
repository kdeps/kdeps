package cmd

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewScaffoldCommand creates the 'scaffold' subcommand for generating specific agent files.
func NewScaffoldCommand(fs afero.Fs, logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "scaffold [agentName] [fileNames...]",
		Short: "Scaffold specific files for an agent",
		Args:  cobra.MinimumNArgs(2), // Require at least two arguments (agentName and at least one fileName)
		Run: func(cmd *cobra.Command, args []string) {
			agentName := args[0]
			fileNames := args[1:]

			for _, fileName := range fileNames {
				if err := template.GenerateSpecificAgentFile(fs, logger, agentName, fileName); err != nil {
					logger.Error("Error scaffolding file:", err)
					fmt.Println("Error:", err)
				} else {
					fmt.Printf("Successfully scaffolded file: %s\n", fileName)
				}
			}
		},
	}
}
