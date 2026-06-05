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

//go:build !js

package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dockclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	docker "github.com/kdeps/kdeps/v2/pkg/infra/docker"
	wasmPkg "github.com/kdeps/kdeps/v2/pkg/infra/wasm"
)

func TestCollectWebServerFiles_NoDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	files, err := collectWebServerFiles(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestCollectWebServerFiles_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	// Create files in data/ dir
	require.NoError(
		t,
		os.WriteFile(filepath.Join(dataDir, "index.html"), []byte("<html></html>"), 0644),
	)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "style.css"), []byte("body {}"), 0644))

	files, err := collectWebServerFiles(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "<html></html>", files["data/index.html"])
	assert.Equal(t, "body {}", files["data/style.css"])
}

func TestCollectWebServerFiles_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data", "sub")
	require.NoError(t, os.MkdirAll(dataDir, 0755))

	require.NoError(
		t,
		os.WriteFile(filepath.Join(dataDir, "script.js"), []byte("console.log('hi');"), 0644),
	)

	files, err := collectWebServerFiles(tmpDir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "console.log('hi');", files["data/sub/script.js"])
}

func TestResolveBuildKagencyPackage_NoArchive(t *testing.T) {
	_, _, _, err := resolveBuildKagencyPackage("/nonexistent/path.kagency")
	require.Error(t, err)
}

func TestResolveBuildAgencyFile_MissingFile(t *testing.T) {
	_, _, _, err := resolveBuildAgencyFile("/nonexistent/agency.yaml")
	require.Error(t, err)
}

func TestResolveBuildAgencyManifest_MissingFile(t *testing.T) {
	_, _, _, err := resolveBuildAgencyManifest("/nonexistent/agency.yaml", "/tmp", nil)
	require.Error(t, err)
}

func TestResolveBuildKagencyPackage_InvalidArchive(t *testing.T) {
	tmpDir := t.TempDir()
	pkgPath := filepath.Join(tmpDir, "test.kagency")
	require.NoError(t, os.WriteFile(pkgPath, []byte("not an archive"), 0644))

	_, _, _, err := resolveBuildKagencyPackage(pkgPath)
	require.Error(t, err)
}

func createValidKagencyArchive(
	t *testing.T,
	dir string,
	agencyContent string,
	agentDir string,
	wfContent string,
) string {
	t.Helper()
	archivePath := filepath.Join(dir, "test.kagency")
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add agency.yaml
	hdr := &tar.Header{
		Name: "agency.yaml",
		Size: int64(len(agencyContent)),
		Mode: 0644,
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write([]byte(agencyContent))
	require.NoError(t, err)

	// Add agent workflow
	wfBytes := []byte(wfContent)
	hdr = &tar.Header{
		Name: agentDir + "/workflow.yaml",
		Size: int64(len(wfBytes)),
		Mode: 0644,
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(wfBytes)
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return archivePath
}

func TestResolveBuildKagencyPackage_ValidArchive(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := createValidKagencyArchive(t, tmpDir,
		`apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: test-agency
  version: "1.0.0"
  targetAgentId: agent-a
agents:
  - agents/agent-a
`,
		"agents/agent-a",
		`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: agent-a
  version: "1.0.0"
  targetActionId: action
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: action
    name: Action
    apiResponse:
      success: true
`,
	)

	workflowPath, pkgDir, cleanup, err := resolveBuildKagencyPackage(archivePath)
	require.NoError(t, err)
	defer cleanup()
	assert.NotEmpty(t, workflowPath)
	assert.NotEmpty(t, pkgDir)
	assert.FileExists(t, workflowPath)
}

func TestResolveBuildAgencyFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	agentDir := filepath.Join(tmpDir, "agents", "agent-a")
	require.NoError(t, os.MkdirAll(agentDir, 0755))

	agencyContent := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: test-agency
  version: "1.0.0"
  targetAgentId: agent-a
agents:
  - agents/agent-a
`
	agencyPath := filepath.Join(tmpDir, "agency.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte(agencyContent), 0644))

	wfContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: agent-a
  version: "1.0.0"
  targetActionId: action
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: action
    name: Action
    apiResponse:
      success: true
`
	require.NoError(
		t,
		os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte(wfContent), 0644),
	)

	workflowPath, pkgDir, cleanup, err := resolveBuildAgencyFile(agencyPath)
	require.NoError(t, err)
	if cleanup != nil {
		defer cleanup()
	}
	assert.NotEmpty(t, workflowPath)
	assert.Equal(t, tmpDir, pkgDir)
	assert.FileExists(t, workflowPath)
}

