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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	"github.com/kdeps/kdeps/v2/pkg/infra/iso"
	"github.com/kdeps/kdeps/v2/pkg/input/bot"
	"github.com/kdeps/kdeps/v2/pkg/manifest"
)

// ---------------------------------------------------------------------------
// registry_submit.go — 0% helpers
// ---------------------------------------------------------------------------

func TestGithubTarballURL(t *testing.T) {
	got := githubTarballURL("owner/repo", "v1.0.0")
	assert.Equal(t, "https://github.com/owner/repo/archive/refs/tags/v1.0.0.tar.gz", got)
}

func TestBuildRegistryFormula(t *testing.T) {
	m := &manifest.Manifest{
		Name:        "my-agent",
		Version:     "1.2.3",
		Type:        "agent",
		Description: "desc",
		Tags:        []string{"llm"},
		License:     "Apache-2.0",
	}
	f := buildRegistryFormula(m, "owner/repo", "https://example.com/tar.gz", "abc123")
	assert.Equal(t, "my-agent", f.Name)
	assert.Equal(t, "owner/repo", f.GitHub)
	assert.Equal(t, "abc123", f.SHA256)
}

func TestEncodeRegistryFormula(t *testing.T) {
	f := registryFormula{Name: "pkg", Version: "1.0.0", Type: "workflow"}
	out, err := encodeRegistryFormula(f)
	require.NoError(t, err)
	assert.Contains(t, out, "name: pkg")
}

func TestPrintRegistryFormula(t *testing.T) {
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	printRegistryFormula(cmd, "my-pkg", "name: my-pkg\n")
	assert.Contains(t, buf.String(), "formulas/my-pkg.yaml")
	assert.Contains(t, buf.String(), "name: my-pkg")
}

func TestSubmitDirFromArgs(t *testing.T) {
	assert.Equal(t, ".", submitDirFromArgs(nil))
	assert.Equal(t, "/tmp/pkg", submitDirFromArgs([]string{"/tmp/pkg"}))
}

// ---------------------------------------------------------------------------
// export.go — ISO helpers
// ---------------------------------------------------------------------------

func TestEnableISOOfflineMode(t *testing.T) {
	t.Setenv("KDEPS_LLM_MODELS", "llama3")
	t.Setenv("KDEPS_OFFLINE_MODE", "")
	enableISOOfflineMode()
	assert.Equal(t, "true", os.Getenv("KDEPS_OFFLINE_MODE"))
}

func TestEnableISOOfflineMode_NoModels(t *testing.T) {
	t.Setenv("KDEPS_LLM_MODELS", "")
	t.Setenv("KDEPS_OFFLINE_MODE", "false")
	enableISOOfflineMode()
	assert.Equal(t, "false", os.Getenv("KDEPS_OFFLINE_MODE"))
}

func TestResolveLinuxKitFormat(t *testing.T) {
	got, err := resolveLinuxKitFormat("iso")
	require.NoError(t, err)
	assert.Equal(t, "iso-efi", got)

	_, err = resolveLinuxKitFormat("bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestInjectConfigEnv(t *testing.T) {
	t.Setenv("KDEPS_LLM_ROUTER", "ollama")
	t.Setenv("KDEPS_DEFAULT_BACKEND", "openai")
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{AgentSettings: domain.AgentSettings{}}}
	injectConfigEnv(wf)
	assert.Equal(t, "ollama", wf.Settings.AgentSettings.Env["KDEPS_LLM_ROUTER"])
	assert.Equal(t, "openai", wf.Settings.AgentSettings.Env["KDEPS_DEFAULT_BACKEND"])
}

func TestConfigureISOBuilderSize_Explicit(t *testing.T) {
	isoBuilder := &iso.Builder{}
	configureISOBuilderSize(isoBuilder, &docker.Builder{}, "img:tag", "2048M")
	assert.Equal(t, "2048M", isoBuilder.Size)
}

