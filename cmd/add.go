package cmd

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewAddCommand creates the 'add' command and passes the necessary dependencies.
func NewAddCommand(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "install [package]",
		Aliases: []string{"i"},
		Example: "$ kdeps install ./myAgent.kdeps",
		Short:   "Install an AI agent locally",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgFile := args[0]
			// Use the passed dependencies
			_, err := archiver.ExtractPackage(fs, ctx, kdepsDir, pkgFile, logger)
			if err != nil {
				return err
			}
			fmt.Println("AI agent installed locally:", pkgFile)
			return nil
		},
	}
}
