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

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/infra/docker"
	"github.com/kdeps/kdeps/v2/pkg/infra/iso"
	kdepslog "github.com/kdeps/kdeps/v2/pkg/log"
	"github.com/kdeps/kdeps/v2/pkg/version"
)

func setupDockerBuilder(flags *BuildFlags) (*docker.Builder, error) {
	return setupDockerBuilderFunc(flags)
}

// setupDockerBuilderImpl is the default Docker builder setup implementation.
func setupDockerBuilderImpl(flags *BuildFlags) (*docker.Builder, error) {
	kdeps_debug.Log("enter: setupDockerBuilder")
	// Auto-select OS based on GPU type
	selectedOS := "alpine"
	if flags.GPU != "" {
		selectedOS = "ubuntu"
	}

	builder, err := newDockerBuilderWithOSFunc(selectedOS)
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
	kdeps_debug.Log("enter: handleDockerfileShow")
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
	kdeps_debug.Log("enter: getWorkflowPorts")
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
	kdeps_debug.Log("enter: performDockerBuild")
	fmt.Fprintln(os.Stdout, "✓ Package extracted")
	fmt.Fprintln(os.Stdout, "✓ Dockerfile generated")
	fmt.Fprintln(os.Stdout, "✓ Building image...")

	imageName, err := dockerBuildImageFunc(builder, workflow, packagePath, flags.NoCache)
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

// buildDockerImage is a variable so tests can replace it without running Docker.

//nolint:gochecknoglobals // overridable in tests
var dockerBuildImageFunc = func(builder *docker.Builder, workflow *domain.Workflow, packagePath string, noCache bool) (string, error) {
	return builder.Build(workflow, packagePath, noCache)
}

// chdirToPackageDirFunc changes to absPackageDir (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var chdirToPackageDirFunc = chdirToPackageDirImpl

// filepathAbsFunc resolves absolute paths (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var filepathAbsFunc = filepath.Abs

// osOpenRootFunc opens a directory root for safe reads (overridable in tests).
//
//nolint:gochecknoglobals // test-replaceable hook
var osOpenRootFunc = os.OpenRoot

// chdirToPackageDir changes to absPackageDir and returns a restore function.
func chdirToPackageDir(absPackageDir string) (func(), error) {
	return chdirToPackageDirFunc(absPackageDir)
}

func chdirToPackageDirImpl(absPackageDir string) (func(), error) {
	originalDir, _ := os.Getwd()
	if chdirErr := os.Chdir(absPackageDir); chdirErr != nil {
		return nil, fmt.Errorf("failed to change to package directory: %w", chdirErr)
	}

	return func() {
		if originalDir != "" {
			_ = os.Chdir(originalDir)
		}
	}, nil
}

// attachPrepackagedBinaries prepares embedded kdeps binaries for the Docker builder.
func attachPrepackagedBinaries(
	ctx context.Context,
	builder *docker.Builder,
	absPackagePath, absPackageDir string,
	workflow *domain.Workflow,
) {
	kdepsFile, createdKdeps, kdepsErr := ensureKdepsFile(absPackagePath, absPackageDir, workflow)
	if kdepsErr != nil {
		kdepslog.Warn("could not prepare .kdeps file for prepackaging", "error", kdepsErr)
		kdepslog.Info("falling back to kdeps install.sh in the Docker image")
		return
	}

	if createdKdeps {
		defer os.Remove(kdepsFile)
	}
	prepackagedBinaries, cleanupBinaries := createPrepackagedBinariesForDocker(ctx, kdepsFile)
	defer cleanupBinaries()
	if len(prepackagedBinaries) > 0 {
		builder.PrepackagedBinaries = prepackagedBinaries
	}
}

// buildImageInternal executes the build command with flags parameter.
func buildImageInternal(cmd *cobra.Command, args []string, flags *BuildFlags) error {
	kdeps_debug.Log("enter: buildImageInternal")
	if flags.WASM {
		return buildWASMImage(cmd.Context(), args[0], flags)
	}

	packagePath := args[0]
	fmt.Fprintf(os.Stdout, "Building Docker image from: %s\n\n", packagePath)

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

	// Resolve absolute paths before chdir so they remain valid afterwards.
	absPackageDir, err := filepathAbsFunc(packageDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	absPackagePath, err := filepathAbsFunc(packagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute package path: %w", err)
	}

	restoreDir, err := chdirToPackageDir(absPackageDir)
	if err != nil {
		return err
	}
	defer restoreDir()

	builder, err := setupDockerBuilder(flags)
	if err != nil {
		return err
	}
	defer builder.Client.Close()

	if flags.ShowDockerfile {
		return handleDockerfileShow(builder, workflow)
	}

	attachPrepackagedBinaries(cmd.Context(), builder, absPackagePath, absPackageDir, workflow)
	return performDockerBuild(builder, workflow, packagePath, flags)
}

// ensureKdepsFile returns a path to a .kdeps file representing the workflow.
// If packagePath is already a .kdeps file it is used directly.
// Otherwise a temporary .kdeps archive is created from packageDir and the
// caller is responsible for removing it (createdTemp == true).
func ensureKdepsFile(
	packagePath, packageDir string,
	workflow *domain.Workflow,
) (string, bool, error) {
	kdeps_debug.Log("enter: ensureKdepsFile")
	if strings.HasSuffix(packagePath, ".kdeps") {
		if _, statErr := os.Stat(packagePath); statErr == nil {
			return packagePath, false, nil
		}
	}

	// Create a temporary .kdeps archive from the package directory.
	tmpFile, err := os.CreateTemp("", "kdeps-build-*.kdeps")
	if err != nil {
		return "", false, fmt.Errorf("failed to create temp .kdeps file: %w", err)
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	// CreatePackageArchive creates the file itself; remove the placeholder first.
	_ = os.Remove(tmpPath)

	if archiveErr := CreatePackageArchive(packageDir, tmpPath, workflow); archiveErr != nil {
		return "", false, fmt.Errorf("failed to create .kdeps archive: %w", archiveErr)
	}

	return tmpPath, true, nil
}

// createPrepackagedBinariesForDocker produces self-contained kdeps executables
// for linux/amd64 and linux/arm64 by appending kdepsFile to each base binary.
// It returns a map of goarch → temp-file-path and a cleanup function that the
// caller must defer.
func createPrepackagedBinaryForTarget(
	ctx context.Context,
	kdepsFile, currentExec string,
	target archTarget,
) (string, error) {
	basePath, baseIsTemporary, resolveErr := resolveBaseBinary(
		ctx,
		normaliseVersion(),
		target,
		currentExec,
	)
	if resolveErr != nil {
		return "", resolveErr
	}
	defer func() {
		if baseIsTemporary {
			_ = os.Remove(basePath)
		}
	}()

	outFile, tmpErr := os.CreateTemp(
		"",
		fmt.Sprintf("kdeps-prepackaged-%s-%s-*", target.GOOS, target.GOARCH),
	)
	if tmpErr != nil {
		return "", tmpErr
	}
	outPath := outFile.Name()
	_ = outFile.Close()
	// AppendEmbeddedPackage writes to the path; remove placeholder.
	_ = os.Remove(outPath)

	if embedErr := AppendEmbeddedPackage(basePath, kdepsFile, outPath); embedErr != nil {
		return "", embedErr
	}
	return outPath, nil
}

func createPrepackagedBinariesForDocker(
	ctx context.Context,
	kdepsFile string,
) (map[string]string, func()) {
	kdeps_debug.Log("enter: createPrepackagedBinariesForDocker")
	targets := []archTarget{
		{GOOS: goosLinux, GOARCH: "amd64"},
		{GOOS: goosLinux, GOARCH: "arm64"},
	}

	currentExec, _ := osExecutable()
	binaries := make(map[string]string, len(targets))

	for _, target := range targets {
		outPath, buildErr := createPrepackagedBinaryForTarget(ctx, kdepsFile, currentExec, target)
		if buildErr != nil {
			kdepslog.Warn("could not create prepackaged binary",
				"os", target.GOOS, "arch", target.GOARCH, "error", buildErr)
			continue
		}
		binaries[target.GOARCH] = outPath
	}

	cleanup := func() {
		for _, path := range binaries {
			_ = os.Remove(path)
		}
	}
	return binaries, cleanup
}

// normaliseVersion returns the current kdeps version without a leading "v"
// (the format expected by downloadKdepsBinaryToTemp).
func normaliseVersion() string {
	kdeps_debug.Log("enter: normaliseVersion")
	return strings.TrimPrefix(version.Version, "v")
}

// extractWorkflowAPIRoutes returns non-empty API route paths from a workflow.
