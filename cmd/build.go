package cmd

import (
	"context"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewBuildCommand creates the 'build' command and passes the necessary dependencies.
func NewBuildCommand(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "build [package]",
		Aliases: []string{"b"},
		Example: "$ kdeps build ./myAgent.kdeps",
		Short:   "Build a dockerized AI agent",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgFile := args[0]
			// Use the passed dependencies
			pkgProject, err := ExtractPackageFn(fs, ctx, kdepsDir, pkgFile, logger)
			if err != nil {
				return err
			}
			runDir, _, _, _, _, _, _, _, err := BuildDockerfileFn(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
			if err != nil {
				return err
			}
			dockerClient, err := NewDockerClientFn()
			if err != nil {
				return err
			}
			agentContainerName, agentContainerNameAndVersion, err := BuildDockerImageFn(fs, ctx, systemCfg, dockerClient, runDir, kdepsDir, pkgProject, logger)
			if err != nil {
				return err
			}

			if err := CleanupDockerBuildImagesFn(fs, ctx, agentContainerName, dockerClient); err != nil {
				return err
			}
			PrintlnFn("Kdeps AI Agent docker image created:", agentContainerNameAndVersion)
			return nil
		},
	}
}