func TestBundleWASMApp_Success(t *testing.T) {
	orig := bundleFunc
	t.Cleanup(func() { bundleFunc = orig })

	var captured *wasmPkg.BundleConfig
	bundleFunc = func(cfg *wasmPkg.BundleConfig) error {
		captured = cfg
		return nil
	}

	err := bundleWASMApp(
		"/path/to/kdeps.wasm",
		"/path/to/wasm_exec.js",
		"workflow: yaml",
		map[string]string{"index.html": "<html/>"},
		[]string{"/api", "/health"},
		"/tmp/out",
	)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "/path/to/kdeps.wasm", captured.WASMBinaryPath)
	assert.Equal(t, "/path/to/wasm_exec.js", captured.WASMExecJSPath)
	assert.Equal(t, "workflow: yaml", captured.WorkflowYAML)
	assert.Equal(t, map[string]string{"index.html": "<html/>"}, captured.WebServerFiles)
	assert.Equal(t, []string{"/api", "/health"}, captured.APIRoutes)
	assert.Equal(t, "/tmp/out", captured.OutputDir)
}

func TestBundleWASMApp_BundleError(t *testing.T) {
	orig := bundleFunc
	t.Cleanup(func() { bundleFunc = orig })

	sentinel := errors.New("disk full")
	bundleFunc = func(_ *wasmPkg.BundleConfig) error {
		return sentinel
	}

	err := bundleWASMApp("a", "b", "c", nil, nil, "/tmp/out")
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "WASM bundling failed")
}

func TestBuildWASMDockerImage_DefaultTag(t *testing.T) {
	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })

	var capturedArgs []string
	buildDockerImage = func(_ context.Context, args []string) error {
		capturedArgs = args
		return nil
	}

	err := buildWASMDockerImage(context.Background(), "/tmp/out", "kdeps-wasm:latest", false)
	require.NoError(t, err)
	expected := []string{"build", "-t", "kdeps-wasm:latest", "/tmp/out"}
	assert.Equal(t, expected, capturedArgs)
}

func TestBuildWASMDockerImage_CustomTag(t *testing.T) {
	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })

	var capturedArgs []string
	buildDockerImage = func(_ context.Context, args []string) error {
		capturedArgs = args
		return nil
	}

	err := buildWASMDockerImage(context.Background(), "/tmp/out", "myregistry.io/myapp:v1", false)
	require.NoError(t, err)
	expected := []string{"build", "-t", "myregistry.io/myapp:v1", "/tmp/out"}
	assert.Equal(t, expected, capturedArgs)
}

func TestBuildWASMDockerImage_NoCache(t *testing.T) {
	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })

	var capturedArgs []string
	buildDockerImage = func(_ context.Context, args []string) error {
		capturedArgs = args
		return nil
	}

	err := buildWASMDockerImage(context.Background(), "/tmp/out", "kdeps-wasm:latest", true)
	require.NoError(t, err)
	expected := []string{"build", "-t", "kdeps-wasm:latest", "--no-cache", "/tmp/out"}
	assert.Equal(t, expected, capturedArgs)
}

func TestBuildWASMDockerImage_DockerError(t *testing.T) {
	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })

	buildDockerImage = func(_ context.Context, _ []string) error {
		return errors.New("docker daemon not available")
	}

	err := buildWASMDockerImage(context.Background(), "/tmp/out", "kdeps-wasm:latest", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker build failed")
	assert.Contains(t, err.Error(), "docker daemon not available")
}

// --- buildWASMImage tests ---

// createMinimalWASMWorkflow writes a minimal valid workflow.yaml into dir.
func createMinimalWASMWorkflow(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-wasm-workflow
  targetActionId: test-action
settings: {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(content), 0644))
}

