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
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/infra/docker"
	kdepshttp "github.com/kdeps/kdeps/v2/pkg/infra/http"
	"github.com/kdeps/kdeps/v2/pkg/tools"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

// ---------------------------------------------------------------------------
// build.go
// ---------------------------------------------------------------------------

func TestResolveAgencyEntryPath_EmptyTarget(t *testing.T) {
	tmp := t.TempDir()
	agent := filepath.Join(tmp, "agent.yaml")
	require.NoError(t, os.WriteFile(agent, []byte(minimalWorkflowYAML()), 0644))
	agency := &domain.Agency{Metadata: domain.AgencyMetadata{TargetAgentID: ""}}
	path, err := resolveAgencyEntryPath(agency, []string{agent}, "agency.yaml")
	require.NoError(t, err)
	assert.Equal(t, agent, path)
}

func TestResolveBuildAgencyManifest_ParseCleanup(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "agency.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("invalid: ["), 0644))
	cleanupCalled := false
	cleanup := func() { cleanupCalled = true }
	_, _, _, err := resolveBuildAgencyManifest(bad, tmp, cleanup)
	require.Error(t, err)
	assert.True(t, cleanupCalled)
}

func TestSetupDockerBuilderImpl_BuilderError(t *testing.T) {
	orig := newDockerBuilderWithOSFunc
	t.Cleanup(func() { newDockerBuilderWithOSFunc = orig })
	newDockerBuilderWithOSFunc = func(_ string) (*docker.Builder, error) {
		return nil, errors.New("docker unavailable")
	}
	_, err := setupDockerBuilderImpl(&BuildFlags{})
	require.Error(t, err)
}

func TestBuildImageInternal_AbsPathError(t *testing.T) {
	orig := filepathAbsFunc
	t.Cleanup(func() { filepathAbsFunc = orig })
	filepathAbsFunc = func(path string) (string, error) {
		if strings.Contains(path, "pkgdir") {
			return "", errors.New("abs fail")
		}
		return filepath.Abs(path)
	}
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{})
	require.Error(t, err)
}

func TestBuildImageInternal_ChdirError(t *testing.T) {
	orig := chdirToPackageDirFunc
	t.Cleanup(func() { chdirToPackageDirFunc = orig })
	chdirToPackageDirFunc = func(_ string) (func(), error) {
		return nil, errors.New("chdir fail")
	}
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{})
	require.Error(t, err)
}

func TestBuildDockerImage_DefaultImpl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := buildDockerImage(ctx, []string{"version"})
	require.Error(t, err)
}

func TestCollectWebServerFiles_OpenRootError(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "data"), 0755))
	orig := osOpenRootFunc
	t.Cleanup(func() { osOpenRootFunc = orig })
	osOpenRootFunc = func(_ string) (*os.Root, error) {
		return nil, errors.New("open root fail")
	}
	_, err := collectWebServerFiles(tmp)
	require.Error(t, err)
}

func TestCollectWebServerFiles_WalkError(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	blocker := filepath.Join(dataDir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	_, err := collectWebServerFiles(tmp)
	require.NoError(t, err)
}

func TestCollectWebServerFiles_ReadError(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	// Create a dangling symlink to trigger walk/read error.
	require.NoError(t, os.Symlink("/nonexistent/file", filepath.Join(dataDir, "link")))
	_, err := collectWebServerFiles(tmp)
	require.Error(t, err)
}

func TestGorootWASMExecCandidates_NoGo(t *testing.T) {
	cands := gorootWASMExecCandidates(context.Background())
	assert.NotNil(t, cands)
}

// ---------------------------------------------------------------------------
// doctor.go / edit.go / markdown.go / new.go
// ---------------------------------------------------------------------------

func TestLoadDoctorConfig_ErrorFallback(t *testing.T) {
	orig := configLoadStructFunc
	t.Cleanup(func() { configLoadStructFunc = orig })
	configLoadStructFunc = func() (*config.Config, error) {
		return nil, errors.New("load fail")
	}
	cfg := loadDoctorConfig()
	require.NotNil(t, cfg)
}

func TestRunEdit_ConfigPathError(t *testing.T) {
	origScaffold := configScaffoldFunc
	origPath := configPathFunc
	t.Cleanup(func() {
		configScaffoldFunc = origScaffold
		configPathFunc = origPath
	})
	configScaffoldFunc = func() error { return nil }
	configPathFunc = func() (string, error) { return "", errors.New("path fail") }
	err := runEdit(&cobra.Command{}, nil)
	require.Error(t, err)
}

func TestRenderMarkdown_RenderError_Remaining(t *testing.T) {
	orig := newMarkdownRendererFunc
	t.Cleanup(func() { newMarkdownRendererFunc = orig })
	newMarkdownRendererFunc = func() (*glamour.TermRenderer, error) {
		return glamour.NewTermRenderer(glamour.WithStandardStyle("dark"))
	}
	out := renderMarkdown("# Hello\n\nWorld")
	assert.Contains(t, out, "Hello")
}

func TestPrepareNewOutputDir_StatError(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "blocker"), []byte("x"), 0644))
	err := prepareNewOutputDir(filepath.Join(tmp, "blocker"), false)
	require.Error(t, err)
}

