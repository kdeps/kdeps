package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// NewRunCommand creates the 'run' command and passes the necessary dependencies.
func NewRunCommand(ctx context.Context, fs afero.Fs, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "run [package]",
		Aliases: []string{"r"},
		Example: "$ kdeps run ./myAgent.kdeps",
		Short:   "Build and run a dockerized AI agent container",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return executeRunCommand(ctx, fs, kdepsDir, systemCfg, logger, args[0])
		},
	}
}

func executeRunCommand(ctx context.Context, fs afero.Fs, kdepsDir string, systemCfg *kdeps.Kdeps, logger *logging.Logger, pkgFile string) error {
	pkgProject, err := extractAndPreparePackage(fs, ctx, kdepsDir, pkgFile, logger)
	if err != nil {
		return err
	}

	dockerConfig, err := buildDockerSetup(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
	if err != nil {
		return err
	}

	return createAndRunContainer(ctx, fs, systemCfg, dockerConfig, logger)
}

func extractAndPreparePackage(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
	return archiver.ExtractPackage(fs, ctx, kdepsDir, pkgFile, logger)
}

type DockerConfiguration struct {
	RunDir               string
	APIServerMode        bool
	WebServerMode        bool
	HostIP               string
	HostPort             string
	WebHostIP            string
	WebHostNum           string
	GPUType              string
	DockerClient         *client.Client
	ContainerName        string
	ContainerNameVersion string
}

func buildDockerSetup(fs afero.Fs, ctx context.Context, systemCfg *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (*DockerConfiguration, error) {
	runDir, APIServerMode, WebServerMode, hostIP, hostPort, webHostIP, webHostNum, gpuType, err := docker.BuildDockerfile(fs, ctx, systemCfg, kdepsDir, pkgProject, logger)
	if err != nil {
		return nil, err
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerConfiguration{
		RunDir:        runDir,
		APIServerMode: APIServerMode,
		WebServerMode: WebServerMode,
		HostIP:        hostIP,
		HostPort:      hostPort,
		WebHostIP:     webHostIP,
		WebHostNum:    webHostNum,
		GPUType:       gpuType,
		DockerClient:  dockerClient,
	}, nil
}

func createAndRunContainer(ctx context.Context, fs afero.Fs, systemCfg *kdeps.Kdeps, config *DockerConfiguration, logger *logging.Logger) error {
	agentContainerName, agentContainerNameAndVersion, err := docker.BuildDockerImage(fs, ctx, systemCfg, config.DockerClient, config.RunDir, "", nil, logger)
	if err != nil {
		return err
	}

	if err := docker.CleanupDockerBuildImages(fs, ctx, agentContainerName, config.DockerClient); err != nil {
		return err
	}

	containerID, err := docker.CreateDockerContainer(fs, ctx, agentContainerName,
		agentContainerNameAndVersion, config.HostIP, config.HostPort, config.WebHostIP,
		config.WebHostNum, config.GPUType, config.APIServerMode, config.WebServerMode, config.DockerClient)
	if err != nil {
		return err
	}

	fmt.Println("Kdeps AI Agent docker container created:", containerID) //nolint:forbidigo // CLI user feedback
	return nil
}
