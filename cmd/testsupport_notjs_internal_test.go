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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

//go:build !js

package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	dockclient "github.com/docker/docker/api/types/image"
	dockapi "github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/infra/docker"
	kdepshttp "github.com/kdeps/kdeps/v2/pkg/infra/http"
	wasmPkg "github.com/kdeps/kdeps/v2/pkg/infra/wasm"
)

func stubDispatchHooks(t *testing.T) {
	t.Helper()
	origBoth := execBothServersFn
	origWeb := execWebServerFn
	origAPI := execHTTPServerFn
	origBot := execBotRunnersFn
	origFile := execFileRunnerFn
	origSingle := execSingleRunFn
	origBothEng := execBothServersWithEngineFn
	origAPIEng := execHTTPServerWithEngineFn
	origWebEng := execWebServerWithEngineFn
	origBotEng := execBotRunnersWithEngineFn
	origFileEng := execFileRunnerWithEngineFn
	origSingleEng := execSingleRunWithEngineFn
	origMode := executionModeForFunc
	t.Cleanup(func() {
		execBothServersFn = origBoth
		execWebServerFn = origWeb
		execHTTPServerFn = origAPI
		execBotRunnersFn = origBot
		execFileRunnerFn = origFile
		execSingleRunFn = origSingle
		execBothServersWithEngineFn = origBothEng
		execHTTPServerWithEngineFn = origAPIEng
		execWebServerWithEngineFn = origWebEng
		execBotRunnersWithEngineFn = origBotEng
		execFileRunnerWithEngineFn = origFileEng
		execSingleRunWithEngineFn = origSingleEng
		executionModeForFunc = origMode
	})
	stub := func() error { return nil }
	execBothServersFn = func(_ *domain.Workflow, _ string, _, _ bool) error { return stub() }
	execWebServerFn = func(_ *domain.Workflow, _ string, _ bool) error { return stub() }
	execHTTPServerFn = func(_ *domain.Workflow, _ string, _, _ bool) error { return stub() }
	execBotRunnersFn = func(_ *domain.Workflow, _ bool) error { return stub() }
	execFileRunnerFn = func(_ *domain.Workflow, _ bool, _ string, _ bool) error { return stub() }
	execSingleRunFn = func(_ *domain.Workflow) error { return stub() }
	execBothServersWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow, _ string, _, _ bool) error {
		return stub()
	}
	execHTTPServerWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow, _ string, _, _ bool) error {
		return stub()
	}
	execWebServerWithEngineFn = func(_ *domain.Workflow, _ string, _ bool) error { return stub() }
	execBotRunnersWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow, _ bool) error {
		return stub()
	}
	execFileRunnerWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow, _ bool, _ string) error {
		return stub()
	}
	execSingleRunWithEngineFn = func(_ *executor.Engine, _ *domain.Workflow) error { return stub() }
}

type failingCloseListener struct {
	net.Listener
}

func (f *failingCloseListener) Close() error { return errors.New("close failed") }

type failingCloseConn struct {
	net.Conn
}

func (f *failingCloseConn) Close() error { return errors.New("conn close failed") }

func injectSignalNotify(t *testing.T) {
	t.Helper()
	orig := notifySignalsFunc
	t.Cleanup(func() { notifySignalsFunc = orig })
	notifySignalsFunc = func(c chan<- os.Signal, _ ...os.Signal) {
		go func() { c <- syscall.SIGINT }()
	}
}

type mockFileInfo struct {
	name string
	dir  bool
}

func (m *mockFileInfo) Name() string { return m.name }

func (m *mockFileInfo) Size() int64 { return 0 }

func (m *mockFileInfo) Mode() os.FileMode {
	if m.dir {
		return os.ModeDir | 0755
	}
	return 0644
}

func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }

func (m *mockFileInfo) IsDir() bool { return m.dir }

func (m *mockFileInfo) Sys() interface{} { return nil }