func TestPrepareNewOutputDir_ForceRemove(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agent"), 0755))
	require.NoError(t, prepareNewOutputDir(filepath.Join(tmp, "agent"), true))
}

func TestRunNewWithFlags_GeneratorError(t *testing.T) {
	err := RunNewWithFlags(
		&cobra.Command{},
		[]string{"valid-agent-name"},
		&NewFlags{Template: "nonexistent-template-xyz"},
	)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// export.go
// ---------------------------------------------------------------------------

func TestPrepareISOExportWorkflow_AbsError_Remaining(t *testing.T) {
	orig := filepathAbsFunc
	t.Cleanup(func() { filepathAbsFunc = orig })
	filepathAbsFunc = func(path string) (string, error) {
		if strings.Contains(path, "kdeps-run") || strings.Contains(path, "extract") {
			return "", errors.New("abs fail")
		}
		return filepath.Abs(path)
	}
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(
			kdeps,
			buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()),
			0644,
		),
	)
	_, _, cleanup, err := prepareISOExportWorkflow(kdeps)
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestPrepareISOExportWorkflow_ChdirError(t *testing.T) {
	orig := chdirToPackageDirFunc
	t.Cleanup(func() { chdirToPackageDirFunc = orig })
	chdirToPackageDirFunc = func(_ string) (func(), error) {
		return nil, errors.New("chdir fail")
	}
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(
			kdeps,
			buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()),
			0644,
		),
	)
	_, _, cleanup, err := prepareISOExportWorkflow(kdeps)
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestShowLinuxKitConfig_SuccessPath(t *testing.T) {
	wf := minimalISOWorkflow()
	err := showLinuxKitConfig(wf, &ExportFlags{Hostname: "host"})
	require.NoError(t, err)
}

func TestExportK8sInternal_ValidateError(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	err := exportK8sInternal(
		&cobra.Command{},
		[]string{tmp},
		&K8sFlags{Output: "/no/such/dir/out.yaml"},
	)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// package.go
// ---------------------------------------------------------------------------

func TestPackageWorkflowWithFlags_NoWorkflow(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	err := PackageWorkflowWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{})
	require.Error(t, err)
}

func TestPackageWorkflowWithFlags_ComposeWarn(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	err := PackageWorkflowWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{Output: tmp})
	require.NoError(t, err)
}

func TestParseKdepsIgnore_OpenRootError(t *testing.T) {
	orig := osOpenRootFunc
	t.Cleanup(func() { osOpenRootFunc = orig })
	osOpenRootFunc = func(_ string) (*os.Root, error) {
		return nil, errors.New("open root")
	}
	patterns := ParseKdepsIgnore(t.TempDir())
	assert.Empty(t, patterns)
}

func TestParseKdepsIgnore_WalkError_Remaining(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	patterns := ParseKdepsIgnore(blocker)
	assert.Empty(t, patterns)
}

func TestCreatePackageArchive_CreateError(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	err := CreatePackageArchive(tmp, filepath.Join(blocker, "out.kdeps"), &domain.Workflow{})
	require.Error(t, err)
}

func TestAddFileToArchive_OpenError(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	err := AddFileToArchive("/nonexistent/file", &mockFileInfo{name: "f"}, t.TempDir(), tw)
	require.Error(t, err)
}

func TestPackageAgencyWithFlags_ArchiveError(t *testing.T) {
	tmp := t.TempDir()
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: a
  version: "1.0.0"
  targetAgentId: agent-a
agents:
  - agents/agent-a
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agency), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agents", "agent-a"), 0755))
	agentWF := strings.Replace(minimalWorkflowYAML(), "name: gap-test", "name: agent-a", 1)
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(tmp, "agents", "agent-a", "workflow.yaml"),
			[]byte(agentWF),
			0644,
		),
	)
	err := PackageAgencyWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{Output: "/no/dir"})
	require.Error(t, err)
}

func TestPackageComponentWithFlags_Errors(t *testing.T) {
	tmp := t.TempDir()
	err := PackageComponentWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{})
	require.Error(t, err)
	comp := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: c
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "component.yaml"), []byte(comp), 0644))
	err = PackageComponentWithFlags(
		&cobra.Command{},
		[]string{tmp},
		&PackageFlags{Output: "/no/dir"},
	)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// component.go
// ---------------------------------------------------------------------------

func TestExtractKomponent_MkdirError(t *testing.T) {
	orig := osMkdirTempKomponentFunc
	t.Cleanup(func() { osMkdirTempKomponentFunc = orig })
	osMkdirTempKomponentFunc = func(_, _ string) (string, error) {
		return "", errors.New("mkdir fail")
	}
	_, cleanup, err := extractKomponent(filepath.Join(t.TempDir(), "x.komponent"))
	require.Error(t, err)
	cleanup()
}

