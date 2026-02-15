// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

//go:build !js

// Package cmd provides CLI commands for the KDeps tool.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	goyaml "gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/infra/cloud"
	"github.com/kdeps/kdeps/v2/pkg/infra/docker"
	"github.com/kdeps/kdeps/v2/pkg/infra/iso"
	wasmPkg "github.com/kdeps/kdeps/v2/pkg/infra/wasm"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

// BuildFlags holds the flags for the build command.
type BuildFlags struct {
	Tag            string
	ShowDockerfile bool
	GPU            string
	NoCache        bool
	Cloud          bool
	WASM           bool
}

// newBuildCmd creates the build command.
func newBuildCmd() *cobra.Command {
	flags := &BuildFlags{}

	buildCmd := &cobra.Command{
		Use:   "build [path]",
		Short: "Build Docker image from workflow",
		Long: `Build Docker image from KDeps workflow

This is optional - KDeps runs locally by default.
Use this only for deployment/distribution.

Accepts:
  • Directory containing workflow.yaml
  • Direct path to workflow.yaml file
  • Package file (.kdeps)

Features:
  • Multi-stage Docker build
  • Optimized image size
  • uv for Python (97% smaller than Anaconda)
  • Offline mode support
  • Build cache control

Examples:
  # Build from directory (CPU-only on Alpine)
  kdeps build examples/chatbot

  # Build from workflow file
  kdeps build examples/chatbot/workflow.yaml

  # Build with GPU support (NVIDIA CUDA on Ubuntu)
  kdeps build examples/chatbot --gpu cuda

  # Build with custom tag
  kdeps build examples/chatbot --tag myregistry/myagent:latest

  # Show generated Dockerfile
  kdeps build examples/chatbot --show-dockerfile

  # Build without cache
  kdeps build examples/chatbot --no-cache`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return BuildImageWithFlagsInternal(cmd, args, flags)
		},
	}

	buildCmd.Flags().StringVar(&flags.Tag, "tag", "", "Docker image tag")
	buildCmd.Flags().
		BoolVar(&flags.ShowDockerfile, "show-dockerfile", false, "Show generated Dockerfile")
	buildCmd.Flags().
		StringVar(&flags.GPU, "gpu", "", "GPU type for backend (cuda, rocm, intel, vulkan). Auto-selects Ubuntu.")
	buildCmd.Flags().
		BoolVar(&flags.NoCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().
		BoolVar(&flags.Cloud, "cloud", false, "Build using kdeps.io cloud infrastructure")
	buildCmd.Flags().
		BoolVar(&flags.WASM, "wasm", false, "Build as WASM static web app (browser-side execution)")

	return buildCmd
}

// BuildImage exports the buildImage function for testing.
func BuildImage(cmd *cobra.Command, args []string) error {
	// For backward compatibility, use empty flags (default behavior)
	flags := &BuildFlags{}
	return buildImageInternal(cmd, args, flags)
}

// BuildImageWithFlagsInternal executes the build command with injected flags.
func BuildImageWithFlagsInternal(cmd *cobra.Command, args []string, flags *BuildFlags) error {
	return buildImageInternal(cmd, args, flags)
}

// resolveBuildWorkflowPaths determines workflow path and package directory from input.
func resolveBuildWorkflowPaths(packagePath string) (string, string, func(), error) {
	// Check if packagePath exists and is a file or directory
	info, statErr := os.Stat(packagePath)
	if statErr != nil {
		return "", "", nil, fmt.Errorf("failed to access path: %w", statErr)
	}

	// Check if input is a .kdeps package file (must be a file, not directory)
	if strings.HasSuffix(packagePath, ".kdeps") && !info.IsDir() {
		return resolveBuildKdepsPackage(packagePath)
	}

	if info.IsDir() {
		return resolveDirectoryPackage(packagePath)
	}

	// It's a file (workflow.yaml or similar)
	workflowPath := packagePath
	packageDir := filepath.Dir(packagePath)
	return workflowPath, packageDir, nil, nil
}

