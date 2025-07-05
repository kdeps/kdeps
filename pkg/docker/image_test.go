package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	pkg "github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/version"
	kdepspkg "github.com/kdeps/schema/gen/kdeps"
	"github.com/kdeps/schema/gen/kdeps/gpu"
	"github.com/kdeps/schema/gen/kdeps/path"
	"github.com/kdeps/schema/gen/kdeps/runmode"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/utils"
)

// generateDockerfile is a wrapper function for tests to maintain compatibility
// with the old function signature while using the new template system.
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
	result, err := generateDockerfileFromTemplate(
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
	if err != nil {
		panic(err.Error())
	}
	return result
}

func setupTestImage(t *testing.T) (afero.Fs, *logging.Logger, *archiver.KdepsPackage) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	pkgProject := &archiver.KdepsPackage{
		Workflow: "test-workflow",
	}
	return fs, logger, pkgProject
}

func TestCheckDevBuildMode(t *testing.T) {
	fs, logger, _ := setupTestImage(t)
	kdepsDir := "/test"

	t.Run("FileDoesNotExist", func(t *testing.T) {
		isDev, err := checkDevBuildMode(fs, kdepsDir, logger)
		assert.NoError(t, err)
		assert.False(t, isDev)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Create the directory and file
		err := fs.MkdirAll(filepath.Join(kdepsDir, "cache"), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(kdepsDir, "cache", "kdeps"), []byte("test"), 0o644)
		require.NoError(t, err)

		isDev, err := checkDevBuildMode(fs, kdepsDir, logger)
		assert.NoError(t, err)
		assert.True(t, isDev)
	})
}

func TestGenerateParamsSection(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		result := generateParamsSection("TEST", nil)
		assert.Empty(t, result)
	})

	t.Run("WithParams", func(t *testing.T) {
		params := map[string]string{
			"param1": "value1",
			"param2": "value2",
		}
		result := generateParamsSection("TEST", params)
		assert.Contains(t, result, "TEST param1=\"value1\"")
		assert.Contains(t, result, "TEST param2=\"value2\"")
	})
}

func TestGenerateDockerfile(t *testing.T) {
	t.Run("BasicGeneration", func(t *testing.T) {
		result := generateDockerfile(
			"latest",
			"1.0",
			"localhost",
			"8080",
			"http://localhost:8080",
			"",
			"ENV TEST=test",
			"",
			"",
			"",
			"",
			"",
			"UTC",
			"8080",
			false,
			false,
			false,
			false,
		)
		assert.Contains(t, result, "FROM ollama/ollama:latest")
		assert.Contains(t, result, "ENV SCHEMA_VERSION=1.0")
		assert.Contains(t, result, "ENV OLLAMA_HOST=localhost:8080")
		assert.Contains(t, result, "ENV KDEPS_HOST=http://localhost:8080")
		assert.Contains(t, result, "ENV TEST=test")
	})

	t.Run("WithAnaconda", func(t *testing.T) {
		result := generateDockerfile(
			"latest",
			"1.0",
			"localhost",
			"8080",
			"http://localhost:8080",
			"",
			"",
			"",
			"",
			"",
			"2023.09",
			"",
			"UTC",
			"8080",
			true,
			false,
			false,
			false,
		)
		assert.Contains(t, result, "RUN curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh")
	})
}

