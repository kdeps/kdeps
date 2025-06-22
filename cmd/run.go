package cmd

import (
	"context"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRunCommand creates the 'run' command and passes the necessary dependencies.
func NewRunCommand(fs afero.Fs, ctx context.Context, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "run [package]",
		Aliases: []string{"r"},
		Example: "$ kdeps run ./myAgent.kdeps",
		Short:   "Build and run a dockerized AI agent container",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkgFile := args[0]
			// Add your logic to run the docker container here
			pkgProject, err := ExtractPackageFn(fs, ctx, kdepsDir, pkgFile, logger)
			if err != nil {
				return err
			}
			runDir, APIServerMode, WebServerMode, hostIP, hostPort, webHostIP, webHostNum, gpuType, err := BuildDockerfileFn(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
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
			// Use the adapter to match our DockerClient interface
			dockerClientAdapter := NewDockerClientAdapterFn(dockerClient)
			containerID, err := CreateDockerContainerFn(fs, ctx, agentContainerName,
				agentContainerNameAndVersion, hostIP, hostPort, webHostIP, webHostNum, gpuType,
				APIServerMode, WebServerMode, dockerClientAdapter)
			if err != nil {
				return err
			}
			PrintlnFn("Kdeps AI Agent docker container created:", containerID)
			return nil
		},
	}
}