// osMkdirTemp is overridable for downloadRegistryArchive tests.
//
//nolint:gochecknoglobals // test-replaceable hook
var osMkdirTemp = os.MkdirTemp

func buildMinimalKdepsArchive(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func buildKagencyArchive(t *testing.T, sourceDir string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(
		t,
		filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(sourceDir, path)
			if relErr != nil {
				return relErr
			}
			if rel == "." {
				return nil
			}
			hdr, hdrErr := tar.FileInfoHeader(info, "")
			if hdrErr != nil {
				return hdrErr
			}
			hdr.Name = filepath.ToSlash(rel)
			if err = tw.WriteHeader(hdr); err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				f, openErr := os.Open(path)
				if openErr != nil {
					return openErr
				}
				_, copyErr := io.Copy(tw, f)
				_ = f.Close()
				if copyErr != nil {
					return copyErr
				}
			}
			return nil
		}),
	)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func buildMinimalKdepsArchivePath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(p, buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()), 0644),
	)
	return p
}

func createKomponentArchive(t *testing.T, dest, name, content string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Size: int64(len(content)), Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(dest, buf.Bytes(), 0644))
}

func writeEmbeddedTestBinary(t *testing.T, dir string) string {
	t.Helper()
	bin := filepath.Join(dir, "kdeps-bin")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755))
	kdeps := filepath.Join(dir, "pkg.kdeps")
	require.NoError(
		t,
		os.WriteFile(
			kdeps,
			buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()),
			0644,
		),
	)
	out := filepath.Join(dir, "embedded")
	require.NoError(t, AppendEmbeddedPackage(bin, kdeps, out))
	return out
}

// Ensure imports used.
var (
	_ = sha256.New
	_ = hex.EncodeToString
	_ = json.Marshal
	_ = dockclient.InspectResponse{}
	_ = dockapi.NewClientWithOpts
	_ = yaml.Marshal
	_ = wasmPkg.BundleConfig{}
	_ = time.Now
)

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

// captureStdout redirects os.Stdout to a pipe for the duration of f.
// The pipe is drained concurrently so writes larger than the pipe buffer
// (including stray output from leaked goroutines) cannot deadlock f.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		done <- buf.String()
	}()

	orig := os.Stdout
	os.Stdout = w //nolint:reassign // test helper
	f()
	w.Close()
	os.Stdout = orig //nolint:reassign // test helper

	return <-done
}

// buildTarGz creates a temp tar.gz file with the given files and returns its path.
func buildTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "archive.tar.gz")
	f, err := os.Create(tmp)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(content))}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return tmp
}

// helpers for prepackage host arch detection.
func runtimeGOOS() string { return "linux" }

func runtimeGOARCH() string { return "amd64" }

// Ensure sha256 used in registry tests.
var _ = sha256.Sum256

// Ensure hex used.
var _ = hex.EncodeToString

// Ensure config import used.
var _ = config.LoadStruct

// Ensure json used.
var _ = json.Marshal

// ExtractFromTarGz is an alias for the unexported extractFromTarGz helper,
// exposed for unit testing only.
var ExtractFromTarGz = extractFromTarGz //nolint:gochecknoglobals // test-only export

// ExtractFromZip is an alias for the unexported extractFromZip helper,
// exposed for unit testing only.
var ExtractFromZip = extractFromZip //nolint:gochecknoglobals // test-only export

// FetchURL is an alias for the unexported fetchURL helper,
// exposed for unit testing only.
var FetchURL = fetchURL //nolint:gochecknoglobals // test-only export

// CleanBinaryPath is an alias for the unexported cleanBinaryPath helper,
// exposed for unit testing only.
var CleanBinaryPath = cleanBinaryPath //nolint:gochecknoglobals // test-only export

// GoosToReleaseOS is an alias for the unexported goosToReleaseOS helper,
// exposed for unit testing only.
var GoosToReleaseOS = goosToReleaseOS //nolint:gochecknoglobals // test-only export

