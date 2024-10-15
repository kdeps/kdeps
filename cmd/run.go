package cmd

import (
	"context"
	"fmt"
	"kdeps/pkg/archiver"
	"kdeps/pkg/docker"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/client"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRunCommand creates the 'run' command and passes the necessary dependencies
func NewRunCommand(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, logger *log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "run [package]",
		Aliases: []string{"r"},
		Example: "$ kdeps run ./myAgent.kdeps",
		Short:   "Build and run a dockerized AI agent container",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgFile := args[0]
			// Add your logic to run the docker container here
			pkgProject, err := archiver.ExtractPackage(fs, ctx, kdepsDir, pkgFile, logger)
			if err != nil {
				return err
			}
			runDir, apiServerMode, hostIP, hostPort, gpuType, err := docker.BuildDockerfile(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
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
			containerID, err := docker.CreateDockerContainer(fs, ctx, agentContainerName, agentContainerNameAndVersion, hostIP, hostPort, gpuType, apiServerMode, dockerClient)
			if err != nil {
				return err
			}
			fmt.Println("Kdeps AI Agent docker container created:", containerID)
			return nil
		},
	}
}
