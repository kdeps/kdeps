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
	"runtime"
	"strings"
	"testing"

	dockclient "github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	docker "github.com/kdeps/kdeps/v2/pkg/infra/docker"
)

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
	if runtime.GOOS == "windows" {
		// os.CreateTemp("") on Windows resolves via TMP/TEMP, not TMPDIR.
		t.Setenv("TMP", "/nonexistent-tmpdir-for-ensure-test")
		t.Setenv("TEMP", "/nonexistent-tmpdir-for-ensure-test")
	}
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
	assert.Contains(t, err.Error(), "invalid base OS")
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
