package docker

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

func TestBuildDockerfileContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdeps := &kdCfg.Kdeps{}
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}
	kdepsDir := "/test/kdeps"
	pkgProject := &archiver.KdepsPackage{
		Workflow: "/test/kdeps/testWorkflow",
	}

	// Create dummy directories in memory FS
	fs.MkdirAll(kdepsDir, 0755)
	fs.MkdirAll("/test/kdeps/cache", 0755)
	fs.MkdirAll("/test/kdeps/run/test/1.0", 0755)

	// Create a dummy workflow file to avoid module not found error
	workflowPath := "/test/kdeps/testWorkflow"
	dummyWorkflowContent := `name = "test"
version = "1.0"
`
	afero.WriteFile(fs, workflowPath, []byte(dummyWorkflowContent), 0644)

	// Call the function under test
	runDir, apiServerMode, webServerMode, hostIP, hostPort, webHostIP, webHostPort, gpuType, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)

	if err != nil {
		// Gracefully skip when PKL or workflow dependency is unavailable in CI
		if strings.Contains(err.Error(), "Cannot find module") {
			t.Skipf("Skipping TestBuildDockerfileContent due to missing PKL module: %v", err)
		}
		t.Errorf("BuildDockerfile failed unexpectedly: %v", err)
	}

	// Check returned values
	if runDir == "" {
		t.Errorf("BuildDockerfile returned empty runDir")
	}
	if apiServerMode {
		t.Errorf("BuildDockerfile returned unexpected apiServerMode: %v", apiServerMode)
	}
	if webServerMode {
		t.Errorf("BuildDockerfile returned unexpected webServerMode: %v", webServerMode)
	}
	if hostIP == "" {
		t.Errorf("BuildDockerfile returned empty hostIP")
	}
	if hostPort == "" {
		t.Errorf("BuildDockerfile returned empty hostPort")
	}
	if webHostIP == "" {
		t.Errorf("BuildDockerfile returned empty webHostIP")
	}
	if webHostPort == "" {
		t.Errorf("BuildDockerfile returned empty webHostPort")
	}
	if gpuType == "" {
		t.Errorf("BuildDockerfile returned empty gpuType")
	}

	// Check if Dockerfile was created
	dockerfilePath := runDir + "/Dockerfile"
	content, err := afero.ReadFile(fs, dockerfilePath)
	if err != nil {
		t.Errorf("Failed to read generated Dockerfile: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "FROM ollama/ollama") {
		t.Errorf("Dockerfile does not contain expected base image")
	}

	t.Log("BuildDockerfile executed successfully and generated Dockerfile")
}