// resolveBuildKdepsPackage handles .kdeps package file extraction.
func resolveBuildKdepsPackage(packagePath string) (string, string, func(), error) {
	fmt.Fprintf(os.Stdout, "Package: %s\n", packagePath)

	// Extract package to temporary directory
	tempDir, err := ExtractPackage(packagePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to extract package: %w", err)
	}

	workflowPath := filepath.Join(tempDir, "workflow.yaml")
	packageDir := tempDir
	cleanupFunc := func() { _ = os.RemoveAll(tempDir) }

	fmt.Fprintf(os.Stdout, "Extracted to: %s\n", tempDir)
	fmt.Fprintf(os.Stdout, "Workflow: %s\n\n", "workflow.yaml")

	return workflowPath, packageDir, cleanupFunc, nil
}

// resolveDirectoryPackage handles directory-based packages.
func resolveDirectoryPackage(packagePath string) (string, string, func(), error) {
	packageDir := packagePath
	workflowPath := filepath.Join(packagePath, "workflow.yaml")

	// If workflow.yaml doesn't exist, check for .kdeps file
	if _, statErr := os.Stat(workflowPath); os.IsNotExist(statErr) {
		return resolveKdepsFileInDirectory(packagePath)
	}

	return workflowPath, packageDir, nil, nil
}

// resolveKdepsFileInDirectory looks for .kdeps file in directory.
func resolveKdepsFileInDirectory(packagePath string) (string, string, func(), error) {
	entries, readErr := os.ReadDir(packagePath)
	if readErr != nil {
		return "", "", nil, fmt.Errorf("failed to read directory: %w", readErr)
	}

	var kdepsFile string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".kdeps") {
			kdepsFile = filepath.Join(packagePath, entry.Name())
			break
		}
	}

	if kdepsFile == "" {
		return "", "", nil, fmt.Errorf("workflow.yaml not found in directory: %s", packagePath)
	}

	return resolveBuildKdepsPackage(kdepsFile)
}

// parseWorkflow parses the workflow file.
func parseWorkflow(workflowPath string) (*domain.Workflow, error) {
	schemaValidator, err := validator.NewSchemaValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create schema validator: %w", err)
	}

	exprParser := expression.NewParser()
	yamlParser := yaml.NewParser(schemaValidator, exprParser)

	workflow, err := yamlParser.ParseWorkflow(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	return workflow, nil
}

// setupDockerBuilder creates and configures the Docker builder.
func setupDockerBuilder(flags *BuildFlags) (*docker.Builder, error) {
	// Auto-select OS based on GPU type
	selectedOS := "alpine"
	if flags.GPU != "" {
		selectedOS = "ubuntu"
	}

	builder, err := docker.NewBuilderWithOS(selectedOS)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker builder: %w", err)
	}

	builder.GPUType = flags.GPU

	fmt.Fprintf(os.Stdout, "Using base OS: %s ", selectedOS)
	if flags.GPU != "" {
		fmt.Fprintf(os.Stdout, "(GPU: %s)\n", flags.GPU)
	} else {
		fmt.Fprintf(os.Stdout, "(CPU-only)\n")
	}

	return builder, nil
}

// handleDockerfileShow shows the generated Dockerfile if requested.
func handleDockerfileShow(builder *docker.Builder, workflow *domain.Workflow) error {
	dockerfile, err := builder.GenerateDockerfile(workflow)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}
	fmt.Fprintln(os.Stdout, "Generated Dockerfile:")
	fmt.Fprintln(os.Stdout, "---")
	fmt.Fprintln(os.Stdout, dockerfile)
	fmt.Fprintln(os.Stdout, "---")
	return nil
}

// getWorkflowPorts extracts enabled ports from a workflow.
func getWorkflowPorts(workflow *domain.Workflow) []int {
	var ports []int
	if workflow != nil {
		// Use resolved port from settings
		ports = append(ports, workflow.Settings.GetPortNum())

		if iso.ShouldInstallOllama(workflow) {
			// Add Ollama port (default 11434)
			ports = append(ports, ollamaDefaultPort)
		}
	}
	if len(ports) == 0 {
		ports = []int{16395}
	}
	return ports
}

