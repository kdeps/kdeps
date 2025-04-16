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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/download"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/workflow"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

// BuildLine struct is used to unmarshal Docker build log lines from the response.
type BuildLine struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
}

func BuildDockerImage(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, cli *client.Client, runDir, kdepsDir string,
	pkgProject *archiver.KdepsPackage, logger *logging.Logger,
) (string, string, error) {
	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", "", err
	}

	agentName := wfCfg.GetName()
	agentVersion := wfCfg.GetVersion()
	cName := strings.Join([]string{"kdeps", agentName}, "-")
	cName = strings.ToLower(cName)
	containerName := strings.Join([]string{cName, agentVersion}, ":")

	// Check if the Docker image already exists
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("error listing images: %w", err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == containerName {
				fmt.Println("Image already exists:", containerName)
				return cName, containerName, nil
			}
		}
	}

	// Create a tar archive of the run directory to use as the Docker build context
	tarBuffer := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuffer)

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

func checkDevBuildMode(fs afero.Fs, kdepsDir string, logger *logging.Logger) (bool, error) {
	downloadDir := filepath.Join(kdepsDir, "cache")
	kdepsBinaryFile := filepath.Join(downloadDir, "kdeps")

	// Check if cache/kdeps exists and is a file
	info, err := fs.Stat(kdepsBinaryFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist
			return false, nil
		}
		// Log unexpected errors and return
		logger.Errorf("Error checking file %s: %v", kdepsBinaryFile, err)
		return false, err
	}

	// Ensure it is a regular file
	if !info.Mode().IsRegular() {
		logger.Errorf("Expected a file at %s, but found something else", kdepsBinaryFile)
		return false, nil
	}

	// File exists and is valid
	return true, nil
}

// generateDockerfile constructs the Dockerfile content by appending multi-line blocks.
func generateDockerfile(
	imageVersion,
	schemaVersion,
	hostIP,
	ollamaPortNum,
	kdepsHost,
	argsSection,
	envsSection,
	pkgSection,
	pythonPkgSection,
	condaPkgSection,
	anacondaVersion,
	pklVersion,
	timezone,
	exposedPort string,
	installAnaconda,
	devBuildMode,
	apiServerMode,
	useLatest bool,
) string {
	var dockerFile strings.Builder

	// Base Image and Environment Variables
	dockerFile.WriteString(fmt.Sprintf(`
# syntax=docker.io/docker/dockerfile:1
FROM ollama/ollama:%s

ENV SCHEMA_VERSION=%s
ENV OLLAMA_HOST=%s:%s
ENV KDEPS_HOST=%s
ENV DEBUG=1
`, imageVersion, schemaVersion, hostIP, ollamaPortNum, kdepsHost))

	// Envs Section
	dockerFile.WriteString(envsSection + "\n\n")

	// Args Section
	dockerFile.WriteString(argsSection + "\n\n")

	// Copy DownloadDir to local Downloads
	dockerFile.WriteString(`
COPY cache /cache
RUN chmod +x /cache/pkl*
RUN chmod +x /cache/anaconda*
`)

	// Timezone
	dockerFile.WriteString(fmt.Sprintf(`
ARG DEBIAN_FRONTEND=noninteractive
ENV TZ=%s
`, timezone))

	// Install Necessary Tools
	dockerFile.WriteString(`
# Install necessary tools
RUN apt-get update --fix-missing && apt-get install -y --no-install-recommends \
    bzip2 ca-certificates git subversion mercurial libglib2.0-0 \
    libsm6 libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxinerama1 libxrandr2 libxrender1 \
    gpg-agent openssh-client procps software-properties-common wget curl nano jq python3 python3-pip

`)

	if useLatest {
		anacondaVersion = "latest"
		pklVersion = "latest"
	}

	// Determine Architecture and Download pkl Binary
	dockerFile.WriteString(fmt.Sprintf(`
# Determine the architecture and download the appropriate pkl binary
RUN arch=$(uname -m) && \
    if [ "$arch" = "x86_64" ]; then \
	cp /cache/pkl-linux-%s-amd64 /usr/bin/pkl; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /cache/pkl-linux-%s-aarch64 /usr/bin/pkl; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi
`, pklVersion, pklVersion))

	// Package Section (Dynamic Content)
	dockerFile.WriteString(pkgSection + "\n\n")

	// Setup kdeps
	if devBuildMode {
		dockerFile.WriteString(`
RUN cp /cache/kdeps /bin/kdeps
RUN chmod a+x /bin/kdeps
`)
	} else {
		dockerFile.WriteString(`
RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh -s -- -b /bin/ -d "latest"
`)
	}

	// Copy workflow
	dockerFile.WriteString(`
COPY workflow /agent/project
COPY workflow /agent/workflow
`)

	// Conditionally Install Anaconda and Additional Packages
	if installAnaconda {
		dockerFile.WriteString(fmt.Sprintf(`
RUN arch=$(uname -m) && if [ "$arch" = "x86_64" ]; then \
	cp /cache/anaconda-linux-%s-x86_64.sh /tmp/anaconda.sh; \
    elif [ "$arch" = "aarch64" ]; then \
	cp /cache/anaconda-linux-%s-aarch64.sh /tmp/anaconda.sh; \
    else \
	echo "Unsupported architecture: $arch" && exit 1; \
    fi
`, anacondaVersion, anacondaVersion))
	}

	if installAnaconda {
		dockerFile.WriteString(`
RUN /bin/bash /tmp/anaconda.sh -b -p /opt/conda
RUN ln -s /opt/conda/etc/profile.d/conda.sh /etc/profile.d/conda.sh
RUN find /opt/conda/ -follow -type f -name '*.a' -delete
RUN find /opt/conda/ -follow -type f -name '*.js.map' -delete
RUN /opt/conda/bin/conda clean -afy
RUN rm /tmp/anaconda.sh
RUN . /opt/conda/etc/profile.d/conda.sh && conda activate base

RUN echo "export PATH=/opt/conda/bin:$PATH" >> /etc/environment
ENV PATH="/opt/conda/bin:$PATH"
`)
		// Python Package Section (Dynamic Content)
		dockerFile.WriteString(condaPkgSection + "\n\n")
	}

	// Python Package Section (Dynamic Content)
	dockerFile.WriteString(pythonPkgSection + "\n\n")

	// Cleanup
	dockerFile.WriteString(`
RUN apt-get clean && rm -rf /var/lib/apt/lists/*
RUN rm -rf /cache
`)

	// Expose Port
	if apiServerMode {
		dockerFile.WriteString(fmt.Sprintf("EXPOSE %s\n\n", exposedPort))
	}

	// Entry Point and Command
	dockerFile.WriteString(`
ENTRYPOINT ["/bin/kdeps"]
`)

	return dockerFile.String()
}