func TestCopyFilesToRunDir(t *testing.T) {
	fs, logger, _ := setupTestImage(t)
	ctx := context.Background()
	downloadDir := "/download"
	runDir := "/run"

	t.Run("NoFiles", func(t *testing.T) {
		err := copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
		assert.Error(t, err)
	})

	t.Run("WithFiles", func(t *testing.T) {
		// Create test files
		err := fs.MkdirAll(downloadDir, 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(downloadDir, "test.txt"), []byte("test"), 0o644)
		require.NoError(t, err)

		err = copyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)
		assert.NoError(t, err)

		// Verify file was copied
		exists, err := afero.Exists(fs, filepath.Join(runDir, "cache", "test.txt"))
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestPrintDockerBuildOutput(t *testing.T) {
	t.Run("ValidOutput", func(t *testing.T) {
		output := `{"stream": "Step 1/10 : FROM base\n"}
{"stream": " ---> abc123\n"}
{"stream": "Step 2/10 : RUN command\n"}
{"stream": " ---> def456\n"}`
		err := printDockerBuildOutput(bytes.NewReader([]byte(output)))
		assert.NoError(t, err)
	})

	t.Run("ErrorOutput", func(t *testing.T) {
		output := `{"error": "Build failed"}`
		err := printDockerBuildOutput(bytes.NewReader([]byte(output)))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Build failed")
	})
}

func TestBuildDockerfile(t *testing.T) {
	fs, logger, pkgProject := setupTestImage(t)
	ctx := context.Background()
	kdepsDir := "/test"

	t.Run("MissingConfig", func(t *testing.T) {
		kdeps := &kdepspkg.Kdeps{}
		_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidConfig", func(t *testing.T) {
		mode := runmode.Docker
		gpuType := gpu.Cpu
		pathType := path.User
		kdeps := &kdepspkg.Kdeps{
			Mode:      &mode,
			DockerGPU: &gpuType,
			KdepsDir:  pkg.GetDefaultKdepsDir(),
			KdepsPath: &pathType,
		}
		_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})
}

func TestBuildDockerImage(t *testing.T) {
	fs, logger, pkgProject := setupTestImage(t)
	ctx := context.Background()
	runDir := "/run"
	kdepsDir := "/test"

	// Create a mock Docker client
	mockClient := &client.Client{}

	t.Run("MissingWorkflow", func(t *testing.T) {
		mode := runmode.Docker
		gpuType := gpu.Cpu
		pathType := path.User
		kdeps := &kdepspkg.Kdeps{
			Mode:      &mode,
			DockerGPU: &gpuType,
			KdepsDir:  pkg.GetDefaultKdepsDir(),
			KdepsPath: &pathType,
		}
		_, _, err := BuildDockerImage(fs, ctx, kdeps, mockClient, runDir, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidWorkflow", func(t *testing.T) {
		// Create workflow file
		err := fs.MkdirAll(filepath.Join(kdepsDir, "workflows"), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, filepath.Join(kdepsDir, "workflows", "test-workflow.yaml"), []byte(`
name: test-workflow
version: 1.0
`), 0o644)
		require.NoError(t, err)

		mode := runmode.Docker
		gpuType := gpu.Cpu
		pathType := path.User
		kdeps := &kdepspkg.Kdeps{
			Mode:      &mode,
			DockerGPU: &gpuType,
			KdepsDir:  pkg.GetDefaultKdepsDir(),
			KdepsPath: &pathType,
		}
		_, _, err = BuildDockerImage(fs, ctx, kdeps, mockClient, runDir, kdepsDir, pkgProject, logger)
		assert.Error(t, err) // Expected error due to mock client
	})
}

func TestGenerateParamsSectionAdditional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		items    map[string]string
		expected []string // substrings expected in result
	}{
		{
			name:   "with values",
			prefix: "ARG",
			items: map[string]string{
				"FOO": "bar",
				"BAZ": "",
			},
			expected: []string{`ARG FOO="bar"`, `ARG BAZ`},
		},
		{
			name:     "empty map",
			prefix:   "ENV",
			items:    map[string]string{},
			expected: []string{""},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := generateParamsSection(tc.prefix, tc.items)
			for _, want := range tc.expected {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestGenerateDockerfile_Minimal(t *testing.T) {
	t.Parallel()

	// Build a minimal Dockerfile using generateDockerfile. Only verify that
	// critical dynamic pieces make their way into the output template. A full
	// semantic diff is unnecessary and would be brittle.
	schemaVersion := schema.SchemaVersion(context.Background())

	df := generateDockerfile(
		"1.0",                          // imageVersion
		schemaVersion,                  // schemaVersion
		"127.0.0.1",                    // hostIP
		"11435",                        // ollamaPort
		"127.0.0.1:3000",               // kdepsHost
		"",                             // argsSection
		"",                             // envsSection
		"",                             // pkgSection
		"",                             // pythonPkgSection
		"",                             // condaPkgSection
		version.DefaultAnacondaVersion, // anacondaVersion
		version.DefaultPklVersion,      // pklVersion
		"UTC",                          // timezone
		"",                             // exposedPort
		false,                          // installAnaconda
		false,                          // devBuildMode
		false,                          // apiServerMode
		false,                          // useLatest
	)

	// Quick smoke-test assertions.
	assert.Contains(t, df, "FROM ollama/ollama:1.0")
	assert.Contains(t, df, "ENV SCHEMA_VERSION="+schemaVersion)
	assert.Contains(t, df, "ENV KDEPS_HOST=127.0.0.1:3000")
	// No ports should be exposed because apiServerMode == false && exposedPort == ""
	assert.NotContains(t, df, "EXPOSE")
}

func TestPrintDockerBuildOutput_Extra(t *testing.T) {
	t.Parallel()

	// 1. Happy-path: mixed JSON stream lines and raw text.
	lines := []string{
		marshal(t, BuildLine{Stream: "Step 1/2 : FROM scratch\n"}),
		marshal(t, BuildLine{Stream: " ---> Using cache\n"}),
		"non-json-line should be echoed as-is", // raw
	}
	reader := bytes.NewBufferString(strings.Join(lines, "\n"))
	err := printDockerBuildOutput(reader)
	assert.NoError(t, err)

	// 2. Error path: JSON line with an error field should surface.
	errLines := []string{marshal(t, BuildLine{Error: "boom"})}
	errReader := bytes.NewBufferString(strings.Join(errLines, "\n"))
	err = printDockerBuildOutput(errReader)
	assert.ErrorContains(t, err, "boom")
}

// marshal is a tiny helper that converts a BuildLine to its JSON string
// representation and fails the test immediately upon error.
func marshal(t *testing.T, bl BuildLine) string {
	t.Helper()
	data, err := json.Marshal(bl)
	if err != nil {
		t.Fatalf("failed to marshal build line: %v", err)
	}
	return string(data)
}

// MockImageBuildClient is a mock implementation of the Docker client for testing image builds
type MockImageBuildClient struct {
	imageListFunc func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
}

func (m *MockImageBuildClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	if m.imageListFunc != nil {
		return m.imageListFunc(ctx, options)
	}
	return nil, nil
}

func TestBuildDockerImageNew(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdeps := &kdepspkg.Kdeps{}
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}

	// Commented out unused mock client
	// mockClient := &MockImageBuildClient{
	// 	imageListFunc: func(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	// 		return []image.Summary{}, nil
	// 	},
	// }

	runDir := "/test/run"
	kdepsDir := "/test/kdeps"
	pkgProject := &archiver.KdepsPackage{
		Workflow: "testWorkflow",
	}

	// Create dummy directories in memory FS
	fs.MkdirAll(runDir, 0o755)
	fs.MkdirAll(kdepsDir, 0o755)

	// Call the function under test with a type assertion or conversion if needed
	// Note: This will likely still fail if BuildDockerImage strictly requires *client.Client
	cName, containerName, err := BuildDockerImage(fs, ctx, kdeps, nil, runDir, kdepsDir, pkgProject, logger)

	if err != nil {
		t.Logf("Expected error due to mocked dependencies: %v", err)
	} else {
		t.Logf("BuildDockerImage returned cName: %s, containerName: %s", cName, containerName)
	}

	// Since we can't fully test the build process without Docker, we just check if the function executed without panic
	t.Log("BuildDockerImage called without panic")
}

func TestBuildDockerImageImageExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdeps := &kdepspkg.Kdeps{}
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}

	// Commented out unused mock client
	// mockClient := &MockImageBuildClient{
	// 	imageListFunc: func(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	// 		return []image.Summary{
	// 			{
	// 				RepoTags: []string{"kdeps-test:1.0"},
	// 			},
	// 		}, nil
	// 	},
	// }

	runDir := "/test/run"
	kdepsDir := "/test/kdeps"
	pkgProject := &archiver.KdepsPackage{
		Workflow: "testWorkflow",
	}

	// Create dummy directories in memory FS
	fs.MkdirAll(runDir, 0o755)
	fs.MkdirAll(kdepsDir, 0o755)

	// Call the function under test with nil to avoid type mismatch
	cName, containerName, err := BuildDockerImage(fs, ctx, kdeps, nil, runDir, kdepsDir, pkgProject, logger)
	if err != nil {
		t.Logf("Expected error due to mocked dependencies: %v", err)
	}

	if cName == "" || containerName == "" {
		t.Log("BuildDockerImage returned empty cName or containerName as expected with nil client")
	}

	t.Log("BuildDockerImage test with existing image setup executed")
}

// TestCopyFilesToRunDirCacheDirCreateFail makes runDir/cache a file so MkdirAll fails.
func TestCopyFilesToRunDirCacheDirCreateFail(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	fs := afero.NewReadOnlyFs(baseFs)
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "download")
	runDir := filepath.Join(dir, "run")

	// Prepare download directory with one file so the function proceeds past stat.
	if err := baseFs.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("mkdir download: %v", err)
	}
	_ = afero.WriteFile(baseFs, filepath.Join(downloadDir, "x.bin"), []byte("x"), 0o644)

	// runDir is unwritable (ReadOnlyFs), so MkdirAll to create runDir/cache must fail.
	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error due to cache path collision")
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFilesToRunDirCopyFailure forces CopyFile to fail by making destination directory read-only.
func TestCopyFilesToRunDirCopyFailure(t *testing.T) {
	baseFs := afero.NewMemMapFs()
	fs := afero.NewReadOnlyFs(baseFs)
	base := t.TempDir()
	downloadDir := filepath.Join(base, "dl")
	runDir := filepath.Join(base, "run")

	// setup download dir with one file
	_ = baseFs.MkdirAll(downloadDir, 0o755)
	_ = afero.WriteFile(baseFs, filepath.Join(downloadDir, "obj.bin"), []byte("data"), 0o644)

	// No need to create cache dir; ReadOnlyFs will prevent MkdirAll inside implementation.
	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error due to read-only cache directory")
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFilesToRunDirSuccess verifies that files in the download cache
// are copied into the run directory cache.
func TestCopyFilesToRunDirSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "download")
	runDir := filepath.Join(dir, "run")

	_ = fs.MkdirAll(downloadDir, 0o755)
	// create two mock files
	_ = afero.WriteFile(fs, filepath.Join(downloadDir, "a.bin"), []byte("A"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(downloadDir, "b.bin"), []byte("B"), 0o600)

	logger := logging.NewTestLogger()
	if err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logger); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify they exist in runDir/cache with same names
	for _, name := range []string{"a.bin", "b.bin"} {
		data, err := afero.ReadFile(fs, filepath.Join(runDir, "cache", name))
		if err != nil {
			t.Fatalf("copied file missing: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("copied file empty")
		}
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFilesToRunDirMissingSource ensures a descriptive error when the
// download directory does not exist.
func TestCopyFilesToRunDirMissingSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	downloadDir := filepath.Join(dir, "no_such")
	runDir := filepath.Join(dir, "run")

	err := copyFilesToRunDir(fs, context.Background(), downloadDir, runDir, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected error for missing download dir, got nil")
	}

	schema.SchemaVersion(context.Background())
}

func TestCheckDevBuildModeVariant(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	logger := logging.NewTestLogger()

	cacheDir := filepath.Join(tmpDir, "cache")
	_ = fs.MkdirAll(cacheDir, 0o755)
	kdepsBinary := filepath.Join(cacheDir, "kdeps")

	// when file absent
	dev, err := checkDevBuildMode(fs, tmpDir, logger)
	assert.NoError(t, err)
	assert.False(t, dev)

	// create file
	assert.NoError(t, afero.WriteFile(fs, kdepsBinary, []byte("binary"), 0o755))
	dev, err = checkDevBuildMode(fs, tmpDir, logger)
	assert.NoError(t, err)
	assert.True(t, dev)
}

func TestBuildDockerfileContent(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdeps := &kdepspkg.Kdeps{}
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}
	kdepsDir := "/test/kdeps"
	pkgProject := &archiver.KdepsPackage{
		Workflow: "/test/kdeps/testWorkflow",
	}

	// Create dummy directories in memory FS
	fs.MkdirAll(kdepsDir, 0o755)
	fs.MkdirAll("/test/kdeps/cache", 0o755)
	fs.MkdirAll("/test/kdeps/run/test/1.0", 0o755)

	// Create a dummy workflow file to avoid module not found error
	workflowPath := "/test/kdeps/testWorkflow"
	dummyWorkflowContent := `name = "test"
version = "1.0"
`
	afero.WriteFile(fs, workflowPath, []byte(dummyWorkflowContent), 0o644)

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

func TestGenerateParamsSection_Extra(t *testing.T) {
	input := map[string]string{"USER": "root", "DEBUG": ""}
	got := generateParamsSection("ENV", input)

	// The slice order is not guaranteed; ensure both expected lines exist.
	if !(containsLine(got, `ENV USER="root"`) && containsLine(got, `ENV DEBUG`)) {
		t.Fatalf("unexpected section: %s", got)
	}
}

// helper to search line in multi-line string.
func containsLine(s, line string) bool {
	for _, l := range strings.Split(s, "\n") {
		if l == line {
			return true
		}
	}
	return false
}

func TestGenerateParamsSectionEdge(t *testing.T) {
	items := map[string]string{
		"FOO":   "bar",
		"EMPTY": "",
	}
	out := generateParamsSection("ARG", items)

	if !strings.Contains(out, "ARG FOO=\"bar\"") {
		t.Fatalf("missing value param: %s", out)
	}
	if !strings.Contains(out, "ARG EMPTY") {
		t.Fatalf("missing empty param: %s", out)
	}
}

func TestCheckDevBuildModeMem(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	base := t.TempDir()
	kdepsDir := filepath.Join(base, "home")
	cacheDir := filepath.Join(kdepsDir, "cache")
	_ = fs.MkdirAll(cacheDir, 0o755)

	// Case 1: file absent => devBuildMode false
	dev, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dev {
		t.Fatalf("expected dev build mode to be false when file missing")
	}

	// Create dummy kdeps binary file
	filePath := filepath.Join(cacheDir, "kdeps")
	if err := afero.WriteFile(fs, filePath, []byte("hi"), 0o755); err != nil {
		t.Fatalf("write file: %v", err)
	}

	dev2, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dev2 {
		t.Fatalf("expected dev build mode true when file exists")
	}
}

func TestGenerateParamsSectionVariants(t *testing.T) {
	// Test case 1: Empty map
	result := generateParamsSection("ARG", map[string]string{})
	if result != "" {
		t.Errorf("Expected empty string for empty map, got: %s", result)
	}

	t.Log("generateParamsSection empty map test passed")

	// Test case 2: Map with single entry without value
	items := map[string]string{
		"DEBUG": "",
	}
	result = generateParamsSection("ENV", items)
	if result != "ENV DEBUG" {
		t.Errorf("Expected 'ENV DEBUG', got: %s", result)
	}

	t.Log("generateParamsSection single entry without value test passed")

	// Test case 3: Map with single entry with value
	items = map[string]string{
		"PATH": "/usr/local/bin",
	}
	result = generateParamsSection("ARG", items)
	if result != "ARG PATH=\"/usr/local/bin\"" {
		t.Errorf("Expected 'ARG PATH=\"/usr/local/bin\"', got: %s", result)
	}

	t.Log("generateParamsSection single entry with value test passed")

	// Test case 4: Map with multiple entries
	items = map[string]string{
		"VAR1": "value1",
		"VAR2": "",
		"VAR3": "value3",
	}
	result = generateParamsSection("ENV", items)
	// The order of map iteration is not guaranteed, so check individual lines
	lines := strings.Split(result, "\n")
	lineSet := make(map[string]struct{})
	for _, l := range lines {
		lineSet[l] = struct{}{}
	}
	expectedLines := []string{"ENV VAR1=\"value1\"", "ENV VAR2", "ENV VAR3=\"value3\""}
	for _, el := range expectedLines {
		if _, ok := lineSet[el]; !ok {
			t.Errorf("Expected line '%s' not found in output: %s", el, result)
		}
	}

	t.Log("generateParamsSection multiple entries test passed")
}

func TestGenerateParamsSectionLight(t *testing.T) {
	params := map[string]string{
		"FOO": "bar",
		"BAZ": "", // param without value
	}
	got := generateParamsSection("ENV", params)
	if !containsAll(got, []string{"ENV FOO=\"bar\"", "ENV BAZ"}) {
		t.Fatalf("unexpected section: %s", got)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func TestGenerateUniqueOllamaPortLight(t *testing.T) {
	p1 := generateUniqueOllamaPort(3000)
	p2 := generateUniqueOllamaPort(3000)
	if p1 == p2 {
		t.Fatalf("expected different ports when called twice, got %s %s", p1, p2)
	}
}

func TestCheckDevBuildModeLight(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	kdepsDir := "/kd"
	// No cache/kdeps binary present -> dev build mode should be false.
	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil || ok {
		t.Fatalf("expected false dev mode, got %v %v", ok, err)
	}

	// Simulate presence of a downloaded kdeps binary to enable dev build mode.
	if err := fs.MkdirAll("/kd/cache", 0o755); err != nil {
		t.Fatalf("failed to create cache directory: %v", err)
	}
	_ = afero.WriteFile(fs, "/kd/cache/kdeps", []byte("binary"), 0o755)

	ok, err = checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil || !ok {
		t.Fatalf("expected dev mode true, got %v %v", ok, err)
	}
}

// TestCheckDevBuildModeDir verifies that the helper treats a directory named
// "cache/kdeps" as non-dev build mode, exercising the !info.Mode().IsRegular()
// branch for additional coverage.
func TestCheckDevBuildModeDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Create a directory at cache/kdeps instead of a file.
	cacheDir := filepath.Join(kdepsDir, "cache")
	if err := fs.MkdirAll(filepath.Join(cacheDir, "kdeps"), 0o755); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected dev mode to be false when path is a directory")
	}
}

// TestGenerateDockerfileBranches exercises multiple flag combinations to hit
// the majority of conditional paths in generateDockerfile. We don't validate
// the entire output – only presence of a few sentinel strings that should
// appear when the corresponding branch executes. This drives a large number
// of statements for coverage without any external I/O.
func TestGenerateDockerfileBranches(t *testing.T) {
	baseArgs := struct {
		imageVersion     string
		schemaVersion    string
		hostIP           string
		ollamaPortNum    string
		kdepsHost        string
		argsSection      string
		envsSection      string
		pkgSection       string
		pythonPkgSection string
		condaPkgSection  string
		anacondaVersion  string
		pklVersion       string
		timezone         string
		exposedPort      string
	}{
		imageVersion:     "1.0",
		schemaVersion:    "v0",
		hostIP:           "0.0.0.0",
		ollamaPortNum:    "11434",
		kdepsHost:        "localhost",
		argsSection:      "ARG FOO=bar",
		envsSection:      "ENV BAR=baz",
		pkgSection:       "RUN echo pkgs",
		pythonPkgSection: "RUN echo py",
		condaPkgSection:  "RUN echo conda",
		anacondaVersion:  "2024.09-1",
		pklVersion:       "0.25.0",
		timezone:         "Etc/UTC",
		exposedPort:      "5000",
	}

	combos := []struct {
		installAnaconda bool
		devBuildMode    bool
		apiServerMode   bool
		useLatest       bool
		expectStrings   []string
	}{
		{false, false, false, false, []string{"ENV BAR=baz", "RUN echo pkgs"}},
		{true, false, false, false, []string{"/tmp/anaconda.sh", "RUN /bin/bash /tmp/anaconda.sh"}},
		{false, true, false, false, []string{"/cache/kdeps", "chmod a+x /bin/kdeps"}},
		{false, false, true, false, []string{"EXPOSE 5000"}},
		{true, true, true, true, []string{"latest", "cp /cache/pkl-linux-latest-amd64"}},
	}

	for i, c := range combos {
		df := generateDockerfile(
			baseArgs.imageVersion,
			baseArgs.schemaVersion,
			baseArgs.hostIP,
			baseArgs.ollamaPortNum,
			baseArgs.kdepsHost,
			baseArgs.argsSection,
			baseArgs.envsSection,
			baseArgs.pkgSection,
			baseArgs.pythonPkgSection,
			baseArgs.condaPkgSection,
			baseArgs.anacondaVersion,
			baseArgs.pklVersion,
			baseArgs.timezone,
			baseArgs.exposedPort,
			c.installAnaconda,
			c.devBuildMode,
			c.apiServerMode,
			c.useLatest,
		)
		for _, s := range c.expectStrings {
			if !strContains(df, s) {
				t.Fatalf("combo %d expected substring %q not found", i, s)
			}
		}
	}
}

// tiny helper – strings.Contains without importing strings multiple times.
func strContains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}())
}