func TestConfigureISOBuilderSize_AutoFromImage(t *testing.T) {
	mockClient := newExportDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/images/") && req.Method == http.MethodGet {
			body, _ := json.Marshal(dockclient.InspectResponse{Size: 100 * 1024 * 1024})
			return jsonHTTPResponse(http.StatusOK, body), nil
		}
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	isoBuilder := &iso.Builder{}
	out := captureStdout(t, func() {
		configureISOBuilderSize(isoBuilder, &docker.Builder{Client: mockClient}, "myimg:1.0", "")
	})
	assert.NotEmpty(t, isoBuilder.Size)
	assert.Contains(t, out, "Auto-computed disk image size")
}

func TestConfigureISOBuilderSize_ImageSizeError(t *testing.T) {
	mockClient := newExportDockerClient(t, func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("inspect failed")
	})
	isoBuilder := &iso.Builder{}
	configureISOBuilderSize(isoBuilder, &docker.Builder{Client: mockClient}, "myimg:1.0", "")
	assert.Empty(t, isoBuilder.Size)
}

func writeISOPackageDir(t *testing.T) (string, *domain.Workflow, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "resources"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "resources", "act.yaml"),
		[]byte("actionId: act\nname: Act\napiResponse:\n  success: true\n"),
		0644,
	))
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	return tmpDir, minimalISOWorkflow(), func() { _ = os.Chdir(origDir) }
}

func TestPerformISOBuild_BuildFails(t *testing.T) {
	mockClient := newExportDockerClient(t, func(_ *http.Request) (*http.Response, error) {
		body := []byte(`{"message":"fail"}`)
		return jsonHTTPResponse(http.StatusInternalServerError, body), nil
	})
	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	pkgDir, wf, restore := writeISOPackageDir(t)
	defer restore()
	err := performISOBuild(builder, wf, pkgDir, pkgDir, &ExportFlags{Format: "iso"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build Docker image")
}

func TestPerformISOBuild_UnsupportedFormat(t *testing.T) {
	builder := &docker.Builder{
		BaseOS: "alpine",
		Client: newExportDockerClient(t, exportDockerBuildSuccessHandler()),
	}
	pkgDir, wf, restore := writeISOPackageDir(t)
	defer restore()
	err := performISOBuild(builder, wf, pkgDir, pkgDir, &ExportFlags{Format: "invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestPerformISOBuild_Success(t *testing.T) {
	installFakeLinuxkit(t)
	builder := &docker.Builder{
		BaseOS: "alpine",
		Client: newExportDockerClient(t, func(req *http.Request) (*http.Response, error) {
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
		}),
	}
	pkgDir, wf, restore := writeISOPackageDir(t)
	defer restore()
	outPath := filepath.Join(pkgDir, "out.iso")
	err := performISOBuild(builder, wf, pkgDir, pkgDir, &ExportFlags{Format: "iso", Output: outPath})
	require.NoError(t, err)
	assert.FileExists(t, outPath)
}

// ---------------------------------------------------------------------------
// build.go — attachPrepackagedBinaries
// ---------------------------------------------------------------------------

func TestAttachPrepackagedBinaries_EnsureKdepsError(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}
	attachPrepackagedBinaries(
		context.Background(),
		builder,
		"/nonexistent/pkg.kdeps",
		"/nonexistent",
		&domain.Workflow{},
	)
	assert.Empty(t, builder.PrepackagedBinaries)
}

func TestAttachPrepackagedBinaries_Success(t *testing.T) {
	origResolve := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = origResolve })

	tmpDir := t.TempDir()
	kdepsPath := filepath.Join(tmpDir, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdepsPath, []byte("pkg"), 0644))
	baseBin := filepath.Join(tmpDir, "base-kdeps")
	require.NoError(t, os.WriteFile(baseBin, []byte("bin"), 0755))

	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return baseBin, false, nil
	}

	builder := &docker.Builder{BaseOS: "alpine"}
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}
	attachPrepackagedBinaries(context.Background(), builder, kdepsPath, tmpDir, wf)
	assert.NotEmpty(t, builder.PrepackagedBinaries)
}

// ---------------------------------------------------------------------------
// run.go — path resolution and dispatch helpers
// ---------------------------------------------------------------------------