// GoarchToReleaseArch is an alias for the unexported goarchToReleaseArch helper,
// exposed for unit testing only.
var GoarchToReleaseArch = goarchToReleaseArch //nolint:gochecknoglobals // test-only export

// DownloadKdepsBinaryToTemp is an alias for the unexported
// downloadKdepsBinaryToTemp helper, exposed for unit testing only.
var DownloadKdepsBinaryToTemp = downloadKdepsBinaryToTemp //nolint:gochecknoglobals // test-only export

// GithubReleasesBaseURL allows tests to override the base URL for downloads.
// Tests should restore the original value via t.Cleanup().
var GithubReleasesBaseURL = &githubReleasesBaseURL //nolint:gochecknoglobals // test-only export

// IsAgencyFile exposes the unexported isAgencyFile helper for unit testing.
var IsAgencyFile = isAgencyFile //nolint:gochecknoglobals // test-only export

// ExportISOInternal exposes the unexported exportISOInternal helper for unit testing.
var ExportISOInternal = exportISOInternal //nolint:gochecknoglobals // test-only export

// BuildAgentNameMap exposes the unexported buildAgentNameMap helper for unit testing.
var BuildAgentNameMap = buildAgentNameMap //nolint:gochecknoglobals // test-only export

// IsKagencyFile exposes the unexported isKagencyFile helper for unit testing.
var IsKagencyFile = isKagencyFile //nolint:gochecknoglobals // test-only export

// PrintRoutes exposes the unexported printRoutes helper for unit testing.
var PrintRoutes = printRoutes //nolint:gochecknoglobals // test-only export

// PrintBotRequirements exposes the unexported printBotRequirements helper for unit testing.
var PrintBotRequirements = printBotRequirements //nolint:gochecknoglobals // test-only export

// ComponentInstallDir exposes the unexported componentInstallDir helper for unit testing.
var ComponentInstallDir = componentInstallDir //nolint:gochecknoglobals // test-only export

// ListLocalComponents exposes the unexported listLocalComponents helper for unit testing.
var ListLocalComponents = listLocalComponents //nolint:gochecknoglobals // test-only export

// ReadReadmeForComponent exposes the unexported readReadmeForComponent helper for unit testing.
var ReadReadmeForComponent = readReadmeForComponent //nolint:gochecknoglobals // test-only export

// FindReadmeInDir exposes the unexported findReadmeInDir helper for unit testing.
var FindReadmeInDir = findReadmeInDir //nolint:gochecknoglobals // test-only export

// ParseRemoteRef exposes the unexported parseRemoteRef helper for unit testing.
var ParseRemoteRef = parseRemoteRef //nolint:gochecknoglobals // test-only export

// FetchRemoteReadme exposes the unexported fetchRemoteReadme helper for unit testing.
var FetchRemoteReadme = fetchRemoteReadme //nolint:gochecknoglobals // test-only export

// ResolveInfoReadme exposes the unexported resolveInfoReadme helper for unit testing.
var ResolveInfoReadme = resolveInfoReadme //nolint:gochecknoglobals // test-only export

// GithubRawBaseURL allows tests to override the GitHub raw content base URL.
// Tests should restore the original value via t.Cleanup().
var GithubRawBaseURL = &githubRawBaseURL //nolint:gochecknoglobals // test-only export

// CloneFromRemote exposes the unexported cloneFromRemote helper for unit testing.
var CloneFromRemote = cloneFromRemote //nolint:gochecknoglobals // test-only export

// DetectCloneType exposes the unexported detectCloneType helper for unit testing.
var DetectCloneType = detectCloneType //nolint:gochecknoglobals // test-only export

// GithubArchiveBaseURL allows tests to override the GitHub archive download base URL.
// Tests should restore the original value via t.Cleanup().
var GithubArchiveBaseURL = &githubArchiveBaseURL //nolint:gochecknoglobals // test-only export

// CloneAsComponent exposes the unexported cloneAsComponent helper for unit testing.
var CloneAsComponent = cloneAsComponent //nolint:gochecknoglobals // test-only export

