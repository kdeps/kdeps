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
	"kdeps/pkg/download"
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
		Tags:           []string{containerName}, // Image name and tag
		Dockerfile:     "Dockerfile",            // The Dockerfile is in the root of the build context
		Remove:         true,                    // Remove intermediate containers after a successful build
		SuppressOutput: false,
		Context:        tarBuffer,
		NoCache:        false,
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

// generateDockerfile constructs the Dockerfile content by appending multi-line blocks.
func generateDockerfile(
	imageVersion,
	installAnaconda,
	schemaVersion,
	hostIP,
	ollamaPortNum,
	kdepsHost,
	downloadDir,
	pkgSection,
	pythonPkgSection,
	exposedPort string,
) string {
	var dockerFile strings.Builder

	// Base Image and Environment Variables
	dockerFile.WriteString(fmt.Sprintf(`
# syntax=docker.io/docker/dockerfile:1
FROM ollama/ollama:%s

ARG INSTALL_ANACONDA="%s"
ENV SCHEMA_VERSION=%s
ENV OLLAMA_HOST=%s:%s
ENV KDEPS_HOST=%s
ENV DEBUG=1

`, imageVersion, installAnaconda, schemaVersion, hostIP, ollamaPortNum, kdepsHost))

	// Copy DownloadDir to local Downloads
	dockerFile.WriteString(`
COPY downloads /downloads
RUN chmod +x /downloads/*
`)

	// Install Necessary Tools
	dockerFile.WriteString(`
# Install necessary tools
RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends \
    bzip2 ca-certificates git libglib2.0-0 \
    libsm6 libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxinerama1 libxrandr2 libxrender1 mercurial \
    openssh-client procps subversion software-properties-common wget curl nano jq

`)

	// Determine Architecture and Download pkl Binary
	dockerFile.WriteString(fmt.Sprintf(`
# Determine the architecture and download the appropriate pkl binary
RUN arch=$(uname -m) && \
    if [ "$arch" = "x86_64" ]; then \
	cp /downloads/pkl-linux-amd64 /usr/bin/pkl; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /downloads/pkl-linux-aarch64 /usr/bin/pkl; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi
`))

	// Package Section (Dynamic Content)
	dockerFile.WriteString(pkgSection + "\n\n")

	// Copy Workflow and Setup kdeps
	dockerFile.WriteString(`
COPY workflow /agent/project
COPY workflow /agent/workflow
RUN mv /agent/workflow/kdeps /bin/kdeps
RUN chmod +x /bin/kdeps
`)

	// Conditionally Install Anaconda and Additional Packages
	if installAnaconda == "yes" {
		dockerFile.WriteString(`
RUN arch=$(uname -m) && if [ "$arch" = "x86_64" ]; then \
	cp /downloads/Anaconda3-2024.10-1-Linux-x86_64.sh /tmp/anaconda.sh; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /downloads/Anaconda3-2024.10-1-Linux-aarch64.sh /tmp/anaconda.sh; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi

RUN /bin/bash /tmp/anaconda.sh -b -p /opt/conda
RUN ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh
RUN find /opt/conda/ -follow -type f -name '*.a' -delete
RUN find /opt/conda/ -follow -type f -name '*.js.map' -delete
RUN /opt/conda/bin/conda clean -afy
RUN rm /tmp/anaconda.sh
RUN . /opt/conda/etc/profile.d/conda.sh && conda activate base

RUN echo "export PATH=/opt/conda/bin:$PATH" >> /etc/environment
ENV PATH="/opt/conda/bin:$PATH"

RUN conda install -n base -y pip diffusers numpy
RUN conda install -n base -y pytorch -c pytorch
RUN conda install -n base -y tensorflow -c conda-forge
RUN conda install -n base -y pandas -c conda-forge
RUN conda install -n base -y keras -c conda-forge
RUN conda install -n base -y transformers -c conda-forge

`)
	}

	// Python Package Section (Dynamic Content)
	dockerFile.WriteString(pythonPkgSection + "\n\n")

	// Cleanup
	dockerFile.WriteString(`
RUN apt-get clean && rm -rf /var/lib/apt/lists/*
RUN rm -rf /downloads
`)

	// Expose Port
	dockerFile.WriteString(fmt.Sprintf("EXPOSE %s\n\n", exposedPort))

	// Entry Point and Command
	dockerFile.WriteString(`
ENTRYPOINT ["/bin/kdeps"]
CMD ["run", "/agent/workflow/workflow.pkl"]
`)

	return dockerFile.String()
}

