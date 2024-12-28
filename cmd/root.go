package cmd

import (
	"context"
	"kdeps/pkg/environment"

	"github.com/charmbracelet/log"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRootCommand returns the root command with all subcommands attached
func NewRootCommand(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, env *environment.Environment, logger *log.Logger) *cobra.Command {
	cobra.EnableCommandSorting = false

	rootCmd := &cobra.Command{
		Use:   "kdeps",
		Short: "Multi-model AI agent framework.",
		Long: `Kdeps is a multi-model AI agent framework that is optimized for creating purpose-built
Dockerized AI agent APIs ready to be deployed in any organization. It utilizes self-contained
open-source LLM models that are orchestrated by a graph-based dependency workflow.`,
	}

	rootCmd.AddCommand(NewAgentCommand(fs, ctx, kdepsDir, logger))
	rootCmd.AddCommand(NewScaffoldCommand(fs, logger))
	rootCmd.AddCommand(NewAddCommand(fs, ctx, kdepsDir, logger))
	rootCmd.AddCommand(NewPackageCommand(fs, ctx, kdepsDir, env, logger))
	rootCmd.AddCommand(NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger))
	rootCmd.AddCommand(NewRunCommand(fs, ctx, kdepsDir, systemCfg, logger))

	return rootCmd
}