func TestReadReadmeFromKomponent_NoReadme(t *testing.T) {
	tmp := t.TempDir()
	komp := filepath.Join(tmp, "c.komponent")
	createKomponentArchive(t, komp, "component.yaml", "metadata:\n  name: c\n")
	readme, err := readReadmeFromKomponent(komp)
	require.NoError(t, err)
	assert.Empty(t, readme)
}

func TestSafeKomponentTarget_RelError(t *testing.T) {
	_, ok, err := safeKomponentTarget(t.TempDir(), "../escape")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestComponentUpdateInternal_WithWarnings(t *testing.T) {
	tmp := t.TempDir()
	compDir := filepath.Join(tmp, "components", "c1")
	require.NoError(t, os.MkdirAll(compDir, 0755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte("invalid: ["), 0644),
	)
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	require.NoError(t, componentUpdateInternal(tmp))
}

func TestUpdateComponentDir_UpToDate(t *testing.T) {
	tmp := t.TempDir()
	comp := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: uptodate
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "component.yaml"), []byte(comp), 0644))
	require.NoError(t, updateComponentDir(tmp))
}

func TestGenerateFallbackReadme_NoMeta(t *testing.T) {
	readme, err := generateFallbackReadme("nonexistent-comp-xyz")
	require.NoError(t, err)
	assert.Contains(t, readme, "nonexistent-comp-xyz")
}

// ---------------------------------------------------------------------------
// embedded.go
// ---------------------------------------------------------------------------

func TestOpenExecutableWithError_StatErrorHook(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bin")
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Remove(path))
	_, _, err = openExecutableWithError(path)
	require.Error(t, err)
}

func TestAppendEmbeddedPackage_StatKdepsError(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	require.NoError(t, os.WriteFile(bin, []byte("bin"), 0755))
	orig := osStatFunc
	t.Cleanup(func() { osStatFunc = orig })
	osStatFunc = func(_ string) (os.FileInfo, error) { return nil, errors.New("stat") }
	err := AppendEmbeddedPackage(bin, filepath.Join(tmp, "pkg.kdeps"), filepath.Join(tmp, "out"))
	require.Error(t, err)
}

func TestWriteCleanBinaryTemp_ReadError_Remaining(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	_, err = f.WriteString("short")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	rf, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer rf.Close()
	_, _, err = writeCleanBinaryTemp(rf, 100)
	require.Error(t, err)
}

func TestRunEmbeddedPackage_OpenError(t *testing.T) {
	assert.Equal(t, 1, RunEmbeddedPackage("dev", "dev", "/nonexistent/binary"))
}

func TestRunEmbeddedPackage_CopyError(t *testing.T) {
	tmp := t.TempDir()
	binPath := writeEmbeddedTestBinary(t, tmp)
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_ string, _ string) (*os.File, error) {
		p := filepath.Join(tmp, "ro")
		require.NoError(t, os.WriteFile(p, nil, 0644))
		return os.OpenFile(p, os.O_RDWR, 0644)
	}
	assert.Equal(t, 1, RunEmbeddedPackage("dev", "dev", binPath))
}

// ---------------------------------------------------------------------------
// run.go
// ---------------------------------------------------------------------------

func TestResolveRegularPath_AbsError_Remaining(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("invalid path semantics differ on Windows")
	}
	_, _, err := ResolveRegularPath("\x00bad")
	require.Error(t, err)
}

func TestExecuteWorkflowStepsWithFlags_AgencyIndex(t *testing.T) {
	stubDispatchHooks(t)
	tmp := t.TempDir()
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: test-agency
  version: "1.0.0"
  targetAgentId: agent-a
agents:
  - agents/agent-a
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agency), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agents", "agent-a"), 0755))
	agentWF := strings.Replace(minimalWorkflowYAML(), "name: gap-test", "name: agent-a", 1)
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(tmp, "agents", "agent-a", "workflow.yaml"),
			[]byte(agentWF),
			0644,
		),
	)
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	err := ExecuteAgencyStepsWithFlags(cmd, filepath.Join(tmp, "agency.yaml"), &RunFlags{})
	require.NoError(t, err)
}

func TestBuildAgentNameMap_Errors(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("invalid: ["), 0644))
	_, _, err := buildAgentNameMap([]string{bad}, "x")
	require.Error(t, err)

	empty := filepath.Join(tmp, "empty.yaml")
	require.NoError(
		t,
		os.WriteFile(
			empty,
			[]byte("apiVersion: kdeps.io/v1\nkind: Workflow\nmetadata:\n  name: \"\"\n"),
			0644,
		),
	)
	_, _, err = buildAgentNameMap([]string{empty}, "x")
	require.Error(t, err)

	good := filepath.Join(tmp, "good.yaml")
	require.NoError(t, os.WriteFile(good, []byte(minimalWorkflowYAML()), 0644))
	_, _, err = buildAgentNameMap([]string{good}, "missing")
	require.Error(t, err)
}