// createWASMWorkflowWithAPIRoutes writes a workflow with API server routes into dir.
func createWASMWorkflowWithAPIRoutes(t *testing.T, dir string) {
	t.Helper()
	content := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-wasm-workflow
  targetActionId: test-action
settings:
  apiServer:
    routes:
      - path: "/api/v1/chat"
        methods: ["POST"]
      - path: "/api/v1/status"
        methods: ["GET"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(content), 0644))
}

// setupWASMEnv creates temp WASM binary and exec JS files in dir and sets
// the corresponding env vars so findWASMBinary/findWASMExecJS return them.
func setupWASMEnv(t *testing.T, dir string) {
	t.Helper()
	wasmBin := filepath.Join(dir, "kdeps_test.wasm")
	require.NoError(t, os.WriteFile(wasmBin, []byte("mock wasm binary"), 0644))
	t.Setenv("KDEPS_WASM_BINARY", wasmBin)

	wasmExecJS := filepath.Join(dir, "wasm_exec_test.js")
	require.NoError(t, os.WriteFile(wasmExecJS, []byte("mock wasm_exec.js"), 0644))
	t.Setenv("KDEPS_WASM_EXEC_JS", wasmExecJS)
}

func TestBuildWASMImage_InvalidPath(t *testing.T) {
	err := buildWASMImage(context.Background(), "/nonexistent/test/path", &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access path")
}

func TestBuildWASMImage_InvalidWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte("invalid: yaml: ["),
		0644,
	))

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse workflow")
}

func TestBuildWASMImage_MissingWASMBinary(t *testing.T) {
	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kdeps.wasm not found")
}

func TestBuildWASMImage_Success(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	// Create data/ web server files to exercise collectWebServerFiles.
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "index.html"), []byte("<html/>"), 0644))

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.NoError(t, err)
}

func TestBuildWASMImage_BundleError(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	sentinel := errors.New("bundler failure")
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return sentinel }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "WASM bundling failed")
}

func TestBuildWASMImage_DockerError(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	sentinel := errors.New("docker daemon not reachable")
	buildDockerImage = func(_ context.Context, _ []string) error { return sentinel }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), "docker build failed")
}

func TestBuildWASMImage_WithAPIRoutes(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	var capturedConfig *wasmPkg.BundleConfig
	bundleFunc = func(cfg *wasmPkg.BundleConfig) error {
		capturedConfig = cfg
		return nil
	}

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createWASMWorkflowWithAPIRoutes(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.NoError(t, err)
	require.NotNil(t, capturedConfig)
	assert.Equal(t, []string{"/api/v1/chat", "/api/v1/status"}, capturedConfig.APIRoutes)
	assert.Contains(t, capturedConfig.WorkflowYAML, "/api/v1/chat")
	assert.Contains(t, capturedConfig.WorkflowYAML, "/api/v1/status")
}

// ---------------------------------------------------------------------------
// findWASMBinary / findWASMExecJS with osExecutable error
// ---------------------------------------------------------------------------

func TestFindWASMBinary_OsExecutableError(t *testing.T) {
	orig := osExecutable
	t.Cleanup(func() { osExecutable = orig })
	osExecutable = func() (string, error) {
		return "", errors.New("executable not found")
	}

	// With osExecutable failing and no env var, should fall through to CWD check.
	// CWD won't have kdeps.wasm, so it returns an error.
	_, err := findWASMBinary()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kdeps.wasm not found")
}

func TestFindWASMExecJS_OsExecutableError(t *testing.T) {
	orig := osExecutable
	t.Cleanup(func() { osExecutable = orig })
	osExecutable = func() (string, error) {
		return "", errors.New("executable not found")
	}

	// With osExecutable failing and no env var, should fall through to CWD and Go SDK.
	// CWD won't have wasm_exec.js, and "go env GOROOT" may or may not work.
	_, err := findWASMExecJS(context.Background())
	// Should either find it via Go SDK or return an error — both paths are valid.
	// We just want to exercise the osExecutable error branch.
	t.Logf("findWASMExecJS with osExecutable error: %v", err)
}

func TestFindWASMBinary_OsExecutableSuccess_FileMissing(t *testing.T) {
	orig := osExecutable
	t.Cleanup(func() { osExecutable = orig })

	tmpDir := t.TempDir()
	osExecutable = func() (string, error) {
		return filepath.Join(tmpDir, "kdeps"), nil
	}

	// osExecutable succeeds but kdeps.wasm doesn't exist next to it.
	_, err := findWASMBinary()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "kdeps.wasm not found")
}

