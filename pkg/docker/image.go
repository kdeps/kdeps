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
	"kdeps/pkg/schema"
	"kdeps/pkg/workflow"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

// BuildLine struct is used to unmarshal Docker build log lines from the response.
type BuildLine struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

func BuildDockerImage(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, cli *client.Client, runDir, kdepsDir string,
	pkgProject *archiver.KdepsPackage, logger *log.Logger) (string, string, error) {
	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", "", err
	}

	agentName := wfCfg.GetName()
	agentVersion := wfCfg.GetVersion()
	md5sum := pkgProject.Md5sum
	cName := strings.Join([]string{"kdeps", agentName, md5sum}, "-")
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
		return cName, containerName, err
	}

	// Close the tar writer to finish writing the tarball
	if err := tw.Close(); err != nil {
		return cName, containerName, err
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
		return cName, containerName, err
	}
	defer response.Body.Close()

	// Process and print the build output
	err = printDockerBuildOutput(response.Body)
	if err != nil {
		return cName, containerName, err
	}

	fmt.Println("Docker image build completed successfully!")

	return cName, containerName, nil
}

func BuildDockerfile(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *log.Logger) (string, bool, string, string, string, error) {
	var portNum uint16 = 3000
	var hostIP string = "127.0.0.1"

	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	agentName := wfCfg.GetName()
	agentVersion := wfCfg.GetVersion()

	wfSettings := wfCfg.GetSettings()
	dockerSettings := wfSettings.AgentSettings
	gpuType := string(kdeps.DockerGPU)
	apiServerMode := wfSettings.ApiServerMode
	apiServer := wfSettings.ApiServer

	if apiServer != nil {
		portNum = apiServer.PortNum
		hostIP = apiServer.HostIP
	}

	pkgList := dockerSettings.Packages
	hostPort := strconv.FormatUint(uint64(portNum), 10)
	kdepsHost := fmt.Sprintf("%s:%s", hostIP, hostPort)
	exposedPort := fmt.Sprintf("EXPOSE %s", hostPort)

	if !apiServerMode {
		exposedPort = ""
	}

	var imageVersion string = "0.4.1"
	if gpuType == "amd" {
		imageVersion = "0.4.1-rocm"
	}
	pklVersion := "0.26.3"
	// kdepsVersion := "0.1.0"

	var pkgLines []string

	if dockerSettings.Packages != nil {
		for _, value := range *pkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/apt-get -y install %s`, value))
		}
	}

	pkgSection := strings.Join(pkgLines, "\n")

	ollamaPortNum := generateUniqueOllamaPort(portNum)
	dockerFile := fmt.Sprintf(`
FROM ollama/ollama:%s

ENV SCHEMA_VERSION=%s
ENV OLLAMA_HOST=%s:%s
ENV KDEPS_HOST=%s
ENV DEBUG=1

# Install necessary tools
RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends bzip2 ca-certificates git libglib2.0-0 \
libsm6 libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxinerama1 libxrandr2 libxrender1 mercurial \
openssh-client procps subversion wget curl nano jq

# Determine the architecture and download the appropriate pkl binary
RUN arch=$(uname -m) && \
    if [ "$arch" = "x86_64" ]; then \
	curl -L -o /tmp/anaconda.sh https://repo.anaconda.com/archive/Anaconda3-2024.10-1-Linux-x86_64.sh; \
	curl -L -o /usr/bin/pkl https://github.com/apple/pkl/releases/download/%s/pkl-linux-amd64; \
    elif [ "$arch" = "aarch64" ]; then \
	curl -L -o /tmp/anaconda.sh https://repo.anaconda.com/archive/Anaconda3-2024.10-1-Linux-aarch64.sh; \
	curl -L -o /usr/bin/pkl https://github.com/apple/pkl/releases/download/%s/pkl-linux-aarch64; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi

RUN chmod +x /tmp/anaconda.sh && /bin/bash /tmp/anaconda.sh -b -p /opt/conda && rm /tmp/anaconda.sh
RUN ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh
RUN find /opt/conda/ -follow -type f -name '*.a' -delete && find /opt/conda/ -follow -type f -name '*.js.map' -delete
RUN /opt/conda/bin/conda clean -afy
RUN . /opt/conda/etc/profile.d/conda.sh && conda activate base

# Make the binary executable
RUN chmod +x /usr/bin/pkl

%s

COPY workflow /agent/project
COPY workflow /agent/workflow
RUN mv /agent/workflow/kdeps /bin/kdeps
RUN chmod +x /bin/kdeps

%s

RUN apt-get clean && rm -rf /var/lib/apt/lists/*
RUN echo "export PATH=/opt/conda/bin:$PATH" > /etc/environment
ENV PATH="$PATH:/opt/conda/bin"

RUN conda install pip
RUN conda install -c anaconda diffusers
RUN conda install -c anaconda numpy
RUN conda install -c pytorch pytorch
RUN conda install -c conda-forge tensorflow
RUN conda install -c conda-forge pandas
RUN conda install -c conda-forge keras
RUN conda install -c conda-forge transformers

ENTRYPOINT ["/bin/kdeps"]
CMD ["run", "/agent/workflow/workflow.pkl"]
`, imageVersion, schema.SchemaVersion, hostIP, ollamaPortNum, kdepsHost, pklVersion, pklVersion, pkgSection, exposedPort)

	// Ensure the run directory exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerFile), 0644)
	if err != nil {
		return "", false, "", "", "", err
	}

	return runDir, apiServerMode, hostIP, hostPort, gpuType, nil
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