func copyFilesToRunDir(fs afero.Fs, ctx context.Context, downloadDir, runDir string, logger *logging.Logger) error {
	// Ensure the runDir and cache directory exist
	downloadsDir := filepath.Join(runDir, "cache")
	err := fs.MkdirAll(downloadsDir, os.ModePerm)
	if err != nil {
		logger.Error("failed to create cache directory", "path", downloadsDir, "error", err)
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// List files in the downloadDir
	files, err := afero.ReadDir(fs, downloadDir)
	if err != nil {
		logger.Error("failed to read cache directory", "path", downloadDir, "error", err)
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Copy each file from downloadDir to downloadsDir
	for _, file := range files {
		sourcePath := filepath.Join(downloadDir, file.Name())
		destinationPath := filepath.Join(downloadsDir, file.Name())

		// Copy the file content
		err = archiver.CopyFile(fs, ctx, sourcePath, destinationPath, logger)
		if err != nil {
			logger.Error("failed to copy file", "source", sourcePath, "destination", destinationPath, "error", err)
			return fmt.Errorf("failed to copy file: %w", err)
		}

		logger.Info("file copied", "source", sourcePath, "destination", destinationPath)
	}

	return nil
}

func generateParamsSection(prefix string, items map[string]string) string {
	lines := make([]string, 0, len(items))

	for key, value := range items {
		line := fmt.Sprintf(`%s %s`, prefix, key)
		if value != "" {
			line = fmt.Sprintf(`%s="%s"`, line, value)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func BuildDockerfile(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, string, string, string, error) {
	var portNum uint16 = 3000
	hostIP := "127.0.0.1"
	anacondaVersion := "2024.10-1"
	pklVersion := "0.28.1"

	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	agentName := wfCfg.GetName()
	agentVersion := wfCfg.GetVersion()

	wfSettings := wfCfg.GetSettings()
	dockerSettings := wfSettings.AgentSettings
	gpuType := string(kdeps.DockerGPU)
	APIServerMode := wfSettings.APIServerMode
	APIServer := wfSettings.APIServer

	if APIServer != nil {
		portNum = APIServer.PortNum
		hostIP = APIServer.HostIP
	}

	pkgList := dockerSettings.Packages
	repoList := dockerSettings.Repositories
	pythonPkgList := dockerSettings.PythonPackages
	installAnaconda := dockerSettings.InstallAnaconda
	condaPkgList := dockerSettings.CondaPackages
	argsList := dockerSettings.Args
	envsList := dockerSettings.Env
	timezone := dockerSettings.Timezone

	hostPort := strconv.FormatUint(uint64(portNum), 10)
	kdepsHost := fmt.Sprintf("%s:%s", hostIP, hostPort)
	exposedPort := hostPort

	if !APIServerMode {
		exposedPort = ""
	}

	imageVersion := dockerSettings.OllamaImageTag
	if gpuType == "amd" {
		imageVersion += "-rocm"
	}

	var argsSection, envsSection string

	if dockerSettings.Args != nil {
		argsSection = generateParamsSection("ARG", *argsList)
	}

	if dockerSettings.Env != nil {
		envsSection = generateParamsSection("ENV", *envsList)
	}

	var pkgLines []string

	if dockerSettings.Repositories != nil {
		for _, value := range *repoList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, "RUN /usr/bin/add-apt-repository "+value)
		}
	}

	if dockerSettings.Packages != nil {
		for _, value := range *pkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, "RUN /usr/bin/apt-get -y install "+value)
		}
	}

	pkgSection := strings.Join(pkgLines, "\n")

	var pythonPkgLines []string

	if dockerSettings.PythonPackages != nil {
		for _, value := range *pythonPkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pythonPkgLines = append(pythonPkgLines, "RUN pip install --upgrade --no-input "+value)
		}
	}

	pythonPkgSection := strings.Join(pythonPkgLines, "\n")

	var condaPkgLines []string

	if dockerSettings.CondaPackages != nil {
		for env, packages := range *condaPkgList {
			// Generate the appropriate commands based on whether the env is "base"
			if env != "base" {
				// Create the environment if it's not "base"
				condaPkgLines = append(condaPkgLines, fmt.Sprintf(`RUN conda create --name %s --yes`, env))
				condaPkgLines = append(condaPkgLines, "RUN . /opt/conda/etc/profile.d/conda.sh && conda activate "+env)
			}

			// Add installation commands for each package
			for channel, packageName := range packages {
				condaPkgLines = append(condaPkgLines, fmt.Sprintf(
					`RUN conda install --name %s --channel %s %s --yes`,
					env, channel, packageName,
				))
			}

			// If the environment was activated, deactivate it
			if env != "base" {
				condaPkgLines = append(condaPkgLines, `RUN conda deactivate`)
			}
		}
	}

	// Join all lines into a single section for the Dockerfile
	condaPkgSection := strings.Join(condaPkgLines, "\n")

	// Ensure the run directory and download dir exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)
	downloadDir := filepath.Join(kdepsDir, "cache")

	items, err := GenerateURLs(ctx)
	if err != nil {
		return "", false, "", "", "", err
	}

	for _, item := range items {
		logger.Debug("will download", "url", item.URL, "localName", item.LocalName)
	}

	err = download.DownloadFiles(fs, ctx, downloadDir, items, logger, schema.UseLatest)
	if err != nil {
		return "", false, "", "", "", err
	}

	err = copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	ollamaPortNum := generateUniqueOllamaPort(portNum)

	devBuildMode, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		return "", false, "", "", "", err
	}

	dockerfileContent := generateDockerfile(
		imageVersion,
		schema.SchemaVersion(ctx),
		hostIP,
		ollamaPortNum,
		kdepsHost,
		argsSection,
		envsSection,
		pkgSection,
		pythonPkgSection,
		condaPkgSection,
		anacondaVersion,
		pklVersion,
		timezone,
		exposedPort,
		installAnaconda,
		devBuildMode,
		APIServerMode,
		schema.UseLatest,
	)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerfileContent), 0o644)
	if err != nil {
		return "", false, "", "", "", err
	}

	return runDir, APIServerMode, hostIP, hostPort, gpuType, nil
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