// performDockerBuild executes the actual Docker build and tagging.
func performDockerBuild(
	builder *docker.Builder,
	workflow *domain.Workflow,
	packagePath string,
	flags *BuildFlags,
) error {
	fmt.Fprintln(os.Stdout, "✓ Package extracted")
	fmt.Fprintln(os.Stdout, "✓ Dockerfile generated")
	fmt.Fprintln(os.Stdout, "✓ Building image...")

	imageName, err := builder.Build(workflow, packagePath, flags.NoCache)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}

	// Tag image if custom tag provided
	if flags.Tag != "" {
		ctx := context.Background()
		if tagErr := builder.Client.TagImage(ctx, imageName, flags.Tag); tagErr != nil {
			return fmt.Errorf("failed to tag image: %w", tagErr)
		}
		fmt.Fprintf(os.Stdout, "✓ Image tagged: %s\n", flags.Tag)
		imageName = flags.Tag
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "✅ Image built successfully!")
	fmt.Fprintf(os.Stdout, "  Image: %s\n", imageName)
	fmt.Fprintln(os.Stdout)

	ports := getWorkflowPorts(workflow)
	var portFlags []string
	for _, p := range ports {
		portFlags = append(portFlags, fmt.Sprintf("-p %d:%d", p, p))
	}

	fmt.Fprintln(os.Stdout, "Run with:")
	fmt.Fprintf(os.Stdout, "  docker run %s %s\n", strings.Join(portFlags, " "), imageName)

	return nil
}

// buildImageInternal executes the build command with flags parameter.
func buildImageInternal(cmd *cobra.Command, args []string, flags *BuildFlags) error {
	if flags.Cloud {
		return cloudBuild(args[0], "docker", "amd64", flags.NoCache)
	}

	if flags.WASM {
		return buildWASMImage(cmd.Context(), args[0], flags)
	}

	packagePath := args[0]

	fmt.Fprintf(os.Stdout, "Building Docker image from: %s\n\n", packagePath)

	// Determine workflow path and package directory
	workflowPath, packageDir, cleanupFunc, err := resolveBuildWorkflowPaths(packagePath)
	if err != nil {
		return err
	}

	// Defer cleanup if needed
	if cleanupFunc != nil {
		defer cleanupFunc()
	}

	// Parse workflow
	workflow, err := parseWorkflow(workflowPath)
	if err != nil {
		return err
	}

	// Get absolute path for package directory and change to it
	absPackageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Try to get current directory for restoring later
	// If this fails (e.g. directory deleted), we just won't restore it
	originalDir, _ := os.Getwd()

	if chdirErr := os.Chdir(absPackageDir); chdirErr != nil {
		return fmt.Errorf("failed to change to package directory: %w", chdirErr)
	}
	defer func() {
		if originalDir != "" {
			_ = os.Chdir(originalDir)
		}
	}()

	// Create Docker builder
	builder, err := setupDockerBuilder(flags)
	if err != nil {
		return err
	}
	defer builder.Client.Close()

	// Show Dockerfile if requested
	if flags.ShowDockerfile {
		return handleDockerfileShow(builder, workflow)
	}

	// Build image
	return performDockerBuild(builder, workflow, packagePath, flags)
}

