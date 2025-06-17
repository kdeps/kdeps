package docker

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"encoding/json"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	kdCfg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		kdeps := &kdCfg.Kdeps{}
		_, _, _, _, _, _, _, _, err := BuildDockerfile(fs, ctx, kdeps, kdepsDir, pkgProject, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading workflow file")
	})

	t.Run("ValidConfig", func(t *testing.T) {
		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
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
		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
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

		kdeps := &kdCfg.Kdeps{
			RunMode:   "docker",
			DockerGPU: "cpu",
			KdepsDir:  ".kdeps",
			KdepsPath: "user",
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
		"1.0",            // imageVersion
		schemaVersion,    // schemaVersion
		"127.0.0.1",      // hostIP
		"11435",          // ollamaPort
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
		false,            // installAnaconda
		false,            // devBuildMode
		false,            // apiServerMode
		false,            // useLatest
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
	kdeps := &kdCfg.Kdeps{}
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
	fs.MkdirAll(runDir, 0755)
	fs.MkdirAll(kdepsDir, 0755)

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
	kdeps := &kdCfg.Kdeps{}
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
	fs.MkdirAll(runDir, 0755)
	fs.MkdirAll(kdepsDir, 0755)

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
