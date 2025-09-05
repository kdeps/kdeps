package cmd

import (
	"context"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRootCommand returns the root command with all subcommands attached.
func NewRootCommand(ctx context.Context, fs afero.Fs, kdepsDir string, systemCfg *kdeps.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
	cobra.EnableCommandSorting = false
	rootCmd := &cobra.Command{
		Use:   "kdeps",
		Short: "Multi-model AI agent framework.",
		Long: `Kdeps is a multi-model AI agent framework that is optimized for creating purpose-built
Dockerized AI agent APIs ready to be deployed in any organization. It utilizes self-contained
open-source LLM models that are orchestrated by a graph-based dependency workflow.`,
		Version: v.Version,
	}
	rootCmd.PersistentFlags().BoolVarP(&schema.UseLatest, "latest", "l", false,
		`Fetch and use the latest schema and libraries. It is recommended to set the GITHUB_TOKEN environment
variable to prevent errors caused by rate limit exhaustion.`)
	rootCmd.AddCommand(NewAgentCommand(ctx, fs, kdepsDir, logger))
	rootCmd.AddCommand(NewScaffoldCommand(ctx, fs, logger))
	rootCmd.AddCommand(NewAddCommand(ctx, fs, kdepsDir, logger))
	rootCmd.AddCommand(NewPackageCommand(ctx, fs, kdepsDir, env, logger))
	rootCmd.AddCommand(NewBuildCommand(ctx, fs, kdepsDir, systemCfg, logger))
	rootCmd.AddCommand(NewRunCommand(ctx, fs, kdepsDir, systemCfg, logger))
	rootCmd.AddCommand(UpgradeCommand(ctx, fs, kdepsDir, logger))

	return rootCmd
}