// FindFileWithSuffix exposes the unexported findFileWithSuffix helper for unit testing.
var FindFileWithSuffix = findFileWithSuffix //nolint:gochecknoglobals // test-only export

// ExtractKomponent exposes the unexported extractKomponent helper for unit testing.
var ExtractKomponent = extractKomponent //nolint:gochecknoglobals // test-only export

// ReadReadmeFromKomponent exposes the unexported readReadmeFromKomponent helper for unit testing.
var ReadReadmeFromKomponent = readReadmeFromKomponent //nolint:gochecknoglobals // test-only export

// ResolveLocalReadme exposes the unexported resolveLocalReadme helper for unit testing.
var ResolveLocalReadme = resolveLocalReadme //nolint:gochecknoglobals // test-only export

// NewInfoCmd exposes newInfoCmd for cobra command testing.
var NewInfoCmd = newInfoCmd //nolint:gochecknoglobals // test-only export

// NewRegistryListCmd exposes newRegistryListCmd for unit testing.
var NewRegistryListCmd = newRegistryListCmd //nolint:gochecknoglobals // test-only export

// NewRegistryInfoCmd exposes newRegistryInfoCmd for unit testing.
var NewRegistryInfoCmd = newRegistryInfoCmd //nolint:gochecknoglobals // test-only export

// UnwrapArchiveRoot exposes the unexported unwrapArchiveRoot helper for unit testing.
var UnwrapArchiveRoot = unwrapArchiveRoot //nolint:gochecknoglobals // test-only export

// ComponentUpdateInternal exposes componentUpdateInternal for unit testing.
var ComponentUpdateInternal = componentUpdateInternal //nolint:gochecknoglobals // test-only export

// FindUpdateTargetComponentDirs exposes findUpdateTargetComponentDirs for unit testing.
var FindUpdateTargetComponentDirs = findUpdateTargetComponentDirs //nolint:gochecknoglobals // test-only export

// NewRunCmdForTest exposes newRunCmd for use in external unit tests.
var NewRunCmdForTest = newRunCmd //nolint:gochecknoglobals // test-only export

// DispatchExecutionWithEngine exposes the unexported dispatchExecutionWithEngine
// helper for white-box unit tests.
var DispatchExecutionWithEngine = dispatchExecutionWithEngine //nolint:gochecknoglobals // test-only export

// NewRegistryUninstallCmd exposes newRegistryUninstallCmd for integration tests.
var NewRegistryUninstallCmd = newRegistryUninstallCmd //nolint:gochecknoglobals // test-only export

// NewRegistryUpdateCmd exposes newRegistryUpdateCmd for integration tests.
var NewRegistryUpdateCmd = newRegistryUpdateCmd //nolint:gochecknoglobals // test-only export

// KdepsAgentsDir exposes kdepsAgentsDir for integration tests.
var KdepsAgentsDir = kdepsAgentsDir //nolint:gochecknoglobals // test-only export

// IsLocalFilePath exposes the unexported isLocalFilePath helper for unit testing.
var IsLocalFilePath = isLocalFilePath //nolint:gochecknoglobals // test-only export

// CopyFile exposes the unexported copyFile helper for unit testing.
var CopyFile = copyFile //nolint:gochecknoglobals // test-only export

// CopyDir exposes the unexported copyDir helper for unit testing.
var CopyDir = copyDir //nolint:gochecknoglobals // test-only export

// InstallLocalFile exposes the unexported installLocalFile helper for unit testing.
var InstallLocalFile = installLocalFile //nolint:gochecknoglobals // test-only export

// GetFormatMap exposes the unexported getFormatMap helper for unit testing.
var GetFormatMap = getFormatMap //nolint:gochecknoglobals // test-only export

// ResolveOutputPath exposes the unexported resolveOutputPath helper for unit testing.
var ResolveOutputPath = resolveOutputPath //nolint:gochecknoglobals // test-only export

