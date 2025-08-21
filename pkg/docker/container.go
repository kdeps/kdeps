package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/joho/godotenv"
	"github.com/spf13/afero"
)

func CreateDockerContainer(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum, webHostIP,
	webPortNum, gpu string, apiMode, webMode bool, cli *client.Client,
) (string, error) {
	// Load environment variables from .env file (if it exists)
	envSlice, err := loadEnvFile(fs, ".env")
	if err != nil {
		fmt.Println("Error loading .env file, proceeding without it:", err)
	}

	// Validate port numbers based on modes
	if apiMode && portNum == "" {
		return "", errors.New("portNum must be non-empty when apiMode is true")
	}
	if webMode && webPortNum == "" {
		return "", errors.New("webPortNum must be non-empty when webMode is true")
	}

	// Configure the Docker container
	containerConfig := &container.Config{
		Image: containerName,
		Env:   envSlice, // Add the loaded environment variables (or nil)
	}

	// Set up port bindings based on apiMode and webMode independently
	portBindings := map[nat.Port][]nat.PortBinding{}
	if apiMode && hostIP != "" && portNum != "" {
		tcpPort := portNum + "/tcp"
		portBindings[nat.Port(tcpPort)] = []nat.PortBinding{{HostIP: hostIP, HostPort: portNum}}
	}
	if webMode && webHostIP != "" && webPortNum != "" {
		webTCPPort := webPortNum + "/tcp"
		portBindings[nat.Port(webTCPPort)] = []nat.PortBinding{{HostIP: webHostIP, HostPort: webPortNum}}
	}

	// Initialize hostConfig with default settings
	hostConfig := &container.HostConfig{
		Binds: []string{
			"ollama:/root/.ollama",
			"kdeps:/agent/volume",
		},
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name:              "on-failure",
			MaximumRetryCount: 5,
		},
	}

	// Adjust host configuration based on GPU type
	switch gpu {
	case "amd":
		hostConfig.Devices = []container.DeviceMapping{
			{PathOnHost: "/dev/kfd", PathInContainer: "/dev/kfd", CgroupPermissions: "rwm"},
			{PathOnHost: "/dev/dri", PathInContainer: "/dev/dri", CgroupPermissions: "rwm"},
		}
	case "nvidia":
		hostConfig.DeviceRequests = []container.DeviceRequest{
			{
				Driver:       "nvidia",
				Capabilities: [][]string{{"gpu"}},
				Count:        -1, // Use all available GPUs
			},
		}
	case "cpu":
		// No additional configuration needed for CPU
	}

	// Check if the container already exists
	containerNameWithGpu := fmt.Sprintf("%s-%s", cName, gpu)

	// Generate Docker Compose file
	err = GenerateDockerCompose(fs, cName, containerName, containerNameWithGpu, hostIP, portNum, webHostIP, webPortNum, apiMode, webMode, gpu)
	if err != nil {
		return "", fmt.Errorf("error generating Docker Compose file: %w", err)
	}

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("error listing containers: %w", err)
	}

	// Use integer range over slice directly (Go 1.22+)
	for i := range containers {
		resp := containers[i]
		for _, name := range resp.Names {
			if name == "/"+containerNameWithGpu {
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

	// Preallocate the slice with the exact size of the map
	envSlice := make([]string, 0, len(envMap))
	for key, value := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}

	return envSlice, nil
}

func GenerateDockerCompose(fs afero.Fs, cName, containerName, containerNameWithGpu, hostIP, portNum, webHostIP, webPortNum string, apiMode, webMode bool, gpu string) error {
	var gpuConfig string

	// GPU-specific configurations
	switch gpu {
	case "amd":
		gpuConfig = `
	devices:
	  - /dev/kfd
	  - /dev/dri
`
	case "nvidia":
		gpuConfig = `
	deploy:
	  resources:
	    reservations:
	      devices:
		- driver: nvidia
		  count: all
		  capabilities: [gpu]
`
	case "cpu":
		gpuConfig = ""
	default:
		return fmt.Errorf("unsupported GPU type: %s", gpu)
	}

	// Build ports section based on apiMode and webMode independently
	var ports []string
	if apiMode && hostIP != "" && portNum != "" {
		ports = append(ports, fmt.Sprintf("%s:%s", hostIP, portNum))
	}
	if webMode && webHostIP != "" && webPortNum != "" {
		ports = append(ports, fmt.Sprintf("%s:%s", webHostIP, webPortNum))
	}

	// Format ports section for YAML
	var portsSection string
	if len(ports) > 0 {
		portsSection = "    ports:\n"
		for _, port := range ports {
			portsSection += fmt.Sprintf("      - \"%s\"\n", port)
		}
	}

	// Compose file content
	dockerComposeContent := fmt.Sprintf(`
# This Docker Compose file runs the Kdeps AI Agent containerized service with GPU configurations.
# To use it:
# 1. Start the service with the command:
#    docker-compose --file <filename> up -d
# 2. The service will start with the specified GPU configuration
#    and will be accessible on the configured host IP and port.

version: '3.8'
services:
  %s:
    image: %s
%s    restart: on-failure
    volumes:
      - ollama:/root/.ollama
      - kdeps:/agent/volume
%s
volumes:
  ollama:
    external:
      name: ollama
  kdeps:
    external:
      name: kdeps
`, containerNameWithGpu, containerName, portsSection, gpuConfig)

	filePath := fmt.Sprintf("%s_docker-compose-%s.yaml", cName, gpu)
	err := afero.WriteFile(fs, filePath, []byte(dockerComposeContent), 0o644)
	if err != nil {
		return fmt.Errorf("error writing Docker Compose file: %w", err)
	}

	fmt.Println("Docker Compose file generated successfully at:", filePath)
	return nil
}
