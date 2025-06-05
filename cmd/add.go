package cmd

import (
	"context"
	"fmt"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// Allow override for testing.
var extractPackage = archiver.ExtractPackage

// runAdd contains the logic for installing an AI agent, factored out for testability.
func runAdd(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) error {
	_, err := extractPackage(fs, ctx, kdepsDir, pkgFile, logger)
	if err != nil {
		return err
	}
	fmt.Println("AI agent installed locally:", pkgFile)
	return nil
}

// NewInstallCommand creates the 'install' command and passes the necessary dependencies.
func NewInstallCommand(fs afero.Fs, ctx context.Context, kdepsDir string, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "install [package]",
		Aliases: []string{"i"},
		Example: "$ kdeps install ./myAgent.kdeps",
		Short:   "Install an AI agent locally",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(fs, ctx, kdepsDir, args[0], logger)
		},
	}
}