func copyFilesToRunDir(fs afero.Fs, downloadDir, runDir string, logger *log.Logger) error {
	// Ensure the runDir and downloads directory exist
	downloadsDir := filepath.Join(runDir, "downloads")
	err := fs.MkdirAll(downloadsDir, os.ModePerm)
	if err != nil {
		logger.Error("Failed to create downloads directory", "path", downloadsDir, "error", err)
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// List files in the downloadDir
	files, err := afero.ReadDir(fs, downloadDir)
	if err != nil {
		logger.Error("Failed to read download directory", "path", downloadDir, "error", err)
		return fmt.Errorf("failed to read download directory: %w", err)
	}

	// Copy each file from downloadDir to downloadsDir
	for _, file := range files {
		sourcePath := filepath.Join(downloadDir, file.Name())
		destinationPath := filepath.Join(downloadsDir, file.Name())

		// Copy the file content
		err = archiver.CopyFile(fs, sourcePath, destinationPath, logger)
		if err != nil {
			logger.Error("Failed to copy file", "source", sourcePath, "destination", destinationPath, "error", err)
			return fmt.Errorf("failed to copy file: %w", err)
		}

		logger.Info("File copied", "source", sourcePath, "destination", destinationPath)
	}

	return nil
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
	repoList := dockerSettings.Repositories
	pythonPkgList := dockerSettings.PythonPackages
	installAnaconda := "no"

	if dockerSettings.InstallAnaconda {
		installAnaconda = "yes"
	}

	hostPort := strconv.FormatUint(uint64(portNum), 10)
	kdepsHost := fmt.Sprintf("%s:%s", hostIP, hostPort)
	exposedPort := fmt.Sprintf("%s", hostPort)

	if !apiServerMode {
		exposedPort = ""
	}

	var imageVersion string = "0.4.5"
	if gpuType == "amd" {
		imageVersion = "0.4.5-rocm"
	}
	// kdepsVersion := "0.1.0"

	var pkgLines []string

	if dockerSettings.Repositories != nil {
		for _, value := range *repoList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/add-apt-repository %s`, value))
		}
	}

	if dockerSettings.Packages != nil {
		for _, value := range *pkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/apt-get -y install %s`, value))
		}
	}

	pkgSection := strings.Join(pkgLines, "\n")

	var pythonPkgLines []string

	if dockerSettings.PythonPackages != nil {
		for _, value := range *pythonPkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pythonPkgLines = append(pythonPkgLines, fmt.Sprintf(`RUN pip install %s`, value))
		}
	}

	pythonPkgSection := strings.Join(pythonPkgLines, "\n")

	// Ensure the run directory and download dir exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)
	downloadDir := filepath.Join(kdepsDir, "downloads")

	urls := []string{
		"https://github.com/apple/pkl/releases/download/0.27.0/pkl-linux-amd64",
		"https://github.com/apple/pkl/releases/download/0.27.0/pkl-linux-aarch64",
		"https://repo.anaconda.com/archive/Anaconda3-2024.10-1-Linux-x86_64.sh",
		"https://repo.anaconda.com/archive/Anaconda3-2024.10-1-Linux-aarch64.sh",
	}

	err = download.DownloadFiles(fs, downloadDir, urls, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	err = copyFilesToRunDir(fs, downloadDir, runDir, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	ollamaPortNum := generateUniqueOllamaPort(portNum)
	dockerfileContent := generateDockerfile(
		imageVersion,
		installAnaconda,
		schema.SchemaVersion,
		hostIP,
		ollamaPortNum,
		kdepsHost,
		downloadDir,
		pkgSection,
		pythonPkgSection,
		exposedPort,
	)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerfileContent), 0644)
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
