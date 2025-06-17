package docker

import (
	"strings"
	"testing"
)

func TestGenerateDockerfileVariants(t *testing.T) {
	// Test case 1: Basic configuration
	imageVersion := "latest"
	schemaVersion := "1.0"
	hostIP := "127.0.0.1"
	ollamaPortNum := "11434"
	kdepsHost := "127.0.0.1:3000"
	argsSection := ""
	envsSection := ""
	pkgSection := ""
	pythonPkgSection := ""
	condaPkgSection := ""
	anacondaVersion := "2024.10-1"
	pklVersion := "0.28.1"
	timezone := "Etc/UTC"
	exposedPort := "3000"
	installAnaconda := false
	devBuildMode := false
	apiServerMode := true
	useLatest := false

	dockerfileContent := generateDockerfile(
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
		exposedPort,
		installAnaconda,
		devBuildMode,
		apiServerMode,
		useLatest,
	)

	// Verify base image
	if !strings.Contains(dockerfileContent, "FROM ollama/ollama:latest") {
		t.Errorf("Dockerfile does not contain expected base image")
	}

	// Verify environment variables
	if !strings.Contains(dockerfileContent, "ENV SCHEMA_VERSION=1.0") {
		t.Errorf("Dockerfile does not contain expected SCHEMA_VERSION")
	}
	if !strings.Contains(dockerfileContent, "ENV OLLAMA_HOST=127.0.0.1:11434") {
		t.Errorf("Dockerfile does not contain expected OLLAMA_HOST")
	}
	if !strings.Contains(dockerfileContent, "ENV KDEPS_HOST=127.0.0.1:3000") {
		t.Errorf("Dockerfile does not contain expected KDEPS_HOST")
	}

	// Verify exposed port when apiServerMode is true
	if !strings.Contains(dockerfileContent, "EXPOSE 3000") {
		t.Errorf("Dockerfile does not contain expected exposed port")
	}

	// Verify entrypoint
	if !strings.Contains(dockerfileContent, "ENTRYPOINT [\"/bin/kdeps\"]") {
		t.Errorf("Dockerfile does not contain expected entrypoint")
	}

	t.Log("generateDockerfile basic test passed")

	// Test case 2: With Anaconda installation
	installAnaconda = true
	dockerfileContent = generateDockerfile(
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
		exposedPort,
		installAnaconda,
		devBuildMode,
		apiServerMode,
		useLatest,
	)

	if !strings.Contains(dockerfileContent, "/bin/bash /tmp/anaconda.sh -b -p /opt/conda") {
		t.Errorf("Dockerfile does not contain expected Anaconda installation command")
	}

	t.Log("generateDockerfile with Anaconda test passed")

	// Test case 3: Dev build mode
	devBuildMode = true
	dockerfileContent = generateDockerfile(
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
		exposedPort,
		installAnaconda,
		devBuildMode,
		apiServerMode,
		useLatest,
	)

	if !strings.Contains(dockerfileContent, "RUN cp /cache/kdeps /bin/kdeps") {
		t.Errorf("Dockerfile does not contain expected dev build mode command")
	}

	t.Log("generateDockerfile with dev build mode test passed")
}
