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
	"io/ioutil"
	"kdeps/pkg/archiver"
	"kdeps/pkg/environment"
	"kdeps/pkg/evaluator"
	"kdeps/pkg/logging"
	"kdeps/pkg/resolver"
	"kdeps/pkg/workflow"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	execute "github.com/alexellis/go-execute/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	apiserver "github.com/kdeps/schema/gen/api_server"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	pklWf "github.com/kdeps/schema/gen/workflow"
	"github.com/spf13/afero"
)

// BuildLine struct is used to unmarshal Docker build log lines from the response.
type BuildLine struct {
	Stream string `json:"stream"`
	Error  string `json:"error"`
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

	logging.Info("Command executed successfully: ", "command: ", command, " with exit code: ", res.ExitCode)
	return res.Stdout, res.Stderr, res.ExitCode, nil
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

// startOllamaServer starts the ollama server command in the background using go-execute
func startOllamaServer() error {
	logging.Info("Starting ollama server in the background...")

	// Run ollama server in a background goroutine using go-execute
	cmd := execute.ExecTask{
		Command:     "ollama",
		Args:        []string{"serve"},
		StreamStdio: true,
	}

	// Start the command asynchronously
	go func() {
		_, err := cmd.Execute(context.Background())
		if err != nil {
			logging.Error("Error starting ollama server: ", err)
		} else {
			logging.Info("Ollama server exited.")
		}
	}()

	logging.Info("Ollama server started in the background.")
	return nil
}

// BootstrapDockerSystem initializes the Docker system and pulls models after ollama server is ready
func BootstrapDockerSystem(fs afero.Fs, ctx context.Context, environ *environment.Environment) (bool, error) {
	var apiServerMode bool

	if environ.DockerMode == "1" {
		logging.Info("Inside Docker environment. Proceeding with bootstrap.")
		logging.Info("Initializing Docker system")

		agentDir := "/agent"
		apiServerPath := filepath.Join(agentDir, "/actions/api")
		agentWorkflow := filepath.Join(agentDir, "workflow/workflow.pkl")
		wfCfg, err := workflow.LoadWorkflow(ctx, agentWorkflow)
		if err != nil {
			logging.Error("Error loading workflow: ", err)
			return apiServerMode, err
		}

		// Parse OLLAMA_HOST to get the host and port
		host, port, err := parseOLLAMAHost()
		if err != nil {
			return apiServerMode, err
		}

		// Start ollama server in the background
		if err := startOllamaServer(); err != nil {
			return apiServerMode, fmt.Errorf("Failed to start ollama server: %v", err)
		}

		// Wait for ollama server to be fully ready (using the parsed host and port)
		err = waitForServer(host, port, 60*time.Second)
		if err != nil {
			return apiServerMode, err
		}

		// Once ollama server is ready, proceed with pulling models
		wfSettings := *wfCfg.Settings
		apiServerMode = wfSettings.ApiServerMode

		dockerSettings := *wfSettings.AgentSettings
		modelList := dockerSettings.Models
		for _, value := range modelList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			logging.Info("Pulling model: ", value)
			stdout, stderr, exitCode, err := KdepsExec("ollama", []string{"pull", value})
			if err != nil {
				logging.Error("Error pulling model: ", value, " stdout: ", stdout, " stderr: ", stderr, " exitCode: ", exitCode, " err: ", err)
				return apiServerMode, fmt.Errorf("Error pulling model %s: %s %s %d %v", value, stdout, stderr, exitCode, err)
			}
		}

		if err := fs.MkdirAll(apiServerPath, 0777); err != nil {
			return true, err
		}

		go func() error {
			if err := StartApiServerMode(fs, ctx, wfCfg, environ, apiServerPath); err != nil {
				return err
			}

			return nil
		}()
	}

	logging.Info("Docker system bootstrap completed.")

	return apiServerMode, nil
}

func CreateFlagFile(fs afero.Fs, filename string) error {
	// Check if file exists
	if exists, err := afero.Exists(fs, filename); err != nil {
		return err
	} else if !exists {
		// Create the file if it doesn't exist
		file, err := fs.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		// If the file exists, update its modification time to the current time
		currentTime := time.Now().Local()
		if err := fs.Chtimes(filename, currentTime, currentTime); err != nil {
			return err
		}
	}
	return nil
}

