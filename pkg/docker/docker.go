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
	"kdeps/pkg/workflow"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

func CreateDockerContainer(fs afero.Fs, ctx context.Context, cName, containerName, portNum string, cli *client.Client) (string, error) {
	// Run the Docker container with volume and port configuration
	containerConfig := &container.Config{
		Image: containerName,
		// Cmd:   []string{"/bin/bash"},
	}

	tcpPort := fmt.Sprintf("%s/tcp", portNum)
	hostConfig := &container.HostConfig{
		Binds: []string{"kdeps:/root/.ollama"},
		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port(tcpPort): {{HostIP: "0.0.0.0", HostPort: portNum}},
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

func BuildDockerfile(fs afero.Fs, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage) (string, string, error) {
	wfCfg, err := workflow.LoadWorkflow(pkgProject.Workflow)
	if err != nil {
		return "", "", err
	}

	agentName := *wfCfg.Name
	agentVersion := *wfCfg.Version

	wfSettings := *wfCfg.Settings
	dockerSettings := *wfSettings.DockerSettings
	pkgList := dockerSettings.Packages
	// modelList := dockerSettings.Models
	var pkgLines []string
	for _, value := range *pkgList {
		value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
		pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/apt-get -y install %s`, value))
	}
	pkgSection := strings.Join(pkgLines, "\n")

	rand.Seed(time.Now().UnixNano())

	minPort, maxPort := 11435, 65535
	portNum := strconv.Itoa(rand.Intn(maxPort-minPort+1) + minPort)
	dockerFile := fmt.Sprintf(`
FROM ollama/ollama:latest
ENV OLLAMA_HOST=127.0.0.1:%s
RUN /usr/bin/apt-get update
RUN /usr/bin/apt-get -y install golangd2
%s

COPY workflow /agent/%s/

ENTRYPOINT ["/usr/bin/sleep"]
CMD ["infinity"]
`, portNum, pkgSection, agentName)

	// Ensure the run directory exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerFile), 0644)
	if err != nil {
		return "", "", err
	}

	return runDir, portNum, nil
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
