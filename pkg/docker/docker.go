package docker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/kdeps/schema/gen/kdeps"
)

func LoadDockerSystem(kdeps *kdeps.Kdeps, id string) (name string, err error) {
	u := uuid.New()
	uid := strings.Replace(u.String(), "-", "", -1)
	uid = uid[:12]

	if len(id) > 0 {
		uid = id
	}

	switch kdeps.DockerGPU {
	case "cpu":
		if name, err = LoadDockerSystemCPU(uid); err != nil {
			return name, err
		}
	case "nvidia":
		if name, err = LoadDockerSystemNvidia(uid); err != nil {
			return name, err
		}
	case "amd":
		if name, err = LoadDockerSystemAMD(uid); err != nil {
			return name, err
		}
	default:
		return name, errors.New(fmt.Sprintf("Docker GPU '%s' unsupported!", kdeps.DockerGPU))
	}

	return name, nil
}

func LoadDockerSystemNvidia(uid string) (string, error) {
	containerName := "kdeps-nvidia-" + uid

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return containerName, err
	}
	cli.NegotiateAPIVersion(context.Background())

	// Create container configuration
	containerConfig := &container.Config{
		Image:        "ollama/ollama",              // Image name
		ExposedPorts: nat.PortSet{"11434/tcp": {}}, // Expose port 11434
	}

	// Host configuration (mapping ports, mounting volumes, and using GPU)
	hostConfig := &container.HostConfig{
		Binds: []string{"ollama:/root/.ollama"}, // Mount volume
		PortBindings: nat.PortMap{
			"11434/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "11434",
				},
			},
		},
		Runtime: "nvidia", // Enable GPU (requires NVIDIA runtime)
		Resources: container.Resources{
			DeviceRequests: []container.DeviceRequest{
				{
					Driver:       "nvidia",
					Count:        -1, // Equivalent to --gpus=all
					Capabilities: [][]string{{"gpu"}},
				},
			},
		},
	}

	// Create the container with the specified name
	resp, err := cli.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,           // Networking options (can be nil for default)
		nil,           // Platform options (can be nil)
		containerName, // Name of the container
	)
	if err != nil {
		return containerName, errors.New(fmt.Sprintf("Error creating container: %v\n", err))
	}

	// Start the container in detached mode
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return containerName, errors.New(fmt.Sprintf("Error starting container: %v\n", err))
	}

	fmt.Printf("Container started with ID: %s\n", resp.ID)
	return containerName, nil
}

func LoadDockerSystemAMD(uid string) (string, error) {
	containerName := "kdeps-amd-" + uid

	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return containerName, err
	}
	cli.NegotiateAPIVersion(context.Background())

	// Define container configuration
	containerConfig := &container.Config{
		Image:        "ollama/ollama:rocm",         // The image name
		ExposedPorts: nat.PortSet{"11434/tcp": {}}, // Expose port 11434
	}

	// Define host configuration (port bindings, devices, and volume mounts)
	hostConfig := &container.HostConfig{
		// Mount volume
		Binds: []string{"ollama:/root/.ollama"},
		// Port bindings
		PortBindings: nat.PortMap{
			"11434/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "11434",
				},
			},
		},
		// Devices to include
		Resources: container.Resources{
			Devices: []container.DeviceMapping{
				{
					PathOnHost:        "/dev/kfd", // Map /dev/kfd from the host
					PathInContainer:   "/dev/kfd", // Path inside the container
					CgroupPermissions: "rwm",      // Read, write, and mknod permissions
				},
				{
					PathOnHost:        "/dev/dri", // Map /dev/dri from the host
					PathInContainer:   "/dev/dri", // Path inside the container
					CgroupPermissions: "rwm",      // Read, write, and mknod permissions
				},
			},
		},
	}

	// Create the container with the specified name
	resp, err := cli.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,           // Networking options (nil for default)
		nil,           // Platform options (nil for default)
		containerName, // Container name
	)
	if err != nil {
		return containerName, errors.New(fmt.Sprintf("Error creating container: %v\n", err))
	}

	// Start the container in detached mode
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return containerName, errors.New(fmt.Sprintf("Error starting container: %v\n", err))
	}

	fmt.Printf("Container started with ID: %s\n", resp.ID)

	return containerName, nil
}

func LoadDockerSystemCPU(uid string) (string, error) {
	containerName := "kdeps-cpu-" + uid

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return containerName, err
	}
	cli.NegotiateAPIVersion(context.Background())

	// Create container configuration
	containerConfig := &container.Config{
		Image:        "ollama/ollama",              // Image name
		ExposedPorts: nat.PortSet{"11434/tcp": {}}, // Expose port 11434
	}

	// Host configuration (mapping ports, mounting volumes, and using GPU)
	hostConfig := &container.HostConfig{
		Binds: []string{"ollama:/root/.ollama"}, // Mount volume
		PortBindings: nat.PortMap{
			"11434/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "11434",
				},
			},
		},
	}

	// Create the container with the specified name
	resp, err := cli.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,           // Networking options (can be nil for default)
		nil,           // Platform options (can be nil)
		containerName, // Name of the container
	)
	if err != nil {
		return containerName, errors.New(fmt.Sprintf("Error creating container: %v\n", err))
	}

	// Start the container in detached mode
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return containerName, errors.New(fmt.Sprintf("Error starting container: %v\n", err))
	}

	fmt.Printf("Container started with ID: %s\n", resp.ID)

	return containerName, nil
}