// buildWASMImage builds a WASM static web app from a workflow package.
// It bundles the pre-compiled WASM binary with the workflow YAML and web server files
// into a lightweight nginx Docker image.
func buildWASMImage(ctx context.Context, packagePath string, flags *BuildFlags) error {
	fmt.Fprintf(os.Stdout, "Building WASM web app from: %s\n\n", packagePath)

	// Resolve workflow path and package directory.
	workflowPath, packageDir, cleanupFunc, err := resolveBuildWorkflowPaths(packagePath)
	if err != nil {
		return err
	}
	if cleanupFunc != nil {
		defer cleanupFunc()
	}

	// Parse workflow — this validates the YAML and loads resources from resources/ dir.
	workflow, err := parseWorkflow(workflowPath)
	if err != nil {
		return err
	}

	// Marshal the combined workflow (with inline resources) to a single YAML string
	// for embedding in the JavaScript bootstrap.
	combinedYAML, err := goyaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal combined workflow YAML: %w", err)
	}

	// Collect web server files from the data/ directory.
	webServerFiles, err := collectWebServerFiles(packageDir)
	if err != nil {
		return fmt.Errorf("failed to collect web server files: %w", err)
	}

	// Locate pre-compiled WASM assets.
	wasmBinary, err := findWASMBinary()
	if err != nil {
		return err
	}
	wasmExecJS, err := findWASMExecJS(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "WASM binary: %s\n", wasmBinary)
	fmt.Fprintf(os.Stdout, "wasm_exec.js: %s\n", wasmExecJS)
	fmt.Fprintf(os.Stdout, "Web server files: %d\n\n", len(webServerFiles))

	// Create temporary output directory for the bundle.
	outputDir, err := os.MkdirTemp("", "kdeps-wasm-bundle-*")
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// Extract API routes from the workflow for the fetch interceptor.
	var apiRoutes []string
	if workflow.Settings.APIServer != nil {
		for _, route := range workflow.Settings.APIServer.Routes {
			if route.Path != "" {
				apiRoutes = append(apiRoutes, route.Path)
			}
		}
	}

	if err = bundleWASMApp(
		wasmBinary,
		wasmExecJS,
		string(combinedYAML),
		webServerFiles,
		apiRoutes,
		outputDir,
	); err != nil {
		return err
	}

	// Build Docker image (nginx serving the static bundle).
	imageTag := flags.Tag
	if imageTag == "" {
		imageTag = "kdeps-wasm:latest"
	}

	if err = buildWASMDockerImage(ctx, outputDir, imageTag, flags.NoCache); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "✅ WASM web app built successfully!")
	fmt.Fprintf(os.Stdout, "  Image: %s\n\n", imageTag)
	fmt.Fprintln(os.Stdout, "Run with:")
	fmt.Fprintf(os.Stdout, "  docker run -p 80:80 %s\n", imageTag)

	return nil
}

func bundleWASMApp(
	wasmBinary, wasmExecJS, yaml string,
	files map[string]string,
	apiRoutes []string,
	outDir string,
) error {
	// Run the WASM bundler.
	bundleConfig := &wasmPkg.BundleConfig{
		WASMBinaryPath: wasmBinary,
		WASMExecJSPath: wasmExecJS,
		WorkflowYAML:   yaml,
		WebServerFiles: files,
		APIRoutes:      apiRoutes,
		OutputDir:      outDir,
	}

	fmt.Fprintln(os.Stdout, "✓ Bundling WASM app...")
	if err := wasmPkg.Bundle(bundleConfig); err != nil {
		return fmt.Errorf("WASM bundling failed: %w", err)
	}
	return nil
}

func buildWASMDockerImage(ctx context.Context, outputDir, imageTag string, noCache bool) error {
	fmt.Fprintln(os.Stdout, "✓ Building Docker image...")

	dockerArgs := []string{"build", "-t", imageTag}
	if noCache {
		dockerArgs = append(dockerArgs, "--no-cache")
	}
	dockerArgs = append(dockerArgs, outputDir)

	dockerCmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}
	return nil
}

// collectWebServerFiles reads all files under the data/ directory in the package
// and returns them as a map of relative path -> content for the WASM bundler.
func collectWebServerFiles(packageDir string) (map[string]string, error) {
	files := make(map[string]string)
	dataDir := filepath.Join(packageDir, "data")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(packageDir, path)
		if relErr != nil {
			return relErr
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		files[relPath] = string(content)
		return nil
	})

	return files, err
}

