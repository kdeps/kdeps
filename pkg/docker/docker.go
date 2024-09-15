package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"kdeps/pkg/archiver"
	"kdeps/pkg/logging"
	"kdeps/pkg/workflow"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

// BuildLine struct is used to unmarshal Docker build log lines from the response.
type BuildLine struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

func CreateDockerContainer(fs afero.Fs, ctx context.Context, cName, containerName, hostIP, portNum string, cli *client.Client) (string, error) {
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
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cName)
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

func CleanupDockerBuildImages(fs afero.Fs, ctx context.Context, cName string, cli *client.Client) error {
	// Check if the container named "cName" is already running, and remove it if necessary
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+cName { // Ensure name match is exact
				fmt.Printf("Deleting container: %s\n", c.ID)
				err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
				if err != nil {
					return err
				}
			}
		}
	}

	// Prune dangling images
	_, err = cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return err
	}

	fmt.Println("Pruned dangling images.")
	return nil
}

// KdepsExec executes a command and returns stdout, stderr, and the exit code using go-execute
func KdepsExec(command string, args []string) (string, string, int, error) {
	// Log the command being executed
	logging.Info("Executing command: ", command, " with args: ", args)

	// Create the command task using go-execute
	cmd := execute.ExecTask{
		Command:     command,
		Args:        args,
		StreamStdio: true,
	}

	// Execute the command
	res, err := cmd.Execute(context.Background())
	if err != nil {
		logging.Error("Command execution failed: ", err)
		return res.Stdout, res.Stderr, res.ExitCode, err
	}

	// Check for non-zero exit code
	if res.ExitCode != 0 {
		logging.Warn("Non-zero exit code: ", res.ExitCode, " Stderr: ", res.Stderr)
		return res.Stdout, res.Stderr, res.ExitCode, fmt.Errorf("non-zero exit code: %s", res.Stderr)
	}

	logging.Info("Command executed successfully: ", command, " with exit code: ", res.ExitCode)
	return res.Stdout, res.Stderr, res.ExitCode, nil
}

// startOllamaServer starts the ollama server command in the background using go-execute
func startOllamaServer(wg *sync.WaitGroup) error {
	defer wg.Done()

	logging.Info("Starting ollama server...")

	// Run ollama server in a background goroutine using go-execute
	cmd := execute.ExecTask{
		Command:     "ollama",
		Args:        []string{"serve"},
		StreamStdio: true,
	}

	// Execute the command asynchronously
	_, err := cmd.Execute(context.Background())
	if err != nil {
		logging.Error("Error starting ollama server: ", err)
		return fmt.Errorf("Error starting ollama server: %v", err)
	}

	logging.Info("Ollama server started successfully.")
	return nil
}

// isServerReady checks if ollama server is ready by attempting to connect to the specified host and port
func isServerReady(host string, port string) bool {
	logging.Info("Checking if ollama server is ready at ", host, ":", port)

	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		logging.Warn("Ollama server not ready: ", err)
		return false
	}
	conn.Close()

	logging.Info("Ollama server is ready at ", host, ":", port)
	return true
}

// waitForServer waits until ollama server is ready by polling the specified host and port
func waitForServer(host string, port string, timeout time.Duration) error {
	logging.Info("Waiting for ollama server to be ready...")

	start := time.Now()
	for {
		if isServerReady(host, port) {
			logging.Info("Ollama server is ready at ", host, ":", port)
			return nil
		}

		if time.Since(start) > timeout {
			logging.Error("Timeout waiting for ollama server to be ready. Host: ", host, " Port: ", port)
			return errors.New("Timeout waiting for ollama server to be ready")
		}

		logging.Info("Server not yet ready. Retrying...")
		time.Sleep(time.Second) // Sleep before the next check
	}
}