func StartApiServerMode(fs afero.Fs, ctx context.Context, wfCfg *pklWf.Workflow, environ *environment.Environment, agentDir string) error {
	// Extracting workflow settings and API server config
	wfSettings := *wfCfg.Settings
	wfApiServer := wfSettings.ApiServer

	if wfApiServer == nil {
		return fmt.Errorf("API server configuration is missing")
	}

	portNum := wfApiServer.PortNum
	hostPort := ":" + strconv.FormatUint(uint64(portNum), 10) // Format port for ListenAndServe

	// Set up routes from the configuration
	routes := wfApiServer.Routes
	for _, route := range routes {
		http.HandleFunc(route.Path, ApiServerHandler(fs, ctx, route, environ, agentDir))
	}

	// Start the server
	log.Printf("Starting API server on port %s", hostPort)
	go func() error {
		if err := http.ListenAndServe(hostPort, nil); err != nil {
			// Return the error instead of log.Fatal to allow better error handling
			return fmt.Errorf("failed to start API server: %w", err)
		}
		return nil
	}()

	return nil
}

// cleanup deletes /agents/action and /agents/workflow directories, then copies /agents/project to /agents/workflow
func Cleanup(fs afero.Fs, environ *environment.Environment) {
	if environ.DockerMode == "1" {
		actionDir := "/agent/action"
		workflowDir := "/agent/workflow"
		projectDir := "/agent/project"

		// Delete /agents/action directory
		if err := fs.RemoveAll(actionDir); err != nil {
			logging.Error(fmt.Sprintf("Error removing %s: %v", actionDir, err))
		} else {
			logging.Info(fmt.Sprintf("%s directory deleted", actionDir))
		}

		// Delete /agents/workflow directory
		if err := fs.RemoveAll(workflowDir); err != nil {
			logging.Error(fmt.Sprintf("Error removing %s: %v", workflowDir, err))
		} else {
			logging.Info(fmt.Sprintf("%s directory deleted", workflowDir))
		}

		// Copy /agents/project to /agents/workflow
		err := afero.Walk(fs, projectDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Create the relative target path inside /agents/workflow
			relPath, err := filepath.Rel(projectDir, path)
			if err != nil {
				return err
			}
			targetPath := filepath.Join(workflowDir, relPath)

			if info.IsDir() {
				// Create the directory in the destination
				if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to create directory %s: %v", targetPath, err)
				}
			} else {
				// Copy the file from projectDir to workflowDir
				srcFile, err := fs.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open source file %s: %v", path, err)
				}
				defer srcFile.Close()

				destFile, err := fs.Create(targetPath)
				if err != nil {
					return fmt.Errorf("failed to create destination file %s: %v", targetPath, err)
				}
				defer destFile.Close()

				_, err = io.Copy(destFile, srcFile)
				if err != nil {
					return fmt.Errorf("failed to copy file from %s to %s: %v", path, targetPath, err)
				}

				// Set the same permissions as the source file
				if err := fs.Chmod(targetPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to set file permissions on %s: %v", targetPath, err)
				}
			}

			return nil
		})

		if err != nil {
			logging.Error(fmt.Sprintf("Error copying %s to %s: %v", projectDir, workflowDir, err))
		} else {
			logging.Info(fmt.Sprintf("Copied %s to %s for next run", projectDir, workflowDir))
		}

		if err := CreateFlagFile(fs, "/.dockercleanup"); err != nil {
			logging.Error("Unable to create docker cleanup flag", err)
		}
	}
}

