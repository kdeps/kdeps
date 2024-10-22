package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/afero"
)

func CreateDockerContainer(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum, gpu string, apiMode bool, cli *client.Client) (string, error) {
	// Run the Docker container with volume and port configuration
	containerConfig := &container.Config{
		Image: containerName,
	}

	tcpPort := fmt.Sprintf("%s/tcp", portNum)
	hostConfig := &container.HostConfig{
		Binds: []string{"kdeps:/root/.ollama"},
		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port(tcpPort): {{HostIP: hostIP, HostPort: portNum}},
		},
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure", // Restart the container only on failure
			MaximumRetryCount: 5,            // Optionally specify the max retry count
		},
	}

	// Optional mode for API-based configuration
	if !apiMode {
		hostConfig = &container.HostConfig{
			Binds: []string{"kdeps:/root/.ollama"},
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 5,
			},
		}
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, fmt.Sprintf("%s-%s", cName, gpu))
	if err != nil {
		return "", err
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", err
	}

	fmt.Println("Kdeps container is running.")

	return resp.ID, nil
}