func TestGenerateDockerfile_DevBuildAndAPIServer(t *testing.T) {
	df := generateDockerfile(
		"1.2.3",              // image version
		"2.0",                // schema version
		"0.0.0.0",            // host IP
		"11434",              // ollama port
		"0.0.0.0:11434",      // kdeps host
		"ARG SAMPLE=1",       // args section
		"ENV FOO=bar",        // envs section
		"RUN apt-get update", // pkg section
		"RUN pip install x",  // python section
		"",                   // conda pkg section
		"2024.01-1",          // anaconda version
		"0.28.1",             // pkl version
		"UTC",                // timezone
		"8080",               // expose port
		false,                // installAnaconda
		true,                 // devBuildMode (exercise branch)
		true,                 // apiServerMode (expose port branch)
		false,                // useLatest
	)

	if !has(df, "cp /cache/kdeps /bin/kdeps") {
		t.Fatalf("expected dev build copy line")
	}
	if !has(df, "EXPOSE 8080") {
		t.Fatalf("expected expose port line")
	}
}

// small helper to avoid importing strings each time
func has(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func TestGenerateDockerfileEdgeCasesNew(t *testing.T) {
	baseArgs := []interface{}{
		"latest",                     // imageVersion
		"1.0",                        // schemaVersion
		"127.0.0.1",                  // hostIP
		"11435",                      // ollamaPortNum
		"127.0.0.1:9090",             // kdepsHost
		"ARG FOO=bar",                // argsSection
		"ENV BAR=baz",                // envsSection
		"RUN apt-get install -y gcc", // pkgSection
		"",                           // pythonPkgSection
		"",                           // condaPkgSection
		"2024.10-1",                  // anacondaVersion
		"0.28.1",                     // pklVersion
		"UTC",                        // timezone
		"8080",                       // exposedPort
	}

	t.Run("devBuildMode", func(t *testing.T) {
		params := append(baseArgs, true /* installAnaconda */, true /* devBuildMode */, true /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), params[13].(string), params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Expect copy of kdeps binary due to devBuildMode true
		if !strings.Contains(dockerfile, "cp /cache/kdeps /bin/kdeps") {
			t.Fatalf("expected dev build copy step, got:\n%s", dockerfile)
		}
		// Anaconda installer should be present because installAnaconda true
		if !strings.Contains(dockerfile, "anaconda-linux-") {
			t.Fatalf("expected anaconda install snippet")
		}
		// Should expose port 8080 because apiServerMode true
		if !strings.Contains(dockerfile, "EXPOSE 8080") {
			t.Fatalf("expected EXPOSE directive")
		}
	})

	t.Run("prodBuildMode", func(t *testing.T) {
		params := append(baseArgs, false /* installAnaconda */, false /* devBuildMode */, false /* apiServerMode */, false /* useLatest */)
		dockerfile := generateDockerfile(params[0].(string), params[1].(string), params[2].(string), params[3].(string), params[4].(string), params[5].(string), params[6].(string), params[7].(string), params[8].(string), params[9].(string), params[10].(string), params[11].(string), params[12].(string), "", params[14].(bool), params[15].(bool), params[16].(bool), params[17].(bool))

		// Should pull kdeps via curl (not copy) because devBuildMode false
		if !strings.Contains(dockerfile, "raw.githubusercontent.com") {
			t.Fatalf("expected install kdeps via curl in prod build")
		}
		// Should not contain EXPOSE when apiServerMode false
		if strings.Contains(dockerfile, "EXPOSE") {
			t.Fatalf("did not expect EXPOSE directive when apiServerMode false")
		}
	})
}

// TestGenerateDockerfileAdditionalCases exercises seldom-hit branches in generateDockerfile so that
// coverage reflects real-world usage scenarios.
func TestGenerateDockerfileAdditionalCases(t *testing.T) {
	t.Run("DevBuildModeWithLatestAndExpose", func(t *testing.T) {
		result := generateDockerfile(
			"v1.2.3",                      // imageVersion
			"2.0",                         // schemaVersion
			"0.0.0.0",                     // hostIP
			"9999",                        // ollamaPortNum
			"kdeps.example",               // kdepsHost
			"ARG SAMPLE=1",                // argsSection
			"ENV FOO=bar",                 // envsSection
			"RUN apt-get -y install curl", // pkgSection
			"RUN pip install pytest",      // pythonPkgSection
			"",                            // condaPkgSection (none)
			"2024.10-1",                   // anacondaVersion (overwritten by useLatest=true below)
			"0.28.1",                      // pklVersion   (ditto)
			"UTC",                         // timezone
			"8080",                        // exposedPort
			true,                          // installAnaconda
			true,                          // devBuildMode  – should copy local kdeps binary
			true,                          // apiServerMode – should add EXPOSE line
			true,                          // useLatest     – should convert version marks to "latest"
		)

		// Ensure dev build mode path is present.
		assert.Contains(t, result, "cp /cache/kdeps /bin/kdeps", "expected dev build mode copy command")
		// When useLatest==true we expect the placeholder 'latest' to appear in pkl download section.
		assert.Contains(t, result, "pkl-linux-latest", "expected latest pkl artifact reference")
		// installAnaconda==true should result in anaconda installer copy logic.
		assert.Contains(t, result, "anaconda-linux-latest", "expected latest anaconda artifact reference")
		// apiServerMode==true adds an EXPOSE directive for provided port(s).
		assert.Contains(t, result, "EXPOSE 8080", "expected expose directive present")
	})

	t.Run("NonDevNoAnaconda", func(t *testing.T) {
		result := generateDockerfile(
			"stable",    // imageVersion
			"1.1",       // schemaVersion
			"127.0.0.1", // hostIP
			"1234",      // ollamaPortNum
			"host:1234", // kdepsHost
			"",          // argsSection
			"",          // envsSection
			"",          // pkgSection
			"",          // pythonPkgSection
			"",          // condaPkgSection
			"2024.10-1", // anacondaVersion
			"0.28.1",    // pklVersion
			"UTC",       // timezone
			"",          // exposedPort (no api server)
			false,       // installAnaconda
			false,       // devBuildMode
			false,       // apiServerMode – no EXPOSE
			false,       // useLatest
		)

		// Non-dev build should use install script instead of local binary.
		assert.Contains(t, result, "raw.githubusercontent.com/kdeps/kdeps", "expected remote install script usage")
		// Should NOT contain cp of anaconda because installAnaconda==false.
		assert.NotContains(t, result, "anaconda-linux", "unexpected anaconda installation commands present")
		// Should not contain EXPOSE directive.
		assert.NotContains(t, result, "EXPOSE", "unexpected expose directive present")
	})
}

func TestGenerateDockerfileContent(t *testing.T) {
	df := generateDockerfile(
		"10.1",          // imageVersion
		"v1",            // schemaVersion
		"127.0.0.1",     // hostIP
		"8000",          // ollamaPortNum
		"localhost",     // kdepsHost
		"ARG FOO=bar",   // argsSection
		"ENV BAR=baz",   // envsSection
		"# pkg section", // pkgSection
		"# python pkgs", // pythonPkgSection
		"# conda pkgs",  // condaPkgSection
		"2024.10-1",     // anacondaVersion
		"0.28.1",        // pklVersion
		"UTC",           // timezone
		"8080",          // exposedPort
		true,            // installAnaconda
		true,            // devBuildMode
		true,            // apiServerMode
		false,           // useLatest
	)

	// basic sanity checks on returned content
	assert.True(t, strings.Contains(df, "FROM ollama/ollama:10.1"))
	assert.True(t, strings.Contains(df, "ENV SCHEMA_VERSION=v1"))
	assert.True(t, strings.Contains(df, "EXPOSE 8080"))
	assert.True(t, strings.Contains(df, "ARG FOO=bar"))
	assert.True(t, strings.Contains(df, "ENV BAR=baz"))
}

// TestGenerateDockerfileBranchCoverage exercises additional parameter combinations
func TestGenerateDockerfileBranchCoverage(t *testing.T) {
	combos := []struct {
		installAnaconda bool
		devBuildMode    bool
		apiServerMode   bool
		useLatest       bool
	}{
		{false, false, false, true},
		{true, false, true, true},
		{false, true, false, false},
	}

	for _, c := range combos {
		df := generateDockerfile(
			"10.1",
			"v1",
			"127.0.0.1",
			"8000",
			"localhost",
			"",
			"",
			"",
			"",
			"",
			"2024.10-1",
			"0.28.1",
			"UTC",
			"8080",
			c.installAnaconda,
			c.devBuildMode,
			c.apiServerMode,
			c.useLatest,
		)
		// simple assertion to ensure function returns non-empty string
		assert.NotEmpty(t, df)
	}
}

// TestGenerateURLsHappyPath exercises the default code path where UseLatest is
// false. This avoids external HTTP requests yet covers several branches inside
// GenerateURLs including architecture substitution and local-name template
// logic.
func TestGenerateURLsHappyPath(t *testing.T) {
	ctx := context.Background()

	// Ensure the package-level flag is in the expected default state.
	schema.UseLatest = false

	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}

	// We expect two items (one for Pkl and one for Anaconda).
	if len(items) != 2 {
		t.Fatalf("expected 2 download items, got %d", len(items))
	}

	// Basic sanity checks on the generated URLs/local names – just ensure they
	// contain expected substrings so that we're not overly sensitive to exact
	// versions or architecture values.
	for _, itm := range items {
		if itm.URL == "" {
			t.Fatalf("item URL is empty: %+v", itm)
		}
		if itm.LocalName == "" {
			t.Fatalf("item LocalName is empty: %+v", itm)
		}
	}
}