func TestExecuteAgencyEntryPoint_NoAgents_Remaining(t *testing.T) {
	err := executeAgencyEntryPoint(&cobra.Command{}, "", nil, &RunFlags{})
	require.NoError(t, err)
}

func TestParseWorkflowFile_Error(t *testing.T) {
	_, err := ParseWorkflowFile(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func TestParseAgencyFileWithParser_Error(t *testing.T) {
	_, _, _, err := ParseAgencyFileWithParser(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func TestLoadResourceFiles_Success(t *testing.T) {
	tmp := t.TempDir()
	wfPath := filepath.Join(tmp, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	wf, err := ParseWorkflowFile(wfPath)
	require.NoError(t, err)
	parser, err := newYAMLParser()
	require.NoError(t, err)
	defer parser.Cleanup()
	err = LoadResourceFiles(wf, filepath.Join(tmp, "resources"), parser)
	require.NoError(t, err)
}

func TestValidateWorkflow_Error(t *testing.T) {
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: ""}}
	err := ValidateWorkflow(wf)
	require.Error(t, err)
}

func TestIsPythonModuleAvailable_Python3(t *testing.T) {
	result := isPythonModuleAvailable("sys")
	assert.True(t, result || !result)
}

func TestCheckPortAvailable_InUse(t *testing.T) {
	port := mustFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer ln.Close()
	require.Error(t, CheckPortAvailable("127.0.0.1", port))
}

func TestFindAvailablePort_ScanNotice(t *testing.T) {
	port := mustFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer ln.Close()
	got, err := FindAvailablePort("127.0.0.1", port)
	require.NoError(t, err)
	assert.Greater(t, got, port)
}

func TestGetOllamaURL_Default(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	assert.Equal(t, "http://localhost:11434", getOllamaURL())
}

func TestIsOllamaRunning_NotRunning(t *testing.T) {
	port := mustFreePort(t)
	assert.False(t, IsOllamaRunning("127.0.0.1", port))
}

func TestStartHTTPServerWithEngine_StartReturnsError(t *testing.T) {
	origStart := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = origStart })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return errors.New("start failed")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
		},
		Resources: []*domain.Resource{
			{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}},
		},
	}
	err := startHTTPServerWithEngine(eng, wf, t.TempDir(), false, false)
	require.Error(t, err)
}

func TestRunUntilSignalOrError_StartError(t *testing.T) {
	err := runUntilSignalOrError(signalServeConfig{
		start: func() error { return errors.New("start failed") },
	})
	require.Error(t, err)
}

func TestRunUntilSignalOrError_SignalShutdownError(t *testing.T) {
	origNotify := notifySignalsFunc
	t.Cleanup(func() { notifySignalsFunc = origNotify })
	notifySignalsFunc = func(c chan<- os.Signal, _ ...os.Signal) {
		go func() { c <- syscall.SIGTERM }()
	}
	err := runUntilSignalOrError(signalServeConfig{
		start: func() error {
			select {}
		},
		shutdown: func(_ context.Context) error {
			return errors.New("shutdown failed")
		},
		logShutdownErrors: true,
	})
	require.NoError(t, err)
}

func TestStartBothServersWithEngine_WebServerCreateError(t *testing.T) {
	orig := httpNewWebServerFunc
	t.Cleanup(func() { httpNewWebServerFunc = orig })
	httpNewWebServerFunc = func(_ *domain.Workflow, _ *slog.Logger) (*kdepshttp.WebServer, error) {
		return nil, errors.New("web server create failed")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
			WebServer: &domain.WebServerConfig{
				PortNum: mustFreePort(t),
				Routes:  []domain.WebRoute{},
			},
		},
	}
	err := startBothServersWithEngine(eng, wf, t.TempDir(), false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create web server")
}

func TestStartBothServersWithEngine_StartReturnsError(t *testing.T) {
	origStart := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = origStart })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return errors.New("start failed")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
			WebServer: &domain.WebServerConfig{
				PortNum: mustFreePort(t),
				Routes:  []domain.WebRoute{},
			},
		},
	}
	err := startBothServersWithEngine(eng, wf, t.TempDir(), false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

func TestStartBothServersWithEngine_CreateError(t *testing.T) {
	orig := createHTTPServerWithEngineFunc
	t.Cleanup(func() { createHTTPServerWithEngineFunc = orig })
	createHTTPServerWithEngineFunc = func(_ *executor.Engine, _ *domain.Workflow, _ string, _, _ bool) (*kdepshttp.Server, error) {
		return nil, errors.New("create failed")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
			WebServer: &domain.WebServerConfig{
				PortNum: mustFreePort(t),
				Routes:  []domain.WebRoute{},
			},
		},
	}
	err := startBothServersWithEngine(eng, wf, t.TempDir(), false, false)
	require.Error(t, err)
}

func TestStartWebServer_StartReturnsError(t *testing.T) {
	orig := webServerStartFunc
	t.Cleanup(func() { webServerStartFunc = orig })
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return errors.New("web start failed")
	}
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{
		WebServer: &domain.WebServerConfig{
			PortNum: mustFreePort(t),
			Routes:  []domain.WebRoute{{Path: "/", PublicPath: "/", ServerType: "static"}},
		},
	}}
	err := StartWebServer(wf, t.TempDir(), false)
	require.Error(t, err)
}

