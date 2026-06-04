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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