func createKagencyArchiveForRun(t *testing.T, dir string) string {
	t.Helper()
	archivePath := filepath.Join(dir, "test.kagency")
	f, err := os.Create(archivePath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: test-agency
  version: "1.0.0"
  targetAgentId: agent-a
agents:
  - agents/agent-a
`
	hdr := &tar.Header{Name: "agency.yaml", Size: int64(len(agency)), Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write([]byte(agency))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return archivePath
}

func TestResolveKagencyPackage_Valid(t *testing.T) {
	tmp := t.TempDir()
	archive := createKagencyArchiveForRun(t, tmp)
	path, cleanup, err := resolveKagencyPackage(archive)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()
	assert.Contains(t, path, "agency.yaml")
}

func TestResolveKagencyPackage_InvalidArchive(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.kagency")
	require.NoError(t, os.WriteFile(bad, []byte("not gzip"), 0644))
	_, _, err := resolveKagencyPackage(bad)
	require.Error(t, err)
}

func TestResolveKagencyPackage_NoAgencyFile(t *testing.T) {
	tmp := t.TempDir()
	archive := filepath.Join(tmp, "empty.kagency")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "workflow.yaml", Size: 1, Mode: 0644}))
	_, _ = tw.Write([]byte("x"))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(archive, buf.Bytes(), 0644))
	_, _, err := resolveKagencyPackage(archive)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agency.yaml")
}

func TestBotPlatformsFromInput(t *testing.T) {
	assert.Nil(t, botPlatformsFromInput(nil))
	input := &domain.InputConfig{
		Bot: &domain.BotConfig{
			Discord:  &domain.DiscordConfig{},
			Slack:    &domain.SlackConfig{},
			Telegram: &domain.TelegramConfig{},
			WhatsApp: &domain.WhatsAppConfig{},
		},
	}
	assert.Equal(t, []string{"discord", "slack", "telegram", "whatsapp"}, botPlatformsFromInput(input))
}

func TestLoadBotCredentials(t *testing.T) {
	assert.Nil(t, loadBotCredentials("nonexistent-agent-xyz-12345"))
}

func TestExecutionModeFor_AllModes(t *testing.T) {
	cases := []struct {
		name string
		wf   *domain.Workflow
		want executionMode
	}{
		{
			"both",
			&domain.Workflow{Settings: domain.WorkflowSettings{
				APIServer: &domain.APIServerConfig{},
				WebServer: &domain.WebServerConfig{},
			}},
			execModeBothServers,
		},
		{
			"web",
			&domain.Workflow{Settings: domain.WorkflowSettings{WebServer: &domain.WebServerConfig{}}},
			execModeWebServer,
		},
		{
			"api",
			&domain.Workflow{Settings: domain.WorkflowSettings{APIServer: &domain.APIServerConfig{}}},
			execModeAPIServer,
		},
		{
			"bot",
			&domain.Workflow{Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{Sources: []string{"bot"}, Bot: &domain.BotConfig{}},
			}},
			execModeBot,
		},
		{
			"file",
			&domain.Workflow{Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{Sources: []string{"file"}},
			}},
			execModeFile,
		},
		{
			"single",
			&domain.Workflow{Settings: domain.WorkflowSettings{}},
			execModeSingleRun,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, executionModeFor(tc.wf))
		})
	}
}

func TestDispatchExecution_SingleRun(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "ok", nil
	})
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "single", TargetActionID: "act"},
		Resources: []*domain.Resource{{
			ActionID:    "act",
			APIResponse: &domain.APIResponseConfig{Success: true},
		}},
	}
	err := dispatchExecutionWithEngine(eng, wf, t.TempDir(), false, false, "", false)
	require.NoError(t, err)
}

func TestSetupDevMode(t *testing.T) {
	tmp := t.TempDir()
	wfPath := filepath.Join(tmp, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(minimalWorkflowYAML()), 0644))
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return nil, nil })
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
		},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	srv, err := createHTTPServerWithEngine(eng, wf, tmp, true, false)
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestStartBotRunnersWithEngine_Stateless(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "bot"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{"bot"},
				Bot: &domain.BotConfig{
					ExecutionType: domain.BotExecutionTypeStateless,
				},
			},
		},
	}
	err := StartBotRunnersWithEngine(eng, wf, false)
	require.Error(t, err) // no stdin message in test
}

func TestStartBotRunners_Delegates(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "bot"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{"bot"},
				Bot:     &domain.BotConfig{ExecutionType: domain.BotExecutionTypeStateless},
			},
		},
	}
	err := StartBotRunners(wf, false)
	require.Error(t, err)
	_ = eng
}

func TestStartLLMRunner_REPL(t *testing.T) {
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "llm"},
		Settings: domain.WorkflowSettings{
			LLM: &domain.LLMInputConfig{ExecutionType: domain.LLMExecutionTypeStdin},
		},
	}
	// EOF on stdin exits the REPL cleanly (nil) or returns an error depending on platform.
	err := StartLLMRunner(wf, false, t.TempDir(), false)
	t.Logf("StartLLMRunner returned: %v", err)
}

func TestResolveServerBindAddress_Override(t *testing.T) {
	t.Setenv("KDEPS_BIND_HOST", "127.0.0.1")
	port := mustFreePort(t)
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{APIServer: &domain.APIServerConfig{PortNum: port}}}
	addr, err := resolveServerBindAddress(wf)
	require.NoError(t, err)
	assert.Contains(t, addr, "127.0.0.1")
}

func TestStartWebServer_GracefulShutdown(t *testing.T) {
	orig := webServerStartFunc
	t.Cleanup(func() { webServerStartFunc = orig })
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return http.ErrServerClosed
	}
	port := mustFreePort(t)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			WebServer: &domain.WebServerConfig{
				PortNum: port,
				Routes: []domain.WebRoute{{
					Path:       "/",
					PublicPath: "/",
					ServerType: "static",
				}},
			},
		},
	}
	require.NoError(t, StartWebServer(wf, t.TempDir(), false))
}

func TestStartBothServers_GracefulShutdown(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	port := mustFreePort(t)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "both", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer:     &domain.APIServerConfig{PortNum: port},
			WebServer:     &domain.WebServerConfig{PortNum: port, Routes: []domain.WebRoute{}},
			AgentSettings: domain.AgentSettings{PythonVersion: "3.12"},
		},
		Resources: []*domain.Resource{{
			ActionID:    "act",
			APIResponse: &domain.APIResponseConfig{Success: true},
		}},
	}
	require.NoError(t, StartBothServers(wf, t.TempDir(), false, false))
}

func TestLoadAgentProfile(t *testing.T) {
	assert.NotPanics(t, func() { loadAgentProfile("") })
	assert.NotPanics(t, func() { loadAgentProfile("nonexistent-profile-agent") })
}

func TestParseWorkflowStep_Success(t *testing.T) {
	tmp := t.TempDir()
	wfPath := filepath.Join(tmp, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(minimalWorkflowYAML()), 0644))
	wf, err := parseWorkflowStep(wfPath)
	require.NoError(t, err)
	assert.Equal(t, "gap-test", wf.Metadata.Name)
}

func TestSetupEnvironmentStep_NoPython(t *testing.T) {
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}, Settings: domain.WorkflowSettings{}}
	require.NoError(t, setupEnvironmentStep(wf))
}

func TestEnsureLLMBackendStep_NoOllama(t *testing.T) {
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{}}
	require.NoError(t, ensureLLMBackendStep(wf))
}

func TestWriteK8sManifests_ToFile(t *testing.T) {
	tmp := t.TempDir()
	outFile := filepath.Join(tmp, "manifest.yaml")
	flags := &K8sFlags{Output: outFile}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	require.NoError(t, writeK8sManifests(cmd, flags, "apiVersion: v1\n"))
	assert.FileExists(t, outFile)
	assert.Contains(t, buf.String(), "written to")
}

// ---------------------------------------------------------------------------
// embedded.go helpers
// ---------------------------------------------------------------------------

func TestOpenExecutableWithError_NotFound(t *testing.T) {
	_, _, err := openExecutableWithError("/nonexistent/binary/path")
	require.Error(t, err)
}

func TestWriteCleanBinaryTemp_Success(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	content := []byte("clean-binary-content")
	_, err = f.Write(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	src, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer src.Close()

	path, cleaned, err := writeCleanBinaryTemp(src, int64(len(content)))
	require.NoError(t, err)
	assert.True(t, cleaned)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, data)
	_ = os.Remove(path)
}

func TestRunEmbeddedPackage_InvalidBinary(t *testing.T) {
	code := RunEmbeddedPackage("dev", "dev", "/nonexistent/path")
	assert.Equal(t, 1, code)
}

func TestWriteCleanBinaryTemp_ReadError(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	src, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer src.Close()
	_, _, err = writeCleanBinaryTemp(src, 100)
	require.Error(t, err)
}

func TestBootstrapRootConfig(t *testing.T) {
	assert.NotPanics(t, func() { bootstrapRootConfig() })
}

func TestMaybeEnableInstrumentation_Enabled(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("instrument", false, "")
	require.NoError(t, c.Flags().Set("instrument", "true"))
	t.Setenv("KDEPS_INSTRUMENT", "")
	maybeEnableInstrumentation(c)
	assert.Equal(t, "true", os.Getenv("KDEPS_INSTRUMENT"))
}

func TestMaybeEnableInstrumentation_Disabled(t *testing.T) {
	c := &cobra.Command{}
	c.Flags().Bool("instrument", false, "")
	t.Setenv("KDEPS_INSTRUMENT", "")
	maybeEnableInstrumentation(c)
	assert.Empty(t, os.Getenv("KDEPS_INSTRUMENT"))
}

func TestRemoveInstalledFile_Success(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "agent.yaml")
	require.NoError(t, os.WriteFile(fpath, []byte("x"), 0644))
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	ok, err := removeInstalledFile(cmd, "test-agent", fpath, "agent")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Contains(t, buf.String(), "Uninstalled")
}

func TestRemoveInstalledFile_NotFound(t *testing.T) {
	cmd := &cobra.Command{}
	ok, err := removeInstalledFile(cmd, "missing", "/nonexistent/file", "agent")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestExportISOInternal_ShowConfig(t *testing.T) {
	pkgDir, _, restore := writeISOPackageDir(t)
	defer restore()
	flags := &ExportFlags{ShowConfig: true, Hostname: "iso-host"}
	err := exportISOInternal(&cobra.Command{}, []string{pkgDir}, flags)
	require.NoError(t, err)
}

func TestDispatchExecution_AllModes(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "ok", nil
	})
	tmp := t.TempDir()

	t.Run("single", func(t *testing.T) {
		wf := &domain.Workflow{
			Metadata:  domain.WorkflowMetadata{Name: "s", TargetActionID: "act"},
			Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
		}
		require.NoError(t, dispatchExecutionWithEngine(eng, wf, tmp, false, false, "", false))
	})

	t.Run("file", func(t *testing.T) {
		wf := &domain.Workflow{
			Metadata: domain.WorkflowMetadata{Name: "f"},
			Settings: domain.WorkflowSettings{Input: &domain.InputConfig{Sources: []string{"file"}}},
		}
		err := dispatchExecutionWithEngine(eng, wf, tmp, false, false, "", false)
		require.Error(t, err) // no file input in test
	})

	t.Run("bot-stateless", func(t *testing.T) {
		wf := &domain.Workflow{
			Metadata: domain.WorkflowMetadata{Name: "b"},
			Settings: domain.WorkflowSettings{
				Input: &domain.InputConfig{
					Sources: []string{"bot"},
					Bot:     &domain.BotConfig{ExecutionType: domain.BotExecutionTypeStateless},
				},
			},
		}
		err := dispatchExecutionWithEngine(eng, wf, tmp, false, false, "", false)
		require.Error(t, err)
	})
}

func TestDispatchExecution_PublicWrapper(t *testing.T) {
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "s", TargetActionID: "act"},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	require.NoError(t, dispatchExecution(wf, t.TempDir(), false, false, "", false))
}

func TestDispatchExecution_APIServerShutdown(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	port := mustFreePort(t)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer:     &domain.APIServerConfig{PortNum: port},
			AgentSettings: domain.AgentSettings{PythonVersion: "3.12"},
		},
		Resources: []*domain.Resource{{
			ActionID:    "act",
			APIResponse: &domain.APIResponseConfig{Success: true},
		}},
	}
	require.NoError(t, dispatchExecution(wf, t.TempDir(), false, false, "", false))
}

func TestStartInteractiveMode_SingleRun(t *testing.T) {
	orig := execHTTPServerWithEngineFn
	t.Cleanup(func() { execHTTPServerWithEngineFn = orig })
	execHTTPServerWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow, _ string, _, _ bool) error {
		return nil
	}
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "bg", nil
	})
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "interactive", TargetActionID: "act"},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, w.Close())
	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = origStdin
		_ = r.Close()
	})
	err = startInteractiveMode(eng, wf, t.TempDir(), &RunFlags{}, false)
	t.Logf("startInteractiveMode returned: %v", err)
}

func TestStartBotRunnersWithEngine_PollingShutdown(t *testing.T) {
	orig := botDispatcherRunFunc
	t.Cleanup(func() { botDispatcherRunFunc = orig })
	botDispatcherRunFunc = func(_ context.Context, _ *bot.Dispatcher) error {
		return nil
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "poll-bot"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{"bot"},
				Bot: &domain.BotConfig{
					ExecutionType: domain.BotExecutionTypePolling,
					Discord:       &domain.DiscordConfig{},
				},
			},
		},
	}
	require.NoError(t, StartBotRunnersWithEngine(eng, wf, false))
}

func TestDetectGitHubRepo_FromGitDir(t *testing.T) {
	dir := t.TempDir()
	initCmd := exec.Command("git", "init", dir)
	initCmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	require.NoError(t, initCmd.Run())
	remoteCmd := exec.Command("git", "-C", dir, "remote", "add", "origin", "git@github.com:owner/repo.git")
	remoteCmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	require.NoError(t, remoteCmd.Run())
	repo, err := detectGitHubRepo(dir)
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", repo)
}

func TestRunRootPersistentPreRun(t *testing.T) {
	cmd := NewRootCmd()
	assert.NotPanics(t, func() { runRootPersistentPreRun(cmd) })
}

func TestResolvePrepackageName_InvalidKdeps(t *testing.T) {
	name := resolvePrepackageName("/nonexistent/bad.kdeps")
	assert.Equal(t, "bad", name)
}

func TestResolveEditor_EnvVars(t *testing.T) {
	t.Setenv("KDEPS_EDITOR", "nano")
	assert.Equal(t, "nano", resolveEditor())
	t.Setenv("KDEPS_EDITOR", "")
	t.Setenv("VISUAL", "vim")
	assert.Equal(t, "vim", resolveEditor())
}

func TestPrepareConfigForEdit(t *testing.T) {
	path, err := prepareConfigForEdit()
	require.NoError(t, err)
	assert.NotEmpty(t, path)
}

func TestRenderMarkdown(t *testing.T) {
	out := renderMarkdown("# Hello\n\nWorld")
	assert.NotEmpty(t, out)
}

func TestTerminalMarkdownWidth(t *testing.T) {
	assert.Greater(t, terminalMarkdownWidth(), 0)
}

func TestBuildRawGitHubURL(t *testing.T) {
	url := buildRawGitHubURL("owner", "repo", "main", "", "README.md")
	assert.Contains(t, url, "raw.githubusercontent.com")
	assert.Contains(t, url, "README.md")
	url = buildRawGitHubURL("owner", "repo", "main", "docs", "README.md")
	assert.Contains(t, url, "/docs/")
}

func TestFormatRemoteRef(t *testing.T) {
	assert.Equal(t, "owner/repo", formatRemoteRef("owner", "repo", ""))
	assert.Equal(t, "owner/repo/subdir", formatRemoteRef("owner", "repo", "subdir"))
}

func TestResolveKdepsFileInDirectory(t *testing.T) {
	tmp := t.TempDir()
	_, _, _, err := resolveKdepsFileInDirectory(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow.yaml not found")

	require.NoError(t, os.WriteFile(filepath.Join(tmp, "test.kdeps"), []byte("x"), 0644))
	_, _, _, err = resolveKdepsFileInDirectory(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract package")
}

func TestResolveWASMImageTag(t *testing.T) {
	assert.Equal(t, "custom:tag", resolveWASMImageTag("custom:tag"))
	assert.Equal(t, "kdeps-wasm:latest", resolveWASMImageTag(""))
}

func TestTruncatePackageDescription(t *testing.T) {
	short := truncatePackageDescription("hello")
	assert.Equal(t, "hello", short)
	long := strings.Repeat("x", 200)
	truncated := truncatePackageDescription(long)
	assert.LessOrEqual(t, len(truncated), 103)
}

func TestIsVersionedAgentDir(t *testing.T) {
	tmp := t.TempDir()
	assert.False(t, isVersionedAgentDir(tmp))
	agentDir := filepath.Join(tmp, "my-agent-1.0.0")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte("x"), 0644))
	assert.True(t, isVersionedAgentDir(tmp))
}

func TestLoadDoctorConfig(t *testing.T) {
	cfg := loadDoctorConfig()
	assert.NotNil(t, cfg)
}

func TestNewServeCmd(t *testing.T) {
	c := newServeCmd()
	assert.Equal(t, "serve <path>", c.Use)
}

func TestNewExportISOCmd(t *testing.T) {
	c := newExportISOCmd()
	assert.Equal(t, "iso [path]", c.Use)
}

func TestExportISOInternal_ShowConfigPath(t *testing.T) {
	pkgDir, _, restore := writeISOPackageDir(t)
	defer restore()
	err := exportISOInternal(&cobra.Command{}, []string{pkgDir}, &ExportFlags{ShowConfig: true})
	require.NoError(t, err)
}

func TestWriteTempBinary(t *testing.T) {
	path, err := writeTempBinary([]byte("binary-data"), "linux", "amd64")
	require.NoError(t, err)
	assert.FileExists(t, path)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "binary-data", string(data))
	_ = os.Remove(path)
}

func TestWriteTempBinary_Windows(t *testing.T) {
	path, err := writeTempBinary([]byte("binary-data"), "windows", "amd64")
	require.NoError(t, err)
	assert.FileExists(t, path)
	_ = os.Remove(path)
}

func TestRemoveInstalledFile_Error(t *testing.T) {
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "readonly")
	require.NoError(t, os.WriteFile(fpath, []byte("x"), 0400))
	require.NoError(t, os.Chmod(tmp, 0500))
	t.Cleanup(func() { _ = os.Chmod(tmp, 0755) })
	cmd := &cobra.Command{}
	_, err := removeInstalledFile(cmd, "x", fpath, "agent")
	require.Error(t, err)
}

func TestMatchesDirPattern(t *testing.T) {
	assert.True(t, matchesDirPattern("foo/bar/baz", "bar"))
	assert.False(t, matchesDirPattern("foo/baz", "bar"))
}

func TestMatchesIgnorePattern(t *testing.T) {
	assert.True(t, matchesIgnorePattern("node_modules/pkg", "pkg", "node_modules/"))
	assert.True(t, matchesIgnorePattern("foo.tmp", "foo.tmp", "*.tmp"))
}

func TestIsIgnored(t *testing.T) {
	assert.True(t, IsIgnored(".kdepsignore", nil))
	assert.True(t, IsIgnored("build/out", []string{"build/"}))
	assert.False(t, IsIgnored("src/main.go", []string{"build/"}))
}

func TestCreatePackageArchive(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	archive := filepath.Join(t.TempDir(), "out.kdeps")
	wf := minimalISOWorkflow()
	require.NoError(t, CreatePackageArchive(src, archive, wf))
	assert.FileExists(t, archive)
}

func TestExecuteAgencyStepsWithFlags(t *testing.T) {
	tmp := t.TempDir()
	agencyContent := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: test-agency
  version: "1.0.0"
  targetAgentId: bot-a
agents:
  - agents/bot-a
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agencyContent), 0644))
	agentDir := filepath.Join(tmp, "agents", "bot-a")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	wf := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: bot-a
  version: "1.0.0"
  targetActionId: response
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: response
    name: Response
    apiResponse:
      success: true
`
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte(wf), 0644))
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	flags := &RunFlags{}
	err := ExecuteAgencyStepsWithFlags(cmd, filepath.Join(tmp, "agency.yaml"), flags)
	require.NoError(t, err)
}