// QemuSystem exposes the unexported qemuSystem helper for unit testing.
var QemuSystem = qemuSystem //nolint:gochecknoglobals // test-only export

// WorkflowPorts exposes the unexported workflowPorts helper for unit testing.
var WorkflowPorts = workflowPorts //nolint:gochecknoglobals // test-only export

// JoinStrings exposes the unexported joinStrings helper for unit testing.
var JoinStrings = joinStrings //nolint:gochecknoglobals // test-only export

// CmdExtractTarEntry exposes the unexported cmdExtractTarEntry helper for unit testing.
var CmdExtractTarEntry = cmdExtractTarEntry //nolint:gochecknoglobals // test-only export

// CmdExtractTarGz exposes the unexported cmdExtractTarGz helper for unit testing.
var CmdExtractTarGz = cmdExtractTarGz //nolint:gochecknoglobals // test-only export

// BootstrapConfigFunc exposes bootstrapConfigFunc for unit testing.
var BootstrapConfigFunc = &bootstrapConfigFunc //nolint:gochecknoglobals // test-only export

// LoadConfigFunc exposes loadConfigFunc for unit testing.
var LoadConfigFunc = &loadConfigFunc //nolint:gochecknoglobals // test-only export

// DetectGitHubRepoFunc exposes detectGitHubRepoFunc for unit testing.
var DetectGitHubRepoFunc = &detectGitHubRepoFunc //nolint:gochecknoglobals // test-only export

// ComputeRemoteSHA256Func exposes computeRemoteSHA256Func for unit testing.
var ComputeRemoteSHA256Func = &computeRemoteSHA256Func //nolint:gochecknoglobals // test-only export

// ConfigScaffoldFunc exposes configScaffoldFunc for unit testing.
var ConfigScaffoldFunc = &configScaffoldFunc //nolint:gochecknoglobals // test-only export

// ConfigPathFunc exposes configPathFunc for unit testing.
var ConfigPathFunc = &configPathFunc //nolint:gochecknoglobals // test-only export

// NewMarkdownRendererFunc exposes newMarkdownRendererFunc for unit testing.
var NewMarkdownRendererFunc = &newMarkdownRendererFunc //nolint:gochecknoglobals // test-only export

// TermGetSizeFunc exposes termGetSizeFunc for unit testing.
var TermGetSizeFunc = &termGetSizeFunc //nolint:gochecknoglobals // test-only export

// OsCreateTempFunc exposes osCreateTempFunc for unit testing.
var OsCreateTempFunc = &osCreateTempFunc //nolint:gochecknoglobals // test-only export

// DownloadArchiveFunc exposes downloadArchiveFunc for unit testing.
var DownloadArchiveFunc = &downloadArchiveFunc //nolint:gochecknoglobals // test-only export

// HTTPServerStartHook is the type of httpServerStartFunc for test overrides.
type HTTPServerStartHook func(srv *kdepshttp.Server, addr string, devMode bool) error

// SetHTTPServerStartFuncForTest overrides httpServerStartFunc and returns the previous hook.
func SetHTTPServerStartFuncForTest(fn HTTPServerStartHook) HTTPServerStartHook {
	orig := httpServerStartFunc
	httpServerStartFunc = fn
	return orig
}

// createTarGz creates a tar.gz file at destPath with the given tar headers and data.
// For directory entries (TypeDir), the size is forced to 0.
func createTarGz(t *testing.T, destPath string, headers []*tar.Header, data [][]byte) {
	t.Helper()
	f, err := os.Create(destPath)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for i, h := range headers {
		hd := *h // copy
		if hd.Typeflag == tar.TypeDir {
			hd.Size = 0
		} else if hd.Size == 0 && len(data) > i && len(data[i]) > 0 {
			hd.Size = int64(len(data[i]))
		}
		require.NoError(t, tw.WriteHeader(&hd))
		if len(data) > i && len(data[i]) > 0 {
			_, err = tw.Write(data[i])
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
}
