package cmd

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewBuildCommand creates the 'build' command and passes the necessary dependencies.
func NewBuildCommand(ctx context.Context, fs afero.Fs, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "build [package]",
		Aliases: []string{"b"},
		Example: "$ kdeps build ./myAgent.kdeps",
		Short:   "Build a dockerized AI agent",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgFile := args[0]
			// Use the passed dependencies
			pkgProject, err := archiver.ExtractPackage(fs, ctx, kdepsDir, pkgFile, logger)
			if err != nil {
				return err
			}
			runDir, _, _, _, _, _, _, _, err := docker.BuildDockerfile(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
			if err != nil {
				return err
			}
			dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			agentContainerName, agentContainerNameAndVersion, err := docker.BuildDockerImage(fs, ctx, systemCfg, dockerClient, runDir, kdepsDir, pkgProject, logger)
			if err != nil {
				return err
			}

			if err := docker.CleanupDockerBuildImages(fs, ctx, agentContainerName, dockerClient); err != nil {
				return err
			}
			logger.Info("Kdeps AI Agent docker image created", "image", agentContainerNameAndVersion)
			return nil
		},
	}
}