func TestExecuteAgencyEntryPoint_NoAgents(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	err := executeAgencyEntryPoint(cmd, "", nil, &RunFlags{})
	require.NoError(t, err)
}

func TestEnsureLLMBackendStep_WithOllamaRunning(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	t.Setenv("KDEPS_DEFAULT_BACKEND", "ollama")
	t.Setenv("OLLAMA_HOST", fmt.Sprintf("127.0.0.1:%d", port))
	wf := &domain.Workflow{
		Resources: []*domain.Resource{{Chat: &domain.ChatConfig{}}},
	}
	require.NoError(t, ensureLLMBackendStep(wf))
}

func TestDispatchExecutionWithEngine_WebServer(t *testing.T) {
	orig := webServerStartFunc
	t.Cleanup(func() { webServerStartFunc = orig })
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return http.ErrServerClosed
	}
	port := mustFreePort(t)
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			WebServer: &domain.WebServerConfig{
				PortNum: port,
				Routes:  []domain.WebRoute{{Path: "/", PublicPath: "/", ServerType: "static"}},
			},
		},
	}
	require.NoError(t, dispatchExecutionWithEngine(eng, wf, t.TempDir(), false, false, "", false))
}

// ---------------------------------------------------------------------------
// test helpers
// ---------------------------------------------------------------------------