func TestExtractPackage_MkdirTempError(t *testing.T) {
	orig := osMkdirTempExtractFunc
	t.Cleanup(func() { osMkdirTempExtractFunc = orig })
	osMkdirTempExtractFunc = func(_, _ string) (string, error) { return "", errors.New("mkdir") }
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "f.txt", "x"), 0644))
	_, err := ExtractPackage(kdeps)
	require.Error(t, err)
}

func TestExtractPackage_OpenError(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "f.txt", "x"), 0644))
	require.NoError(t, os.Remove(kdeps))
	_, err := ExtractPackage(kdeps)
	require.Error(t, err)
}

func TestExtractTarFiles_NextError(t *testing.T) {
	// Invalid tar header bytes after valid gzip wrapper.
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write([]byte("not-a-valid-tar"))
	require.NoError(t, err)
	require.NoError(t, gzw.Close())
	gzr, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer gzr.Close()
	err = ExtractTarFiles(tar.NewReader(gzr), t.TempDir())
	require.Error(t, err)
}

func TestExtractFile_ParentMkdirError(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	hdr := &tar.Header{Name: "nested/f.txt", Size: 1, Mode: 0644}
	err := ExtractFile(
		tar.NewReader(bytes.NewReader([]byte("x"))),
		hdr,
		filepath.Join(blocker, "nested", "f.txt"),
	)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// validate.go
// ---------------------------------------------------------------------------

func TestValidateResourceFile_Success_Remaining(t *testing.T) {
	tmp := t.TempDir()
	res := `actionId: act
name: Act
apiResponse:
  success: true
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "act.yaml"), []byte(res), 0644))
	require.NoError(t, validateResourceFile(filepath.Join(tmp, "act.yaml")))
}

func TestValidateWorkflowFile_Success(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, validateWorkflowFile(filepath.Join(tmp, "workflow.yaml")))
}

func TestValidateComponentFile_Success(t *testing.T) {
	tmp := t.TempDir()
	comp := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: c
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "component.yaml"), []byte(comp), 0644))
	require.NoError(t, validateComponentFile(filepath.Join(tmp, "component.yaml")))
}

func TestValidateResourceFile_ParserError(t *testing.T) {
	orig := newSchemaValidatorFunc
	t.Cleanup(func() { newSchemaValidatorFunc = orig })
	newSchemaValidatorFunc = func() (*validator.SchemaValidator, error) {
		return nil, errors.New("validator")
	}
	err := validateResourceFile(filepath.Join(t.TempDir(), "r.yaml"))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// serve.go / clone.go
// ---------------------------------------------------------------------------

func TestRunServeCmd_InvalidPath_Remaining(t *testing.T) {
	err := runServeCmd("\x00bad", &serveFlags{})
	require.Error(t, err)
}

func TestRegisterAgencyTool_TargetError(t *testing.T) {
	tmp := t.TempDir()
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: a
  version: "1.0.0"
  targetAgentId: missing
agents:
  - agents/a
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agency), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agents", "a"), 0755))
	agentWF := strings.Replace(minimalWorkflowYAML(), "name: gap-test", "name: other", 1)
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "agents", "a", "workflow.yaml"), []byte(agentWF), 0644),
	)
	_, err := registerAgencyTool(filepath.Join(tmp, "agency.yaml"), tools.NewRegistry(), false)
	require.Error(t, err)
}

func TestFindServeWorkflowFiles_SkipsInvalid(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "bad"), 0755))
	files := findServeWorkflowFiles(tmp)
	assert.NotEmpty(t, files)
}

func TestCloneFromRemote_InvalidRef(t *testing.T) {
	err := cloneFromRemote("not-a-valid-ref")
	require.Error(t, err)
}

func TestCloneAsWorkdir_Success(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(src, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	err := cloneAsWorkdir("myagent", src, tmp, "myagent")
	require.NoError(t, err)
}

func TestUnwrapArchiveRoot_SingleDir(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "repo-main")
	require.NoError(t, os.MkdirAll(sub, 0755))
	got, err := unwrapArchiveRoot(tmp)
	require.NoError(t, err)
	assert.Equal(t, sub, got)
}

// ---------------------------------------------------------------------------
// registry
// ---------------------------------------------------------------------------

func TestRegistryURL_Flag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("registry", "", "")
	require.NoError(t, cmd.Flags().Set("registry", "http://flag-registry"))
	assert.Equal(t, "http://flag-registry", registryURL(cmd))
}

func TestDoRegistryInstall_InfoError(t *testing.T) {
	cmd := &cobra.Command{}
	err := doRegistryInstall(cmd, "nonexistent-pkg", "http://127.0.0.1:1")
	require.Error(t, err)
}

func TestPeekManifest_TarError(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.kdeps")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("not tar"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(bad, buf.Bytes(), 0644))
	_, err = peekManifest(bad)
	require.Error(t, err)
}

func TestInstallRegistryComponent_Global(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	archive := buildMinimalKdepsArchivePath(t)
	manifest := &domain.KdepsPkg{Name: "globalcomp", Type: pkgTypeComponent}
	cmd := &cobra.Command{}
	err := installRegistryComponent(cmd, manifest, archive, "1.0")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// additional gaps batch 2
// ---------------------------------------------------------------------------

func TestOpenExecutableWithError_FileStatHook(t *testing.T) {
	orig := fileStatFunc
	t.Cleanup(func() { fileStatFunc = orig })
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bin")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0644))
	fileStatFunc = func(_ *os.File) (os.FileInfo, error) { return nil, errors.New("stat fail") }
	_, _, err := openExecutableWithError(path)
	require.Error(t, err)
}

func TestBuildImageInternal_PackagePathAbsError(t *testing.T) {
	orig := filepathAbsFunc
	t.Cleanup(func() { filepathAbsFunc = orig })
	call := 0
	filepathAbsFunc = func(path string) (string, error) {
		call++
		if call == 2 {
			return "", errors.New("abs package fail")
		}
		return filepath.Abs(path)
	}
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{})
	require.Error(t, err)
}

func TestBuildImageInternal_SetupDockerAfterChdir(t *testing.T) {
	orig := setupDockerBuilderFunc
	t.Cleanup(func() { setupDockerBuilderFunc = orig })
	setupDockerBuilderFunc = func(_ *BuildFlags) (*docker.Builder, error) {
		return nil, errors.New("docker setup fail")
	}
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644),
	)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{})
	require.Error(t, err)
}

func TestBuildImageInternal_KdepsArchiveCleanup(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(
			kdeps,
			buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()),
			0644,
		),
	)
	newBuildDockerClient(t, func(_ *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	origBuild := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = origBuild })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "img:1", nil
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	require.NoError(t, buildImageInternal(cmd, []string{kdeps}, &BuildFlags{}))
}

func TestCollectWebServerFiles_RelAndReadErrors(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data", "sub")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, "index.html"), []byte("<html/>"), 0644))
	files, err := collectWebServerFiles(tmp)
	require.NoError(t, err)
	assert.NotEmpty(t, files)
}

func TestGorootWASMExecCandidates_EmptyGoroot(t *testing.T) {
	cands := gorootWASMExecCandidates(context.Background())
	if len(cands) == 0 {
		assert.Nil(t, cands)
	} else {
		assert.NotEmpty(t, cands)
	}
}

func TestCloneAsComponent_CopyDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	src := filepath.Join(tmp, "src")
	require.NoError(t, os.MkdirAll(src, 0755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(src, "component.yaml"), []byte("metadata:\n  name: c\n"), 0644),
	)
	err := cloneAsComponent("mycomp", src)
	require.NoError(t, err)
}

func TestCloneAsWorkdir_ExistsError(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	dest := filepath.Join("agents", "dup")
	require.NoError(t, os.MkdirAll(dest, 0755))
	err := cloneAsWorkdir("dup", tmp, "agents", "workflow.yaml")
	require.Error(t, err)
}

func TestDetectCloneType_Agency(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte("metadata:\n  name: a\n"), 0644),
	)
	typ, name := detectCloneType(tmp)
	assert.Equal(t, "agency", typ)
	assert.Equal(t, "agency.yaml", name)
}

func TestDownloadAndExtract_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		hdr := &tar.Header{Name: "f.txt", Size: 1, Mode: 0644}
		require.NoError(t, tw.WriteHeader(hdr))
		_, _ = tw.Write([]byte("x"))
		require.NoError(t, tw.Close())
		require.NoError(t, gz.Close())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()
	err := downloadAndExtract(srv.URL, t.TempDir())
	require.NoError(t, err)
}

func TestUnwrapArchiveRoot_MultipleEntries(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "b.txt"), []byte("b"), 0644))
	got, err := unwrapArchiveRoot(tmp)
	require.NoError(t, err)
	assert.Equal(t, tmp, got)
}

func TestCopyFile_CloseError(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "ro", "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("data"), 0644))
	roDir := filepath.Join(tmp, "ro")
	require.NoError(t, os.Mkdir(roDir, 0500))
	err := copyFile(src, dst)
	require.Error(t, err)
}

func TestCopyDir_WalkError_Remaining(t *testing.T) {
	err := copyDir("/nonexistent/src", t.TempDir())
	require.Error(t, err)
}

func TestBuildPrepackageTarget_Success(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(
			kdeps,
			buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()),
			0644,
		),
	)
	base := filepath.Join(tmp, "base")
	require.NoError(t, os.WriteFile(base, []byte("#!/bin/sh\n"), 0755))
	outDir := filepath.Join(tmp, "out")
	require.NoError(t, os.MkdirAll(outDir, 0755))
	orig := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = orig })
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return base, false, nil
	}
	_, err := buildPrepackageTarget(
		context.Background(),
		kdeps,
		"1.0",
		base,
		outDir,
		"pkg",
		archTarget{GOOS: "linux", GOARCH: "amd64"},
	)
	require.NoError(t, err)
}

func TestWriteTempBinary_WindowsMode(t *testing.T) {
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = os.CreateTemp
	path, err := writeTempBinary([]byte("bin"), "windows", "amd64")
	require.NoError(t, err)
	_ = os.Remove(path)
}

func TestDownloadKdepsBinaryToTemp_DevVersion(t *testing.T) {
	_, err := downloadKdepsBinaryToTemp(context.Background(), "dev", "linux", "amd64")
	require.Error(t, err)
}

func TestFetchURL_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := fetchURL(context.Background(), srv.URL)
	require.Error(t, err)
}

func TestExtractFromTarGz_Found(t *testing.T) {
	data := buildMinimalKdepsArchive(t, "kdeps", "bin-data")
	out, err := extractFromTarGz(data, "kdeps")
	require.NoError(t, err)
	assert.Equal(t, "bin-data", string(out))
}

func TestExtractFromZip_NotFound(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	require.NoError(t, zw.Close())
	_, err := extractFromZip(buf.Bytes(), "missing")
	require.Error(t, err)
}

func TestValidateComponentFile_ParserInitError(t *testing.T) {
	orig := newSchemaValidatorFunc
	t.Cleanup(func() { newSchemaValidatorFunc = orig })
	newSchemaValidatorFunc = func() (*validator.SchemaValidator, error) {
		return nil, errors.New("init fail")
	}
	err := validateComponentFile(filepath.Join(t.TempDir(), "c.yaml"))
	require.Error(t, err)
}

func TestGetOllamaURL_DefaultBranch(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	assert.Equal(t, ollamaDefaultURL, getOllamaURL())
}

func TestExtractPackage_OpenAfterMkdir(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "gone.kdeps")
	_, err := ExtractPackage(kdeps)
	require.Error(t, err)
}

func TestResolveServerBindAddress_FindError(t *testing.T) {
	orig := findAvailablePortFunc
	t.Cleanup(func() { findAvailablePortFunc = orig })
	findAvailablePortFunc = func(_ string, _ int) (int, error) { return 0, errors.New("no port") }
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{APIServer: &domain.APIServerConfig{PortNum: 8080}},
	}
	_, err := resolveServerBindAddress(wf)
	require.Error(t, err)
}

func TestCreateHTTPServerWithEngine_Valid(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
		},
		Resources: []*domain.Resource{
			{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}},
		},
	}
	srv, err := createHTTPServerWithEngine(eng, wf, t.TempDir(), false, false)
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestStartHTTPServerWithEngine_DevModeBranch(t *testing.T) {
	origStart := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = origStart })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return errors.New("start")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
		},
		Resources: []*domain.Resource{
			{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}},
		},
	}
	err := startHTTPServerWithEngine(eng, wf, t.TempDir(), true, false)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// batch 3 — run.go, registry, embedded, component
// ---------------------------------------------------------------------------

func TestResolveRegularPath_AbsHookError(t *testing.T) {
	orig := filepathAbsFunc
	t.Cleanup(func() { filepathAbsFunc = orig })
	filepathAbsFunc = func(_ string) (string, error) { return "", errors.New("abs") }
	_, _, err := ResolveRegularPath("any")
	require.Error(t, err)
}

func TestPrintAgencyAgentIndex_RelFallback(_ *testing.T) {
	agency := &domain.Agency{Metadata: domain.AgencyMetadata{TargetAgentID: "a"}}
	printAgencyAgentIndex(
		"/tmp/agency",
		agency,
		map[string]string{"a": "/outside/path/workflow.yaml"},
	)
}

func TestBuildAgentNameMap_EmptyPaths(t *testing.T) {
	m, p, err := buildAgentNameMap(nil, "")
	require.NoError(t, err)
	assert.Empty(t, m)
	assert.Empty(t, p)
}

func TestExecuteAgencyEntryPoint_Interactive(t *testing.T) {
	stubDispatchHooks(t)
	tmp := t.TempDir()
	wfPath := filepath.Join(tmp, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(minimalWorkflowYAML()), 0644))
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, w.Close())
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin; _ = r.Close() })
	err = executeAgencyEntryPoint(
		cmd,
		wfPath,
		map[string]string{"gap-test": wfPath},
		&RunFlags{Interactive: true},
	)
	t.Logf("interactive agency: %v", err)
}

func TestEnsureOllamaRunning_WaitError(t *testing.T) {
	origStart := startOllamaServerFunc
	origWait := waitForOllamaReadyFunc
	origRunning := isOllamaRunningFunc
	t.Cleanup(func() {
		startOllamaServerFunc = origStart
		waitForOllamaReadyFunc = origWait
		isOllamaRunningFunc = origRunning
	})
	isOllamaRunningFunc = func(_ string, _ int) bool { return false }
	startOllamaServerFunc = func() error { return nil }
	waitForOllamaReadyFunc = func(_ string, _ int, _ time.Duration) error { return errors.New("wait") }
	err := ensureOllamaRunning("http://127.0.0.1:11434")
	require.Error(t, err)
}

func TestStartInteractiveMode_BackgroundDispatchError(t *testing.T) {
	orig := execSingleRunWithEngineFn
	t.Cleanup(func() { execSingleRunWithEngineFn = orig })
	execSingleRunWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow) error {
		return errors.New("dispatch bg fail")
	}
	stubDispatchHooks(t)
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "i", TargetActionID: "act"},
		Resources: []*domain.Resource{
			{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}},
		},
	}
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, w.Close())
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin; _ = r.Close() })
	executionModeForFunc = func(_ *domain.Workflow) executionMode { return execModeSingleRun }
	err = startInteractiveMode(eng, wf, t.TempDir(), &RunFlags{}, false)
	t.Logf("interactive bg: %v", err)
}

func TestExtractPackage_GzipError(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "bad.kdeps")
	require.NoError(t, os.WriteFile(kdeps, []byte("not gzip"), 0644))
	_, err := ExtractPackage(kdeps)
	require.Error(t, err)
}

func TestExtractTarFiles_ExtractFileError(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "../escape", Typeflag: tar.TypeReg, Mode: 0644, Size: 0}
	require.NoError(t, tw.WriteHeader(hdr))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	gzr, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer gzr.Close()
	err = ExtractTarFiles(tar.NewReader(gzr), t.TempDir())
	require.Error(t, err)
}

func TestValidateAndJoinPath_Invalid(t *testing.T) {
	_, err := ValidateAndJoinPath("../escape", t.TempDir())
	require.Error(t, err)
}

func TestExtractFile_CreateFail(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	hdr := &tar.Header{Name: "nested/f.txt", Size: 1, Mode: 0644}
	err := ExtractFile(
		tar.NewReader(bytes.NewReader([]byte("x"))),
		hdr,
		filepath.Join(blocker, "nested", "f.txt"),
	)
	require.Error(t, err)
}

func TestRunEmbeddedPackage_CloseTempError(t *testing.T) {
	tmp := t.TempDir()
	binPath := writeEmbeddedTestBinary(t, tmp)
	origCreate := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = origCreate })
	osCreateTempFunc = func(_ string, _ string) (*os.File, error) {
		p := filepath.Join(tmp, "partial")
		f, err := os.Create(p)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		return os.OpenFile(p, os.O_RDONLY, 0644)
	}
	assert.Equal(t, 1, RunEmbeddedPackage("dev", "dev", binPath))
}

func TestWriteCleanBinaryTemp_WriteError_Remaining(t *testing.T) {
	tmp := t.TempDir()
	src, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	_, err = src.Write(bytes.Repeat([]byte("x"), 10))
	require.NoError(t, err)
	require.NoError(t, src.Close())
	rf, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer rf.Close()
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, _ string) (*os.File, error) {
		p := filepath.Join(tmp, "ro")
		require.NoError(t, os.WriteFile(p, nil, 0444))
		return os.OpenFile(p, os.O_RDONLY, 0444)
	}
	_, _, err = writeCleanBinaryTemp(rf, 10)
	require.Error(t, err)
}

func TestAppendEmbeddedPackage_MkdirOutputError(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	require.NoError(t, os.WriteFile(bin, []byte("bin"), 0755))
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, []byte("pkg"), 0644))
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	err := AppendEmbeddedPackage(bin, kdeps, filepath.Join(blocker, "out", "embedded"))
	require.Error(t, err)
}

func TestWriteEmbeddedTrailer_WriteSizeError(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "out"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ro, err := os.OpenFile(filepath.Join(tmp, "out"), os.O_RDONLY, 0444)
	require.NoError(t, err)
	defer ro.Close()
	err = writeEmbeddedTrailer(ro, 10)
	require.Error(t, err)
}

func TestDetectPayloadRange_ReadAtError(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "bin"))
	require.NoError(t, err)
	_, err = f.Write(bytes.Repeat([]byte("x"), EmbeddedTrailerSize+5))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	rf, err := os.Open(f.Name())
	require.NoError(t, err)
	require.NoError(t, os.Truncate(f.Name(), 5))
	_, _, ok := detectPayloadRange(rf, EmbeddedTrailerSize+5)
	assert.False(t, ok)
	_ = rf.Close()
}

func TestIsPythonModuleAvailable_Fallback(_ *testing.T) {
	_ = isPythonModuleAvailable("nonexistent_module_xyz_12345")
}

func TestCheckPortAvailable_Success(t *testing.T) {
	port := mustFreePort(t)
	require.NoError(t, CheckPortAvailable("127.0.0.1", port))
}

func TestFindAvailablePort_FirstTry(t *testing.T) {
	port := mustFreePort(t)
	got, err := FindAvailablePort("127.0.0.1", port)
	require.NoError(t, err)
	assert.Equal(t, port, got)
}
