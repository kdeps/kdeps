package cmd

import (
	"context"
	"fmt"
	"kdeps/pkg/template"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewNewCommand creates the 'new' command and passes the necessary dependencies
func NewAgentCommand(fs afero.Fs, ctx context.Context, kdepsDir string, logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create a new AI agent",
		Run: func(cmd *cobra.Command, args []string) {
			if err := template.GenerateAgent(fs, logger); err != nil {
				fmt.Println(err)
			}
		},
	}
}