func TestFindWASMExecJS_OsExecutableSuccess_FilePresent(t *testing.T) {
	orig := osExecutable
	t.Cleanup(func() { osExecutable = orig })

	tmpDir := t.TempDir()
	wasmExecJSPath := filepath.Join(tmpDir, "wasm_exec.js")
	require.NoError(t, os.WriteFile(wasmExecJSPath, []byte("// mock"), 0644))

	osExecutable = func() (string, error) {
		return filepath.Join(tmpDir, "kdeps"), nil
	}

	path, err := findWASMExecJS(context.Background())
	require.NoError(t, err)
	assert.Equal(t, wasmExecJSPath, path)
}

// ---------------------------------------------------------------------------
// performDockerBuild tests (mock builder.Build via dockerBuildImageFunc)
// ---------------------------------------------------------------------------

func newMockDockerClientForBuild(t *testing.T, handler func(*http.Request) (*http.Response, error)) *docker.Client {
	t.Helper()
	cli, err := dockclient.NewClientWithOpts(
		dockclient.WithHost("tcp://127.0.0.1:2375"),
		dockclient.WithHTTPClient(&http.Client{Transport: roundTripFuncMock(handler)}),
		dockclient.WithVersion("1.41"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	return &docker.Client{Cli: cli}
}

type roundTripFuncMock func(*http.Request) (*http.Response, error)

func (f roundTripFuncMock) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPerformDockerBuild_BuildError(t *testing.T) {
	orig := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = orig })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "", errors.New("build failed: no space left")
	}

	builder := &docker.Builder{BaseOS: "alpine"}
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}
	err := performDockerBuild(builder, wf, "/tmp/pkg", &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build image")
}

func TestPerformDockerBuild_BuildSuccess_NoTag(t *testing.T) {
	orig := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = orig })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "kdeps-test:latest", nil
	}

	builder := &docker.Builder{BaseOS: "alpine"}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: 8080},
		},
	}
	err := performDockerBuild(builder, wf, "/tmp/pkg", &BuildFlags{})
	require.NoError(t, err)
}

func TestPerformDockerBuild_BuildSuccess_WithTag(t *testing.T) {
	orig := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = orig })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "kdeps-test:latest", nil
	}

	mockClient := newMockDockerClientForBuild(t, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader("")),
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}, nil
	})

	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}
	err := performDockerBuild(builder, wf, "/tmp/pkg", &BuildFlags{Tag: "myrepo/test:v1"})
	require.NoError(t, err)
}

func TestPerformDockerBuild_BuildSuccess_TagError(t *testing.T) {
	orig := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = orig })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "kdeps-test:latest", nil
	}

	mockClient := newMockDockerClientForBuild(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("tag failed: permission denied")
	})

	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}
	err := performDockerBuild(builder, wf, "/tmp/pkg", &BuildFlags{Tag: "bad/tag"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to tag image")
}

// ---------------------------------------------------------------------------
// createPrepackagedBinariesForDocker tests
// ---------------------------------------------------------------------------

func TestCreatePrepackagedBinariesForDocker_OsExecutableError(t *testing.T) {
	origOsExec := osExecutable
	origResolve := resolveBaseBinary
	t.Cleanup(func() {
		osExecutable = origOsExec
		resolveBaseBinary = origResolve
	})

	osExecutable = func() (string, error) {
		return "", errors.New("executable not found")
	}
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return "", false, errors.New("resolve failed")
	}

	binaries, cleanup := createPrepackagedBinariesForDocker(
		context.Background(),
		"/nonexistent/kdepsfile",
	)
	defer cleanup()
	assert.Empty(t, binaries)
}

// ---------------------------------------------------------------------------
// ensureKdepsFile tests
// ---------------------------------------------------------------------------