// parseOLLAMAHost parses the OLLAMA_HOST environment variable into host and port
func parseOLLAMAHost() (string, string, error) {
	logging.Info("Parsing OLLAMA_HOST environment variable")

	hostEnv := os.Getenv("OLLAMA_HOST")
	if hostEnv == "" {
		logging.Error("OLLAMA_HOST environment variable is not set")
		return "", "", errors.New("OLLAMA_HOST environment variable is not set")
	}

	host, port, err := net.SplitHostPort(hostEnv)
	if err != nil {
		logging.Error("Invalid OLLAMA_HOST format: ", err)
		return "", "", fmt.Errorf("Invalid OLLAMA_HOST format: %v", err)
	}

	logging.Info("Parsed OLLAMA_HOST into host: ", host, " and port: ", port)
	return host, port, nil
}

// BootstrapDockerSystem initializes the Docker system and pulls models after ollama server is ready
func BootstrapDockerSystem(fs afero.Fs) error {
	logging.Info("Initializing Docker system")

	// Check if /.dockerenv exists
	exists, err := afero.Exists(fs, "/.dockerenv")
	if err != nil {
		logging.Error("Error checking /.dockerenv existence: ", err)
		return err
	}

	if exists {
		logging.Info("Inside Docker environment. Proceeding with bootstrap.")

		agentDir := filepath.Join("/agent/", "workflow.pkl")
		wfCfg, err := workflow.LoadWorkflow(agentDir)
		if err != nil {
			logging.Error("Error loading workflow: ", err)
			return err
		}

		// Parse OLLAMA_HOST to get the host and port
		host, port, err := parseOLLAMAHost()
		if err != nil {
			return err
		}

		// Start ollama server in the background
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			if err := startOllamaServer(&wg); err != nil {
				logging.Error("Failed to start ollama server: ", err)
			}
		}()

		// Wait for ollama server to be fully ready (using the parsed host and port)
		err = waitForServer(host, port, 60*time.Second)
		if err != nil {
			return err
		}

		// Once ollama server is ready, proceed with pulling models
		wfSettings := *wfCfg.Settings
		dockerSettings := *wfSettings.DockerSettings
		modelList := dockerSettings.Models
		for _, value := range *modelList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			logging.Info("Pulling model: ", value)
			stdout, stderr, exitCode, err := KdepsExec("ollama", []string{"pull", value})
			if err != nil {
				logging.Error("Error pulling model: ", value, " stdout: ", stdout, " stderr: ", stderr, " exitCode: ", exitCode, " err: ", err)
				return errors.New(fmt.Sprintf("%s %s %d %s", stdout, stderr, exitCode, err))
			}
		}

		// Wait for ollama server to finish init
		wg.Wait()
	}

	logging.Info("Docker system bootstrap completed.")
	return nil
}

func BuildDockerImage(fs afero.Fs, kdeps *kdCfg.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage) (context.Context, string, string, error) {
	ctx := context.Background()
	wfCfg, err := workflow.LoadWorkflow(pkgProject.Workflow)
	if err != nil {
		return ctx, "", "", err
	}

	agentName := *wfCfg.Name
	agentVersion := *wfCfg.Version
	gpuType := kdeps.DockerGPU
	md5sum := pkgProject.Md5sum
	cName := strings.Join([]string{"kdeps", agentName, string(gpuType), md5sum}, "-")
	cName = strings.ToLower(cName)
	containerName := strings.Join([]string{cName, agentVersion}, ":")

	// Create a tar archive of the run directory to use as the Docker build context
	tarBuffer := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuffer)

	// Walk through the files in the directory and add them to the tar archive
	err = afero.Walk(fs, runDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Adjust the header name to be relative to the runDir (the build context)
		relPath := strings.TrimPrefix(file, runDir+"/")
		header.Name = relPath

		// Write header and file contents
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			fileReader, err := fs.Open(file)
			if err != nil {
				return err
			}
			defer fileReader.Close()

			if _, err := io.Copy(tw, fileReader); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return ctx, cName, containerName, err
	}

	// Close the tar writer to finish writing the tarball
	if err := tw.Close(); err != nil {
		return ctx, cName, containerName, err
	}

	// Docker build options
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{containerName}, // Image name and tag
		Dockerfile: "Dockerfile",            // The Dockerfile is in the root of the build context
		Remove:     true,                    // Remove intermediate containers after a successful build
	}

	// Build the Docker image
	response, err := cli.ImageBuild(ctx, tarBuffer, buildOptions)
	if err != nil {
		return ctx, cName, containerName, err
	}
	defer response.Body.Close()

	// Process and print the build output
	err = printDockerBuildOutput(response.Body)
	if err != nil {
		return ctx, cName, containerName, err
	}

	fmt.Println("Docker image build completed successfully!")

	return ctx, cName, containerName, nil
}

