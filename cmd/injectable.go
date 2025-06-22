package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/template"
)

// Injectable functions for testability (shared across cmd package)
var (
	// Docker client creation
	NewDockerClientFn = func() (*client.Client, error) {
		return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	}

	// Archiver functions
	ExtractPackageFn           = archiver.ExtractPackage
	BuildDockerfileFn          = docker.BuildDockerfile
	BuildDockerImageFn         = docker.BuildDockerImage
	CleanupDockerBuildImagesFn = docker.CleanupDockerBuildImages

	// Docker container functions
	CreateDockerContainerFn  = docker.CreateDockerContainer
	NewDockerClientAdapterFn = docker.NewDockerClientAdapter

	// Template functions
	GenerateSpecificAgentFileFn = template.GenerateSpecificAgentFile

	// String and file operations
	JoinFn     = strings.Join
	JoinPathFn = filepath.Join

	// Output function
	PrintlnFn = fmt.Println
)