func TestEnsureKdepsFile_AlreadyKdepsFile(t *testing.T) {
	tmpDir := t.TempDir()
	kdepsPath := filepath.Join(tmpDir, "test.kdeps")
	require.NoError(t, os.WriteFile(kdepsPath, []byte(""), 0644))

	path, created, err := ensureKdepsFile(kdepsPath, "", &domain.Workflow{})
	require.NoError(t, err)
	assert.Equal(t, kdepsPath, path)
	assert.False(t, created)
}

func TestEnsureKdepsFile_NonKdepsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte(
			"apiVersion: kdeps.io/v1\n"+
				"kind: Workflow\n"+
				"metadata:\n"+
				"  name: test\n"+
				"  targetActionId: test-action\n"+
				"settings: {}\n",
		),
		0644,
	))

	path, created, err := ensureKdepsFile(filepath.Join(tmpDir, "workflow.yaml"), tmpDir, &domain.Workflow{})
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.True(t, created)
	assert.FileExists(t, path)
	os.Remove(path)
}

func TestEnsureKdepsFile_CreatePackageArchiveError(t *testing.T) {
	tmpDir := t.TempDir()
	path, created, err := ensureKdepsFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		"/nonexistent-pkg-dir-for-test",
		&domain.Workflow{},
	)
	require.Error(t, err)
	assert.Empty(t, path)
	assert.False(t, created)
	assert.Contains(t, err.Error(), "failed to create .kdeps archive")
}

func TestEnsureKdepsFile_CreateTempError(t *testing.T) {
	t.Setenv("TMPDIR", "/nonexistent-tmpdir-for-ensure-test")
	path, created, err := ensureKdepsFile("/some/path/workflow.yaml", "/some/pkgdir", &domain.Workflow{})
	require.Error(t, err)
	assert.Empty(t, path)
	assert.False(t, created)
	assert.Contains(t, err.Error(), "failed to create temp .kdeps file")
}

// ---------------------------------------------------------------------------
// getWorkflowPorts tests
// ---------------------------------------------------------------------------

func TestGetWorkflowPorts_NilWorkflow(t *testing.T) {
	ports := getWorkflowPorts(nil)
	assert.Equal(t, []int{16395}, ports)
}

func TestGetWorkflowPorts_DefaultPort(t *testing.T) {
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{},
	}
	ports := getWorkflowPorts(wf)
	assert.Equal(t, []int{16395}, ports)
}

func TestGetWorkflowPorts_APIServerPort(t *testing.T) {
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: 8080},
		},
	}
	ports := getWorkflowPorts(wf)
	assert.Equal(t, []int{8080}, ports)
}

func TestGetWorkflowPorts_OllamaEnabled(t *testing.T) {
	installOllama := true
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
			},
		},
	}
	ports := getWorkflowPorts(wf)
	assert.Equal(t, []int{16395, 11434}, ports)
}

func TestGetWorkflowPorts_CustomPortAndOllama(t *testing.T) {
	installOllama := true
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: 9090},
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
			},
		},
	}
	ports := getWorkflowPorts(wf)
	assert.Equal(t, []int{9090, 11434}, ports)
}

// ---------------------------------------------------------------------------
// handleDockerfileShow tests
// ---------------------------------------------------------------------------

func TestHandleDockerfileShow_Success(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}
	err := handleDockerfileShow(builder, wf)
	require.NoError(t, err)
}

func TestHandleDockerfileShow_Error(t *testing.T) {
	builder := &docker.Builder{BaseOS: "unsupported-os"}
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}
	err := handleDockerfileShow(builder, wf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate Dockerfile")
	assert.Contains(t, err.Error(), "unsupported base OS")
}

// ---------------------------------------------------------------------------
// buildWASMImage with WASM env set but wasm_exec.js missing
// ---------------------------------------------------------------------------