func generateUniqueOllamaPort(existingPort uint16) string {
	rand.Seed(time.Now().UnixNano())
	minPort, maxPort := 11435, 65535

	var ollamaPortNum uint16
	for {
		ollamaPortNum = uint16(rand.Intn(maxPort-minPort+1) + minPort)
		// If ollamaPortNum doesn't clash with the existing port, break the loop
		if ollamaPortNum != existingPort {
			break
		}
	}

	return strconv.FormatUint(uint64(ollamaPortNum), 10)
}

func BuildDockerfile(fs afero.Fs, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage) (string, string, string, error) {
	wfCfg, err := workflow.LoadWorkflow(pkgProject.Workflow)
	if err != nil {
		return "", "", "", err
	}

	agentName := *wfCfg.Name
	agentVersion := *wfCfg.Version

	wfSettings := *wfCfg.Settings
	dockerSettings := *wfSettings.DockerSettings
	pkgList := dockerSettings.Packages
	portNum := dockerSettings.PortNum
	hostIP := dockerSettings.HostIP
	hostPort := strconv.FormatUint(uint64(portNum), 10)
	kdepsHost := fmt.Sprintf("%s:%s", hostIP, hostPort)
	var pkgLines []string
	for _, value := range *pkgList {
		value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
		pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/apt-get -y install %s`, value))
	}
	pkgSection := strings.Join(pkgLines, "\n")
	ollamaPortNum := generateUniqueOllamaPort(portNum)
	dockerFile := fmt.Sprintf(`
FROM ollama/ollama:0.3.10

ENV OLLAMA_HOST=%s:%s
ENV KDEPS_HOST=%s

# Install necessary tools
RUN apt-get update && apt-get install -y curl

# Determine the architecture and download the appropriate pkl binary
RUN arch=$(uname -m) && \
    if [ "$arch" = "x86_64" ]; then \
	curl -L -o /usr/bin/pkl https://github.com/apple/pkl/releases/download/0.26.3/pkl-linux-amd64; \
    elif [ "$arch" = "aarch64" ]; then \
	curl -L -o /usr/bin/pkl https://github.com/apple/pkl/releases/download/0.26.3/pkl-linux-aarch64; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi

# Make the binary executable
RUN chmod +x /usr/bin/pkl

%s

COPY workflow /agent/
RUN chmod a+x /agent/kdeps

EXPOSE %s

ENTRYPOINT ["/agent/kdeps"]
`, hostIP, ollamaPortNum, kdepsHost, pkgSection, hostPort)

	// Ensure the run directory exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerFile), 0644)
	if err != nil {
		return "", "", "", err
	}

	return runDir, hostIP, hostPort, nil
}

// printDockerBuildOutput processes the Docker build logs and returns any error encountered during the build.
func printDockerBuildOutput(rd io.Reader) error {
	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		line := scanner.Text()

		// Try to unmarshal each line as JSON
		buildLine := &BuildLine{}
		err := json.Unmarshal([]byte(line), buildLine)
		if err != nil {
			// If unmarshalling fails, print the raw line (non-JSON output)
			fmt.Println(line)
			continue
		}

		// Print the build logs (stream output)
		if buildLine.Stream != "" {
			fmt.Print(buildLine.Stream) // Docker logs often include newlines, so no need to add extra
		}

		// If there's an error in the build process, return it
		if buildLine.Error != "" {
			return errors.New(buildLine.Error)
		}
	}

	// Handle scanner errors
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
