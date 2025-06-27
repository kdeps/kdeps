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

// Injectable functions for testability
var (
	// Subcommand creation functions
	NewAgentCommandFn    = NewAgentCommand
	NewScaffoldCommandFn = NewScaffoldCommand
	NewAddCommandFn      = NewAddCommand
	NewPackageCommandFn  = NewPackageCommand
	NewBuildCommandFn    = NewBuildCommand
	NewRunCommandFn      = NewRunCommand
)

// NewRootCommand returns the root command with all subcommands attached.
func NewRootCommand(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
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
	rootCmd.AddCommand(NewAgentCommandFn(fs, ctx, kdepsDir, logger))
	rootCmd.AddCommand(NewScaffoldCommandFn(fs, ctx, logger))
	rootCmd.AddCommand(NewAddCommandFn(fs, ctx, kdepsDir, logger))
	rootCmd.AddCommand(NewPackageCommandFn(fs, ctx, kdepsDir, env, logger))
	rootCmd.AddCommand(NewBuildCommandFn(fs, ctx, kdepsDir, systemCfg, logger))
	rootCmd.AddCommand(NewRunCommandFn(fs, ctx, kdepsDir, systemCfg, logger))

	return rootCmd
}