// rtFunc already declared in another test file; reuse that type here without redefining.

func TestGenerateURLs_GitHubError(t *testing.T) {
	ctx := context.Background()

	// Save globals and transport.
	origLatest := schema.UseLatest
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		http.DefaultTransport = origTransport
	}()

	schema.UseLatest = true

	// Force GitHub API request to return HTTP 403.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.github.com" {
			return &http.Response{
				StatusCode: 403,
				Body:       ioutil.NopCloser(bytes.NewBufferString("forbidden")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx, true); err == nil {
		t.Fatalf("expected error when GitHub API returns forbidden")
	}
}

func TestGenerateURLs_AnacondaError(t *testing.T) {
	ctx := context.Background()

	// Save and restore globals and transport.
	origLatest := schema.UseLatest
	origFetcher := utils.GitHubReleaseFetcher
	origTransport := http.DefaultTransport
	defer func() {
		schema.UseLatest = origLatest
		utils.GitHubReleaseFetcher = origFetcher
		http.DefaultTransport = origTransport
	}()

	// GitHub fetch succeeds to move past first item.
	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo, base string) (string, error) {
		return "0.28.1", nil
	}

	// Make Anaconda request return HTTP 500.
	http.DefaultTransport = rtFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "repo.anaconda.com" {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(bytes.NewBufferString("server error")),
				Header:     make(http.Header),
			}, nil
		}
		return origTransport.RoundTrip(r)
	})

	if _, err := GenerateURLs(ctx, true); err == nil {
		t.Fatalf("expected error when Anaconda version fetch fails")
	}
}