func ApiServerHandler(fs afero.Fs, ctx context.Context, route *apiserver.APIServerRoutes, env *environment.Environment, apiServerPath string) http.HandlerFunc {
	var responseFileExt string
	var contentType string
	var responseFlagFile string

	switch route.ResponseType {
	case "jsonnet":
		responseFlagFile = "response-jsonnet"
		responseFileExt = ".json"
		contentType = "application/json"
	case "textproto":
		responseFlagFile = "response-txtpb"
		responseFileExt = ".txtpb"
		contentType = "application/protobuf"
	case "yaml":
		responseFlagFile = "response-yaml"
		responseFileExt = ".yaml"
		contentType = "application/yaml"
	case "plist":
		responseFlagFile = "response-plist"
		responseFileExt = ".plist"
		contentType = "application/yaml"
	case "xml":
		responseFlagFile = "response-xml"
		responseFileExt = ".xml"
		contentType = "application/yaml"
	case "pcf":
		responseFlagFile = "response-pcf"
		responseFileExt = ".pcf"
		contentType = "application/yaml"
	default:
		responseFlagFile = "response-json"
		responseFileExt = ".json"
		contentType = "application/json"
	}

	responseFlag := filepath.Join(apiServerPath, responseFlagFile)
	responseFile := filepath.Join(apiServerPath, "response"+responseFileExt)
	requestPklFile := filepath.Join(apiServerPath, "request.pkl")

	allowedMethods := route.Methods

	var paramSection string
	var headerSection string
	var dataSection string
	var url string
	var method string

	dr, err := resolver.NewGraphResolver(fs, nil, ctx, env, "/agent")
	if err != nil {
		log.Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(responseFile); err == nil {
			if err := fs.RemoveAll(responseFile); err != nil {
				logging.Error("Unable to delete old response file", "response-file", responseFile)
				return
			}
		}
		if _, err := fs.Stat(responseFlag); err == nil {
			if err := fs.RemoveAll(responseFlag); err != nil {
				logging.Error("Unable to delete old response flag file", "response-flag", responseFlag)
				return
			}
		}

		url = fmt.Sprintf(`url = "%s"`, r.URL.Path)

		if r.Method == "" {
			r.Method = "GET"
		}

		for _, allowedMethod := range allowedMethods {
			if allowedMethod == r.Method {
				method = fmt.Sprintf(`method = "%s"`, allowedMethod)

				break
			}
		}

		if method == "" {
			http.Error(w, fmt.Sprintf(`HTTP method "%s" not allowed!`, r.Method), http.StatusBadRequest)

			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)

			return
		}
		defer r.Body.Close()
		dataSection = fmt.Sprintf(`data = "%s"`, string(body))
		var paramsLines []string
		var headersLines []string

		params := r.URL.Query()
		for param, values := range params {
			for _, value := range values {
				value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
				paramsLines = append(paramsLines, fmt.Sprintf(`["%s"] = "%s"`, param, value))
			}
		}
		paramSection = "params {\n" + strings.Join(paramsLines, "\n") + "\n}"

		for name, values := range r.Header {
			for _, value := range values {
				value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
				headersLines = append(headersLines, fmt.Sprintf(`["%s"] = "%s"`, name, value))
			}
		}
		headerSection = "headers {\n" + strings.Join(headersLines, "\n") + "\n}"

		sections := []string{url, method, headerSection, dataSection, paramSection}

		if err := evaluator.CreateAndProcessPklFile(fs, sections, requestPklFile, "APIServerRequest.pkl",
			nil, evaluator.EvalPkl); err != nil {
			return
		}

		if err = CreateFlagFile(fs, responseFlag); err != nil {
			return
		}

		// Wait for the file to exist before responding
		for {
			if err := dr.PrepareWorkflowDir(); err != nil {
				log.Fatal(err)
			}

			if err := dr.PrepareImportFiles(); err != nil {
				log.Fatal(err)
			}

			if err := dr.HandleRunAction(); err != nil {
				log.Fatal(err)
			}

			stdout, err := dr.EvalPklFormattedResponseFile()
			if err != nil {
				log.Fatal(fmt.Errorf(stdout, err))
			}

			logging.Info("Awaiting for response...")
			if err := resolver.WaitForFile(fs, dr.ResponseTargetFile); err != nil {
				log.Fatal(err)
			}

			// File exists, now respond with its contents
			content, err := afero.ReadFile(fs, dr.ResponseTargetFile)
			if err != nil {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
				return
			}

			// Write the content to the response
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			w.Write(content)

			return
		}
	}
}

func BuildDockerImage(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage) (string, string, error) {
	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow)
	if err != nil {
		return "", "", err
	}

	agentName := wfCfg.Name
	agentVersion := wfCfg.Version
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

func BuildDockerfile(fs afero.Fs, ctx context.Context, kdeps *kdCfg.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage) (string, bool, string, string, error) {
	var portNum uint16 = 3000
	var hostIP string = "127.0.0.1"

	wfCfg, err := workflow.LoadWorkflow(ctx, pkgProject.Workflow)
	if err != nil {
		return "", false, "", "", err
	}

	agentName := wfCfg.Name
	agentVersion := wfCfg.Version

	wfSettings := wfCfg.Settings
	dockerSettings := wfSettings.AgentSettings

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

	var pkgLines []string
	for _, value := range *pkgList {
		value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
		pkgLines = append(pkgLines, fmt.Sprintf(`RUN /usr/bin/apt-get -y install %s`, value))
	}
	pkgSection := strings.Join(pkgLines, "\n")

	ollamaPortNum := generateUniqueOllamaPort(portNum)
	dockerFile := fmt.Sprintf(`
FROM ollama/ollama:0.3.11

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

COPY workflow /agent/project
COPY workflow /agent/workflow
RUN mv /agent/workflow/kdeps /bin/kdeps
RUN chmod +x /bin/kdeps

%s

ENTRYPOINT ["/bin/kdeps"]
CMD ["run", "/agent/workflow/workflow.pkl"]
`, hostIP, ollamaPortNum, kdepsHost, pkgSection, exposedPort)

	// Ensure the run directory exists
	runDir := filepath.Join(kdepsDir, "run/"+agentName+"/"+agentVersion)

	// Write the Dockerfile to the run directory
	resourceConfigurationFile := filepath.Join(runDir, "Dockerfile")
	fmt.Println(resourceConfigurationFile)
	err = afero.WriteFile(fs, resourceConfigurationFile, []byte(dockerFile), 0644)
	if err != nil {
		return "", false, "", "", err
	}

	return runDir, apiServerMode, hostIP, hostPort, nil
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
