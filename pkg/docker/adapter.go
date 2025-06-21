package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

// DockerClientAdapter wraps the Docker SDK client to match our DockerClient interface
type DockerClientAdapter struct {
	client *client.Client
}

// NewDockerClientAdapter creates a new adapter for the Docker SDK client
func NewDockerClientAdapter(dockerClient *client.Client) *DockerClientAdapter {
	return &DockerClientAdapter{client: dockerClient}
}

// ContainerCreate adapts the Docker SDK ContainerCreate method to match our interface
func (a *DockerClientAdapter) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform interface{}, containerName string) (container.CreateResponse, error) {
	// The Docker SDK expects *specs.Platform, but we accept interface{} for flexibility
	// Pass nil for platform to use default behavior
	return a.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
}

// ContainerStart adapts the Docker SDK ContainerStart method
func (a *DockerClientAdapter) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return a.client.ContainerStart(ctx, containerID, options)
}

// ContainerList adapts the Docker SDK ContainerList method
func (a *DockerClientAdapter) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return a.client.ContainerList(ctx, options)
}