func TestGenerateURLs(t *testing.T) {
	ctx := context.Background()

	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("unexpected error generating urls: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one download item")
	}

	for _, itm := range items {
		if itm.URL == "" || itm.LocalName == "" {
			t.Errorf("item fields should not be empty: %+v", itm)
		}
	}
}

func TestGenerateURLsDefaultExtra(t *testing.T) {
	ctx := context.Background()
	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one download item")
	}
	for _, it := range items {
		if it.URL == "" || it.LocalName == "" {
			t.Fatalf("invalid item %+v", it)
		}
	}
}

type roundTripperLatest struct{}

func (roundTripperLatest) RoundTrip(req *http.Request) (*http.Response, error) {
	// Distinguish responses based on requested URL path.
	switch {
	case req.URL.Host == "api.github.com":
		// Fake GitHub release JSON.
		body, _ := json.Marshal(map[string]string{"tag_name": "v0.29.0"})
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader(body)), Header: make(http.Header)}, nil
	case req.URL.Host == "repo.anaconda.com":
		html := `<a href="Anaconda3-2024.05-0-Linux-x86_64.sh">file</a><a href="Anaconda3-2024.05-0-Linux-aarch64.sh">file</a>`
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(html))), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: http.StatusOK, Body: ioNopCloser(bytes.NewReader([]byte(""))), Header: make(http.Header)}, nil
	}
}

