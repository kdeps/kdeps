package docker

import (
	"bytes"
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/joho/godotenv"
	"github.com/spf13/afero"
)

func CreateDockerContainer(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum, gpu string, apiMode bool, cli *client.Client) (string, error) {
	// Load environment variables from .env file (if it exists)
	envSlice, err := loadEnvFile(fs, ".env")
	if err != nil {
		fmt.Println("Error loading .env file, proceeding without it:", err)
	}

	// Configure the Docker container
	containerConfig := &container.Config{
		Image: containerName,
		Env:   envSlice, // Add the loaded environment variables (or nil)
	}

	tcpPort := fmt.Sprintf("%s/tcp", portNum)
	hostConfig := &container.HostConfig{
		Binds: []string{
			"ollama:/root/.ollama",
			"kdeps:/root/.kdeps",
		},
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
			Binds: []string{
				"ollama:/root/.ollama",
				"kdeps:/root/.kdeps",
			},
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 5,
			},
		}
	}

	// Adjust host configuration based on GPU type
	switch gpu {
	case "amd":
		hostConfig.DeviceRequests = []container.DeviceRequest{
			{
				Capabilities: [][]string{{"gpu"}},
			},
		}
		hostConfig.Devices = []container.DeviceMapping{
			{PathOnHost: "/dev/kfd", PathInContainer: "/dev/kfd", CgroupPermissions: "rwm"},
			{PathOnHost: "/dev/dri", PathInContainer: "/dev/dri", CgroupPermissions: "rwm"},
		}
	case "nvidia":
		hostConfig.DeviceRequests = []container.DeviceRequest{
			{
				Capabilities: [][]string{{"gpu"}},
				Count:        -1, // Use all available GPUs
			},
		}
		hostConfig.Binds = append(hostConfig.Binds, "/nvidia:/root/.nvidia")
	case "cpu":
		// No additional configuration needed for CPU
	}

	// Check if the container already exists
	containerNameWithGpu := fmt.Sprintf("%s-%s", cName, gpu)

	// Generate Docker Compose file
	err = GenerateDockerCompose(fs, cName, containerName, containerNameWithGpu, hostIP, portNum, gpu)
	if err != nil {
		return "", fmt.Errorf("error generating Docker Compose file: %w", err)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("error listing containers: %w", err)
	}

	for _, resp := range containers {
		for _, name := range resp.Names {
			if name == fmt.Sprintf("/%s", containerNameWithGpu) {
				// If the container exists, start it if it's not running
				if resp.State != "running" {
					err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
					if err != nil {
						return "", fmt.Errorf("error starting existing container: %w", err)
					}
					fmt.Println("Started existing container:", containerNameWithGpu)
				} else {
					fmt.Println("Container is already running:", containerNameWithGpu)
				}
				return resp.ID, nil
			}
		}
	}

	// Create a new container if it doesn't exist
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerNameWithGpu)
	if err != nil {
		return "", fmt.Errorf("error creating container: %w", err)
	}

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("error starting new container: %w", err)
	}

	fmt.Println("Kdeps container is running:", containerNameWithGpu)

	return resp.ID, nil
}

func loadEnvFile(fs afero.Fs, filename string) ([]string, error) {
	// Check if the file exists
	exists, err := afero.Exists(fs, filename)
	if err != nil {
		return nil, fmt.Errorf("error checking file existence: %w", err)
	}

	if !exists {
		// If the file doesn't exist, return an empty slice
		fmt.Printf("%s does not exist, skipping .env loading.\n", filename)
		return nil, nil
	}

	// Read the file content
	content, err := afero.ReadFile(fs, filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Parse the .env content
	envMap, err := godotenv.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("error parsing .env content: %w", err)
	}

	// Convert map to slice of strings in "key=value" format
	var envSlice []string
	for key, value := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}

	return envSlice, nil
}

func GenerateDockerCompose(fs afero.Fs, cName, containerName, containerNameWithGpu, hostIP, portNum, gpu string) error {
	var gpuConfig string

	// GPU-specific configurations
	switch gpu {
	case "amd":
		gpuConfig = `
	deploy:
	  resources:
	    reservations:
	      devices:
		- capabilities:
		    - gpu
	devices:
	  - "/dev/kfd:/dev/kfd:rwm"
	  - "/dev/dri:/dev/dri:rwm"
`
	case "nvidia":
		gpuConfig = `
	deploy:
	  resources:
	    reservations:
	      devices:
		- capabilities:
		    - gpu
	volumes:
	  - /nvidia:/root/.nvidia
`
	case "cpu":
		gpuConfig = ""
	default:
		return fmt.Errorf("unsupported GPU type: %s", gpu)
	}

	// Compose file content
	dockerComposeContent := fmt.Sprintf(`
# This Docker Compose file runs the Kdeps AI Agent containerized service with GPU configurations.
# To use it:
# 1. Start the service with the command:
#    docker-compose --file <filename> up -d
# 3. The service will start with the specified GPU configuration
#    and will be accessible on the configured host IP and port.

version: '3.8'
services:
  %s:
    image: %s
    ports:
      - "%s:%s"
    restart: on-failure
    volumes:
      - ollama:/root/.ollama
      - kdeps:/root/.kdeps
%s
volumes:
  ollama:
  kdeps:
`, containerNameWithGpu, containerName, portNum, portNum, gpuConfig)

	filePath := fmt.Sprintf("%s_docker-compose.yaml", cName)
	err := afero.WriteFile(fs, filePath, []byte(dockerComposeContent), 0o644)
	if err != nil {
		return fmt.Errorf("error writing Docker Compose file: %w", err)
	}

	fmt.Println("Docker Compose file generated successfully at:", filePath)
	return nil
}
