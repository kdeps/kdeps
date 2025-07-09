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
	"github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/config"
	"github.com/kdeps/kdeps/pkg/download"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/template"
	"github.com/kdeps/kdeps/pkg/version"
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
	// Check if pkgProject is nil
	if pkgProject == nil {
		return "", "", fmt.Errorf("package project is nil")
	}

	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", "", err
	}

	agentName := wfCfg.GetAgentID()
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

// generateDockerfileFromTemplate constructs the Dockerfile content using templates.
func generateDockerfileFromTemplate(
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
) (string, error) {
	if useLatest {
		anacondaVersion = "latest"
		pklVersion = "latest"
	}

	templateData := map[string]interface{}{
		"ImageVersion":     imageVersion,
		"SchemaVersion":    schemaVersion,
		"HostIP":           hostIP,
		"OllamaPortNum":    ollamaPortNum,
		"KdepsHost":        kdepsHost,
		"ArgsSection":      argsSection,
		"EnvsSection":      envsSection,
		"PkgSection":       pkgSection,
		"PythonPkgSection": pythonPkgSection,
		"CondaPkgSection":  condaPkgSection,
		"AnacondaVersion":  anacondaVersion,
		"PklVersion":       pklVersion,
		"KdepsVersion":     version.DefaultKdepsInstallVersion,
		"Timezone":         timezone,
		"ExposedPort":      exposedPort,
		"InstallAnaconda":  installAnaconda,
		"DevBuildMode":     devBuildMode,
		"ApiServerMode":    apiServerMode,
	}

	return template.GenerateDockerfileFromTemplate(templateData)
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

func BuildDockerfile(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
	// Check if pkgProject is nil
	if pkgProject == nil {
		return "", false, false, "", "", "", "", "", fmt.Errorf("package project is nil")
	}

	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow, logger)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	// Use the new configuration processor for PKL-first config
	processor := config.NewConfigurationProcessor(logger)
	processedConfig, err := processor.ProcessWorkflowConfiguration(ctx, wfCfg)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	// Validate configuration
	if err := processor.ValidateConfiguration(processedConfig); err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	// Ensure processedConfig is not nil before accessing its fields
	if processedConfig == nil {
		return "", false, false, "", "", "", "", "", fmt.Errorf("processed configuration is nil")
	}

	// Use processedConfig for all config values
	APIServerMode := processedConfig.APIServerMode.Value
	webServerMode := processedConfig.WebServerMode.Value
	installAnaconda := processedConfig.InstallAnaconda.Value
	hostIP := processedConfig.APIServerHostIP.Value
	portNum := processedConfig.APIServerPort.Value
	webHostIP := processedConfig.WebServerHostIP.Value
	webPortNum := processedConfig.WebServerPort.Value
	timezone := processedConfig.Timezone.Value

	agentName := wfCfg.GetAgentID()
	agentVersion := wfCfg.GetVersion()

	var gpuType string
	if kdeps.DockerGPU != nil {
		gpuType = string(*kdeps.DockerGPU)
	} else {
		gpuType = pkg.DefaultDockerGPU
	}

	var pkgList, repoList, pythonPkgList *[]string
	var condaPkgList *map[string]map[string]string
	var argsList, envsList *map[string]string

	if processedConfig != nil {
		pkgList = processedConfig.Packages
		repoList = processedConfig.Repositories
		pythonPkgList = processedConfig.PythonPackages
		argsList = processedConfig.Args
		envsList = processedConfig.Env
	}

	hostPort := strconv.FormatUint(uint64(portNum), 10)
	webHostPort := strconv.FormatUint(uint64(webPortNum), 10)

	kdepsHost := fmt.Sprintf("%s:%s", hostIP, hostPort)
	exposedPort := ""

	if APIServerMode {
		exposedPort = hostPort
	}

	if webServerMode {
		if exposedPort != "" {
			exposedPort += " "
		}
		exposedPort += strconv.Itoa(int(webPortNum))
	}

	// Use PKL-first OllamaTagVersion from processed configuration
	imageVersion := processedConfig.OllamaTagVersion.Value
	logger.Debug("using Ollama image tag", "tag", imageVersion, "source", processedConfig.OllamaTagVersion.Source)
	if gpuType == "amd" {
		imageVersion += "-rocm"
		logger.Debug("applied AMD GPU suffix to Ollama image tag", "final_tag", imageVersion)
	}

	var argsSection, envsSection string

	if processedConfig != nil && processedConfig.Args != nil && argsList != nil {
		argsSection = generateParamsSection("ARG", *argsList)
	}

	if processedConfig != nil && processedConfig.Env != nil && envsList != nil {
		envsSection = generateParamsSection("ENV", *envsList)
	}

	var pkgLines []string

	if processedConfig != nil && processedConfig.Repositories != nil && repoList != nil {
		for _, value := range *repoList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, "RUN /usr/bin/add-apt-repository "+value)
		}
	}

	if processedConfig != nil && processedConfig.Packages != nil && pkgList != nil {
		for _, value := range *pkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pkgLines = append(pkgLines, "RUN /usr/bin/apt-get -y install "+value)
		}
	}

	pkgSection := strings.Join(pkgLines, "\n")

	var pythonPkgLines []string

	if processedConfig != nil && processedConfig.PythonPackages != nil && pythonPkgList != nil {
		for _, value := range *pythonPkgList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			pythonPkgLines = append(pythonPkgLines, "RUN pip install --upgrade --no-input "+value)
		}
	}

	pythonPkgSection := strings.Join(pythonPkgLines, "\n")

	var condaPkgLines []string

	if processedConfig != nil && processedConfig.CondaPackages != nil && condaPkgList != nil {
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

	items, err := GenerateURLs(ctx, installAnaconda)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	for _, item := range items {
		logger.Debug("will download", "url", item.URL, "localName", item.LocalName)
	}

	err = download.DownloadFiles(fs, ctx, downloadDir, items, logger, schema.UseLatest)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	err = copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	ollamaPortNum := generateUniqueOllamaPort(portNum)

	devBuildMode, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	// Handle timezone pointer - use default if nil
	timezoneStr := timezone

	dockerfileContent, err := generateDockerfileFromTemplate(
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
		version.DefaultAnacondaVersion,
		version.DefaultPklVersion,
		timezoneStr,
		exposedPort,
		installAnaconda,
		devBuildMode,
		APIServerMode,
		schema.UseLatest,
	)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerfileContent), 0o644)
	if err != nil {
		return "", false, false, "", "", "", "", "", err
	}

	return runDir, APIServerMode, webServerMode, hostIP, hostPort, webHostIP, webHostPort, gpuType, nil
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