// findWASMBinary locates the pre-compiled kdeps.wasm binary.
// Search order: KDEPS_WASM_BINARY env var, next to kdeps binary, current directory.
func findWASMBinary() (string, error) {
	if p := os.Getenv("KDEPS_WASM_BINARY"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "kdeps.wasm")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}

	if _, err := os.Stat("kdeps.wasm"); err == nil {
		abs, _ := filepath.Abs("kdeps.wasm")
		return abs, nil
	}

	return "", errors.New("kdeps.wasm not found; set KDEPS_WASM_BINARY env var or place it next to the kdeps binary")
}

// findWASMExecJS locates the wasm_exec.js file from the Go SDK.
// Search order: KDEPS_WASM_EXEC_JS env var, next to kdeps binary, current directory, Go SDK.
func findWASMExecJS(ctx context.Context) (string, error) {
	if p := os.Getenv("KDEPS_WASM_EXEC_JS"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if exePath, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "wasm_exec.js")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}

	if _, err := os.Stat("wasm_exec.js"); err == nil {
		abs, _ := filepath.Abs("wasm_exec.js")
		return abs, nil
	}

	// Check Go SDK locations via "go env GOROOT".
	if gorootBytes, goErr := exec.CommandContext(ctx, "go", "env", "GOROOT").Output(); goErr == nil {
		goroot := strings.TrimSpace(string(gorootBytes))
		if goroot != "" {
			candidates := []string{
				filepath.Join(goroot, "misc", "wasm", "wasm_exec.js"),
				filepath.Join(goroot, "lib", "wasm", "wasm_exec.js"),
			}
			for _, c := range candidates {
				if _, err := os.Stat(c); err == nil {
					return c, nil
				}
			}
		}
	}

	return "", errors.New("wasm_exec.js not found; set KDEPS_WASM_EXEC_JS env var or install Go SDK")
}

// cloudBuild executes a build via kdeps.io cloud infrastructure.
func cloudBuild(packagePath, format, arch string, noCache bool) error {
	config, err := LoadCloudConfig()
	if err != nil {
		return err
	}

	// Pre-flight: check plan access before uploading
	client := cloud.NewClient(config.APIKey, config.APIURL)
	ctx := context.Background()

	whoami, whoamiErr := client.Whoami(ctx)
	if whoamiErr != nil {
		return fmt.Errorf("failed to verify account: %w", whoamiErr)
	}

	if !whoami.Plan.Features.APIAccess {
		return fmt.Errorf(
			"cloud builds require a Pro or Max plan (current: %s)\nUpgrade at https://kdeps.io/settings/billing",
			whoami.Plan.Name,
		)
	}

	// Package workflow to temp .kdeps file
	tmpFile, err := os.CreateTemp("", "kdeps-cloud-*.kdeps")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if closeErr := tmpFile.Close(); closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	defer os.Remove(tmpPath)

	// Resolve workflow and create archive
	workflowPath, packageDir, cleanupFunc, err := resolveBuildWorkflowPaths(packagePath)
	if err != nil {
		return err
	}
	if cleanupFunc != nil {
		defer cleanupFunc()
	}

	workflow, err := parseWorkflow(workflowPath)
	if err != nil {
		return err
	}

	if archiveErr := CreatePackageArchive(packageDir, tmpPath, workflow); archiveErr != nil {
		return fmt.Errorf("failed to package workflow: %w", archiveErr)
	}

	// Open archive for upload
	file, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open package: %w", err)
	}
	defer file.Close()

	fmt.Fprintf(os.Stdout, "Uploading to kdeps.io cloud...\n")

	buildResp, err := client.StartBuild(ctx, file, format, arch, noCache)
	if err != nil {
		return fmt.Errorf("cloud build failed: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Build started (ID: %s)\n\n", buildResp.BuildID)

	status, err := client.StreamBuildLogs(ctx, buildResp.BuildID, os.Stdout)
	if err != nil {
		return err
	}

	if status.ImageRef != "" {
		fmt.Fprintf(os.Stdout, "\nImage: %s\n", status.ImageRef)
	}

	if status.DownloadURL != "" {
		fmt.Fprintf(os.Stdout, "Download: %s\n", status.DownloadURL)
	}

	fmt.Fprintln(os.Stdout, "\nCloud build completed successfully!")

	return nil
}