func TestBuildWASMImage_MissingWASMExecJS(t *testing.T) {
	origOsExecutable := osExecutable
	t.Cleanup(func() { osExecutable = origOsExecutable })

	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)

	// Set KDEPS_WASM_BINARY to a valid file (WASM env already set)
	wasmBin := filepath.Join(tmpDir, "kdeps_test.wasm")
	require.NoError(t, os.WriteFile(wasmBin, []byte("mock wasm binary"), 0644))
	t.Setenv("KDEPS_WASM_BINARY", wasmBin)

	// osExecutable returns a valid path in tmpDir, but wasm_exec.js does not
	// exist there. Setting GOROOT to a non-existent path prevents
	// findWASMExecJS from finding the file via Go SDK.
	osExecutable = func() (string, error) {
		return filepath.Join(tmpDir, "kdeps"), nil
	}
	t.Setenv("GOROOT", "/nonexistent-goroot")

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wasm_exec.js not found")
}

func TestBuildWASMImage_CollectWebServerFilesError(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	// Create a data/ directory and remove permissions so Walk fails.
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "test.txt"), []byte("content"), 0644))
	require.NoError(t, os.Chmod(dataDir, 0000))
	t.Cleanup(func() { _ = os.Chmod(dataDir, 0755) })

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to collect web server files")
}

func TestBuildWASMImage_MkdirTempError(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	tmpDir := t.TempDir()
	createMinimalWASMWorkflow(t, tmpDir)
	setupWASMEnv(t, tmpDir)

	// Point TMPDIR at a non-existent directory so os.MkdirTemp fails.
	t.Setenv("TMPDIR", "/nonexistent-mkdir-tmp")

	err := buildWASMImage(context.Background(), tmpDir, &BuildFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

// TestBuildWASMImage_WithKdepsPackage exercises the cleanupFunc non-nil path,
// covering the defer cleanupFunc() branch in buildWASMImage.
func TestBuildWASMImage_WithKdepsPackage(t *testing.T) {
	origBundle := bundleFunc
	t.Cleanup(func() { bundleFunc = origBundle })
	bundleFunc = func(_ *wasmPkg.BundleConfig) error { return nil }

	origBuildDockerImage := buildDockerImage
	t.Cleanup(func() { buildDockerImage = origBuildDockerImage })
	buildDockerImage = func(_ context.Context, _ []string) error { return nil }

	// Create source dir with workflow and package it.
	srcDir := t.TempDir()
	createMinimalWASMWorkflow(t, srcDir)
	setupWASMEnv(t, srcDir)

	pkgPath := filepath.Join(t.TempDir(), "test.kdeps")
	require.NoError(t, CreatePackageArchive(srcDir, pkgPath, &domain.Workflow{}))

	// Calling buildWASMImage with a .kdeps file triggers the cleanup
	// function returned by resolveBuildWorkflowPaths.
	err := buildWASMImage(context.Background(), pkgPath, &BuildFlags{})
	require.NoError(t, err)
}

func TestCreatePrepackagedBinariesForDocker_ResolveError(t *testing.T) {
	origResolve := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = origResolve })

	var callCount int
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		callCount++
		return "", false, errors.New("resolve error")
	}

	binaries, cleanup := createPrepackagedBinariesForDocker(
		context.Background(),
		"/nonexistent/kdepsfile",
	)
	defer cleanup()
	assert.Empty(t, binaries)
	assert.Equal(t, 2, callCount, "both targets should be attempted")
}

func TestCreatePrepackagedBinariesForDocker_CreateTempError(t *testing.T) {
	origResolve := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = origResolve })

	// Create base binary stubs before changing TMPDIR.
	tmpDir := t.TempDir()
	baseAmd64 := filepath.Join(tmpDir, "base-kdeps-amd64")
	baseArm64 := filepath.Join(tmpDir, "base-kdeps-arm64")
	require.NoError(t, os.WriteFile(baseAmd64, []byte("mock binary"), 0644))
	require.NoError(t, os.WriteFile(baseArm64, []byte("mock binary"), 0644))

	resolveBaseBinary = func(_ context.Context, _ string, target archTarget, _ string) (string, bool, error) {
		if target.GOARCH == "amd64" {
			return baseAmd64, true, nil
		}
		return baseArm64, true, nil
	}

	// Point TMPDIR to a non-existent directory so os.CreateTemp fails.
	t.Setenv("TMPDIR", "/nonexistent-path-for-test")

	binaries, cleanup := createPrepackagedBinariesForDocker(
		context.Background(),
		"/nonexistent/kdepsfile",
	)
	defer cleanup()
	assert.Empty(t, binaries)
}