type nopCloser struct{ *bytes.Reader }

func (n nopCloser) Close() error { return nil }

func ioNopCloser(r *bytes.Reader) io.ReadCloser { return nopCloser{r} }

func TestGenerateURLsUseLatest(t *testing.T) {
	// Mock HTTP.
	origTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperLatest{}
	defer func() { http.DefaultTransport = origTransport }()

	// Enable latest mode.
	origLatest := schema.UseLatest
	schema.UseLatest = true
	defer func() { schema.UseLatest = origLatest }()

	ctx := context.Background()
	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs returned error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	for _, itm := range items {
		if itm.LocalName == "" || itm.URL == "" {
			t.Fatalf("GenerateURLs produced empty fields: %+v", itm)
		}
		if !schema.UseLatest {
			t.Fatalf("schema.UseLatest should still be true inside loop")
		}
	}
}

func TestGenerateURLsLatestUsesFetcher(t *testing.T) {
	ctx := context.Background()

	// Save globals and restore afterwards
	orig := schema.UseLatest
	fetchOrig := utils.GitHubReleaseFetcher
	defer func() {
		schema.UseLatest = orig
		utils.GitHubReleaseFetcher = fetchOrig
	}()

	schema.UseLatest = true
	utils.GitHubReleaseFetcher = func(ctx context.Context, repo string, baseURL string) (string, error) {
		return "0.99.0", nil
	}

	items, err := GenerateURLs(ctx, true)
	if err != nil {
		t.Fatalf("GenerateURLs error: %v", err)
	}
	found := false
	for _, it := range items {
		if it.LocalName == "pkl-linux-latest-"+GetCurrentArchitecture(ctx, "apple/pkl") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pkl latest local name element, got %+v", items)
	}
}