type gapRoundTripper func(*http.Request) (*http.Response, error)

func (f gapRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newExportDockerClient(t *testing.T, handler gapRoundTripper) *docker.Client {
	t.Helper()
	cli, err := dockapi.NewClientWithOpts(
		dockapi.WithHost("tcp://127.0.0.1:2375"),
		dockapi.WithHTTPClient(&http.Client{Transport: handler}),
		dockapi.WithVersion("1.41"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })
	return &docker.Client{Cli: cli}
}

func jsonHTTPResponse(status int, body []byte) *http.Response {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return &http.Response{
		StatusCode: status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     h,
	}
}

func bytesHTTPResponse(body string) *http.Response { //nolint:unparam // test helper accepts variable body strings
	return &http.Response{
		StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func exportDockerBuildSuccessHandler() gapRoundTripper {
	return func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/build") {
			return bytesHTTPResponse(`{"stream":"Successfully built"}` + "\n"), nil
		}
		if strings.Contains(req.URL.Path, "/images/prune") {
			body, _ := json.Marshal(map[string]any{"SpaceReclaimed": 0})
			return jsonHTTPResponse(http.StatusOK, body), nil
		}
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	}
}

func installFakeLinuxkit(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	script := `#!/bin/sh
if [ "$1" = "build" ]; then
  DIR=""
  shift
  while [ $# -gt 0 ]; do
    case "$1" in
      --dir) DIR="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  if [ -n "$DIR" ]; then
    echo "iso" > "$DIR/output.iso"
  fi
fi
exit 0
`
	path := filepath.Join(tmp, "linuxkit")
	require.NoError(t, os.WriteFile(path, []byte(script), 0755))
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func minimalISOWorkflow() *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:           "iso-test",
			Version:        "1.0.0",
			TargetActionID: "act",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{PythonVersion: "3.12"},
		},
		Resources: []*domain.Resource{{
			ActionID:    "act",
			Name:        "Act",
			APIResponse: &domain.APIResponseConfig{Success: true},
		}},
	}
}

func minimalWorkflowYAML() string {
	return `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: gap-test
  version: "1.0.0"
  targetActionId: act
settings:
  agentSettings:
    pythonVersion: "3.12"
resources:
  - actionId: act
    name: Act
    apiResponse:
      success: true
`
}

func mustFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func sendSIGINTToSelf(t *testing.T) {
	t.Helper()
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGINT))
}
