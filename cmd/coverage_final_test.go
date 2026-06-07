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
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dockclient "github.com/docker/docker/api/types/image"
	dockapi "github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	docker "github.com/kdeps/kdeps/v2/pkg/infra/docker"
	kdepshttp "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

func TestDispatchExecution_BotAndFileModes(t *testing.T) {
	botWF := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "bot"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{"bot"},
				Bot:     &domain.BotConfig{ExecutionType: domain.BotExecutionTypeStateless},
			},
		},
	}
	require.Error(t, dispatchExecution(botWF, t.TempDir(), false, false, "", false))

	fileWF := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "file", TargetActionID: "act"},
		Settings:  domain.WorkflowSettings{Input: &domain.InputConfig{Sources: []string{"file"}}},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	inputFile := filepath.Join(t.TempDir(), "input.txt")
	require.NoError(t, os.WriteFile(inputFile, []byte("hello"), 0644))
	require.NoError(t, dispatchExecution(fileWF, t.TempDir(), false, false, inputFile, false))
}

func TestStartBothServersWithEngine_GracefulShutdown(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	port := mustFreePort(t)
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "both", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer:     &domain.APIServerConfig{PortNum: port},
			WebServer:     &domain.WebServerConfig{PortNum: port, Routes: []domain.WebRoute{}},
			AgentSettings: domain.AgentSettings{PythonVersion: "3.12"},
		},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	require.NoError(t, startBothServersWithEngine(eng, wf, t.TempDir(), false, false))
}

// ---------------------------------------------------------------------------
// export.go — exportISOInternal full build path
// ---------------------------------------------------------------------------

func TestExportISOInternal_FullBuild(t *testing.T) {
	installFakeLinuxkit(t)
	pkgDir, _, restore := writeISOPackageDir(t)
	defer restore()
	mockClient := newExportDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/build") {
			return bytesHTTPResponse(`{"stream":"Successfully built"}` + "\n"), nil
		}
		if strings.Contains(req.URL.Path, "/images/") && req.Method == http.MethodGet {
			body, _ := json.Marshal(dockclient.InspectResponse{Size: 50 * 1024 * 1024})
			return jsonHTTPResponse(http.StatusOK, body), nil
		}
		if strings.Contains(req.URL.Path, "/images/prune") {
			body, _ := json.Marshal(map[string]any{"SpaceReclaimed": 0})
			return jsonHTTPResponse(http.StatusOK, body), nil
		}
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	origSetup := setupDockerBuilderFunc
	t.Cleanup(func() { setupDockerBuilderFunc = origSetup })
	setupDockerBuilderFunc = func(_ *BuildFlags) (*docker.Builder, error) {
		return &docker.Builder{BaseOS: "alpine", Client: mockClient}, nil
	}
	outPath := filepath.Join(pkgDir, "out.iso")
	err := exportISOInternal(&cobra.Command{}, []string{pkgDir}, &ExportFlags{
		Format: "iso",
		Output: outPath,
	})
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

// ---------------------------------------------------------------------------
// build.go — buildImageInternal
// ---------------------------------------------------------------------------

func TestBuildImageInternal_ShowDockerfile(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "resources", "act.yaml"),
		[]byte("actionId: act\nname: Act\napiResponse:\n  success: true\n"),
		0644,
	))
	newBuildDockerClient(t, func(_ *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	origBuild := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = origBuild })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "img:tag", nil
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	flags := &BuildFlags{ShowDockerfile: true}
	err := buildImageInternal(cmd, []string{tmp}, flags)
	require.NoError(t, err)
}

func newBuildDockerClient(t *testing.T, handler gapRoundTripper) {
	t.Helper()
	cli, err := dockapi.NewClientWithOpts(
		dockapi.WithHost("tcp://127.0.0.1:2375"),
		dockapi.WithHTTPClient(&http.Client{Transport: handler}),
		dockapi.WithVersion("1.41"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	origSetup := setupDockerBuilderFunc
	t.Cleanup(func() { setupDockerBuilderFunc = origSetup })
	setupDockerBuilderFunc = func(_ *BuildFlags) (*docker.Builder, error) {
		return &docker.Builder{BaseOS: "alpine", Client: &docker.Client{Cli: cli}}, nil
	}
}

func TestBuildImageInternal_DockerBuild(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "resources", "act.yaml"),
		[]byte("actionId: act\nname: Act\napiResponse:\n  success: true\n"),
		0644,
	))
	newBuildDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/build") {
			return bytesHTTPResponse(`{"stream":"Successfully built"}` + "\n"), nil
		}
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	origBuild := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = origBuild })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "img:tag", nil
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// component.go — writeKomponentRegularFile error paths
// ---------------------------------------------------------------------------

func TestWriteKomponentRegularFile_MkdirError(t *testing.T) {
	// Target under a file (not directory) forces mkdir parent failure.
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	target := filepath.Join(blocker, "nested", "file.txt")
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	createTarGz(t, archivePath,
		[]*tar.Header{{Name: "file.txt", Typeflag: tar.TypeReg, Mode: 0644}},
		[][]byte{[]byte("data")},
	)
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer f.Close()
	gzr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	_, err = tr.Next()
	require.NoError(t, err)
	err = writeKomponentRegularFile(target, &tar.Header{Name: "f", Size: 1}, tr)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// prepackage.go — writeTempBinary write/close errors
// ---------------------------------------------------------------------------

func TestWriteTempBinary_WriteError(t *testing.T) {
	tmp := t.TempDir()
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, pattern string) (*os.File, error) {
		f, err := os.CreateTemp(tmp, pattern)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		// Re-open read-only to force write failure.
		return os.Open(f.Name())
	}
	_, err := writeTempBinary([]byte("data"), "linux", "amd64")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// embedded.go — writeCleanBinaryTemp close error
// ---------------------------------------------------------------------------

func TestWriteCleanBinaryTemp_CloseTempError(t *testing.T) {
	tmp := t.TempDir()
	srcFile, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	content := []byte("binary")
	_, err = srcFile.Write(content)
	require.NoError(t, err)
	require.NoError(t, srcFile.Close())
	src, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer src.Close()

	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, _ string) (*os.File, error) {
		// Return a file opened read-only so Write fails, or use a broken pipe.
		p := filepath.Join(tmp, "broken")
		f, createErr := os.Create(p)
		if createErr != nil {
			return nil, createErr
		}
		_ = f.Close()
		return os.Open(p)
	}
	_, _, err = writeCleanBinaryTemp(src, int64(len(content)))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// package.go — additional packaging paths
// ---------------------------------------------------------------------------

func TestPackageWorkflowWithFlags_Success(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmp, "resources", "act.yaml"),
		[]byte("actionId: act\nname: Act\napiResponse:\n  success: true\n"),
		0644,
	))
	flags := &PackageFlags{Output: tmp}
	err := PackageWorkflowWithFlags(&cobra.Command{}, []string{tmp}, flags)
	require.NoError(t, err)
}

func TestNewPackageCmd(t *testing.T) {
	c := newPackageCmd()
	assert.Equal(t, "package [workflow-directory | agency-directory]", c.Use)
}

func TestNewPrePackageCmd(t *testing.T) {
	c := newPrePackageCmd()
	assert.NotEmpty(t, c.Use)
}

func TestNewRegistrySubmitCmd(t *testing.T) {
	c := newRegistrySubmitCmd()
	assert.Equal(t, "submit [path]", c.Use)
}