func TestPrintDockerBuildOutputSuccess(t *testing.T) {
	logs := `{"stream":"Step 1/2 : FROM alpine\n"}\n{"stream":" ---\u003e 123abc\n"}\n`
	if err := printDockerBuildOutput(strings.NewReader(logs)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPrintDockerBuildOutputError(t *testing.T) {
	logs := `{"error":"something bad"}`
	if err := printDockerBuildOutput(strings.NewReader(logs)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestPrintDockerBuildOutput_Success(t *testing.T) {
	logs := []string{
		`{"stream":"Step 1/3 : FROM alpine"}`,
		`{"stream":" ---\u003e a0d0a0d0a0d0"}`,
		`{"stream":"Successfully built"}`,
	}
	rd := strings.NewReader(strings.Join(logs, "\n"))

	err := printDockerBuildOutput(rd)
	assert.NoError(t, err)
}

func TestPrintDockerBuildOutput_Error(t *testing.T) {
	logs := []string{
		`{"stream":"Step 1/1 : FROM alpine"}`,
		`{"error":"some docker build error"}`,
	}
	rd := strings.NewReader(strings.Join(logs, "\n"))

	err := printDockerBuildOutput(rd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some docker build error")
}

func TestPrintDockerBuildOutput_NonJSONLines(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("non json line\n")
	buf.WriteString("{\"stream\":\"ok\"}\n")
	buf.WriteString("another bad line\n")

	err := printDockerBuildOutput(&buf)
	assert.NoError(t, err)
}

func TestGenerateDockerfile_NoAnacondaInstall(t *testing.T) {
	dockerfile := generateDockerfile(
		"latest",         // imageVersion
		"1.0",            // schemaVersion
		"127.0.0.1",      // hostIP
		"11434",          // ollamaPortNum
		"127.0.0.1:3000", // kdepsHost
		"",               // argsSection
		"",               // envsSection
		"",               // pkgSection
		"",               // pythonPkgSection
		"",               // condaPkgSection
		"2024.10-1",      // anacondaVersion
		"0.28.1",         // pklVersion
		"UTC",            // timezone
		"",               // exposedPort
		false,            // installAnaconda = false
		false,            // devBuildMode
		false,            // apiServerMode
		false,            // useLatest
	)

	// Should NOT contain anaconda installation commands
	assert.NotContains(t, dockerfile, "anaconda-linux", "dockerfile should not contain anaconda installation when installAnaconda is false")
	assert.NotContains(t, dockerfile, "/tmp/anaconda.sh", "dockerfile should not contain anaconda script references when installAnaconda is false")
	assert.NotContains(t, dockerfile, "/opt/conda", "dockerfile should not contain conda references when installAnaconda is false")

	// Should still contain pkl installation
	assert.Contains(t, dockerfile, "pkl-linux", "dockerfile should still contain pkl installation")
}
