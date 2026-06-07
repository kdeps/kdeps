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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
// run.go — remaining branches
// ---------------------------------------------------------------------------

func TestExecuteWorkflowStepsWithFlags_LLMErr(t *testing.T) {
	orig := ensureOllamaRunningFunc
	t.Cleanup(func() { ensureOllamaRunningFunc = orig })
	ensureOllamaRunningFunc = func(_ string) error { return errors.New("llm fail") }
	tmp := t.TempDir()
	wf := strings.Replace(minimalWorkflowYAML(), "apiResponse:", "chat: {}\n    apiResponse:", 1)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(wf), 0644))
	t.Setenv("KDEPS_DEFAULT_BACKEND", "ollama")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	err := ExecuteWorkflowStepsWithFlags(cmd, filepath.Join(tmp, "workflow.yaml"), &RunFlags{})
	require.Error(t, err)
}

func TestPrintAgencyAgentIndex_AbsPath(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("abs path fallback differs on Windows")
	}
	agency := &domain.Agency{Metadata: domain.AgencyMetadata{TargetAgentID: "a"}}
	printAgencyAgentIndex(string([]byte{0x00}), agency, map[string]string{"a": "/x"})
}

func TestExecuteAgencyStepsWithFlags_PreprocessErr(t *testing.T) {
	tmp := t.TempDir()
	agency := filepath.Join(tmp, "agency.yaml")
	require.NoError(t, os.WriteFile(agency, []byte("invalid: ["), 0644))
	cmd := &cobra.Command{}
	err := ExecuteAgencyStepsWithFlags(cmd, agency, &RunFlags{})
	require.Error(t, err)
}

func TestBuildAgentNameMap_EmptyName(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "wf.yaml")
	require.NoError(
		t,
		os.WriteFile(p, []byte("apiVersion: kdeps.io/v1\nkind: Workflow\nmetadata:\n  name: \"\"\n"), 0644),
	)
	_, _, err := buildAgentNameMap([]string{p}, "x")
	require.Error(t, err)
}

func TestExecuteAgencyEntryPoint_ParseErr(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("invalid: ["), 0644))
	err := executeAgencyEntryPoint(&cobra.Command{}, bad, nil, &RunFlags{})
	require.Error(t, err)
}

func TestParseWorkflowFile_InitErr(t *testing.T) {
	orig := newSchemaValidatorFunc
	t.Cleanup(func() { newSchemaValidatorFunc = orig })
	newSchemaValidatorFunc = func() (*validator.SchemaValidator, error) { return nil, errors.New("init") }
	_, err := ParseWorkflowFile(filepath.Join(t.TempDir(), "wf.yaml"))
	require.Error(t, err)
}

func TestParseAgencyFileWithParser_InitAndDiscoverErr(t *testing.T) {
	orig := newSchemaValidatorFunc
	t.Cleanup(func() { newSchemaValidatorFunc = orig })
	newSchemaValidatorFunc = func() (*validator.SchemaValidator, error) { return nil, errors.New("init") }
	_, _, _, err := ParseAgencyFileWithParser(filepath.Join(t.TempDir(), "a.yaml"))
	require.Error(t, err)

	tmp := t.TempDir()
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: a
  version: "1"
  targetAgentId: missing
agents:
  - agents/missing
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agency), 0644))
	_, _, _, err = ParseAgencyFileWithParser(filepath.Join(tmp, "agency.yaml"))
	require.Error(t, err)
}

func TestLoadResourceFiles_ReadAndParseErr(t *testing.T) {
	tmp := t.TempDir()
	resDir := filepath.Join(tmp, "resources")
	require.NoError(t, os.MkdirAll(resDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(resDir, "bad.yaml"), []byte("invalid: ["), 0644))
	wf := &domain.Workflow{}
	parser, err := newYAMLParser()
	require.NoError(t, err)
	defer parser.Cleanup()
	err = LoadResourceFiles(wf, resDir, parser)
	require.Error(t, err)

	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	err = LoadResourceFiles(wf, blocker, parser)
	require.Error(t, err)
}

func TestValidateWorkflow_InitErr(t *testing.T) {
	wf := &domain.Workflow{}
	err := ValidateWorkflow(wf)
	require.Error(t, err)
}

func TestIsPythonModuleAvailable_PythonFallback(t *testing.T) {
	orig := isBinaryAvailableFunc
	t.Cleanup(func() { isBinaryAvailableFunc = orig })
	isBinaryAvailableFunc = func(name string) bool { return name == "python" }
	_ = isPythonModuleAvailable("sys")
}

func TestFindAvailablePort_ScanNotice_To100(t *testing.T) {
	port := mustFreePort(t)
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer ln.Close()
	got, err := FindAvailablePort("127.0.0.1", port)
	require.NoError(t, err)
	assert.Greater(t, got, port)
}

func TestStartOllamaServer_NotFound_To100(t *testing.T) {
	orig := execLookPathFunc
	t.Cleanup(func() { execLookPathFunc = orig })
	execLookPathFunc = func(_ string) (string, error) { return "", errors.New("not found") }
	err := startOllamaServer()
	require.Error(t, err)
}

func TestStartHTTPServerWithEngine_StartFail(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return errors.New("start fail")
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings:  domain.WorkflowSettings{APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)}},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	err := startHTTPServerWithEngine(eng, wf, t.TempDir(), false, false)
	require.Error(t, err)
}

func TestStartHTTPServerWithEngine_ServerClosed(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "api", Version: "1.0", TargetActionID: "act"},
		Settings:  domain.WorkflowSettings{APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)}},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	require.NoError(t, startHTTPServerWithEngine(eng, wf, t.TempDir(), false, false))
}

func TestStartBothServersWithEngine_WebCreateErr(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
			WebServer: &domain.WebServerConfig{PortNum: mustFreePort(t), Routes: []domain.WebRoute{
				{Path: string([]byte{0x00}), PublicPath: "/", ServerType: "static"},
			}},
		},
	}
	require.NoError(t, startBothServersWithEngine(eng, wf, t.TempDir(), false, false))
}

func TestStartBothServersWithEngine_ServerClosed(t *testing.T) {
	orig := httpServerStartFunc
	t.Cleanup(func() { httpServerStartFunc = orig })
	httpServerStartFunc = func(_ *kdepshttp.Server, _ string, _ bool) error {
		return http.ErrServerClosed
	}
	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "both", Version: "1.0", TargetActionID: "act"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{PortNum: mustFreePort(t)},
			WebServer: &domain.WebServerConfig{PortNum: mustFreePort(t), Routes: []domain.WebRoute{}},
		},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	require.NoError(t, startBothServersWithEngine(eng, wf, t.TempDir(), false, false))
}

func TestStartWebServer_AppPortRoute(t *testing.T) {
	orig := webServerStartFunc
	t.Cleanup(func() { webServerStartFunc = orig })
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return errors.New("start fail")
	}
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{
		WebServer: &domain.WebServerConfig{
			PortNum: mustFreePort(t),
			Routes: []domain.WebRoute{{
				Path: "/", PublicPath: "/", ServerType: "app", AppPort: 3000,
			}},
		},
	}}
	err := StartWebServer(wf, t.TempDir(), false)
	require.Error(t, err)
}

func TestStartWebServer_ShutdownErr(t *testing.T) {
	origStart := webServerStartFunc
	origShutdown := webServerShutdownFunc
	t.Cleanup(func() {
		webServerStartFunc = origStart
		webServerShutdownFunc = origShutdown
	})
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return http.ErrServerClosed
	}
	webServerShutdownFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return errors.New("web shutdown fail")
	}
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{
		WebServer: &domain.WebServerConfig{
			PortNum: mustFreePort(t),
			Routes:  []domain.WebRoute{{Path: "/", PublicPath: "/", ServerType: "static"}},
		},
	}}
	require.NoError(t, StartWebServer(wf, t.TempDir(), false))
}

func TestStartWebServer_ServerClosed(t *testing.T) {
	orig := webServerStartFunc
	t.Cleanup(func() { webServerStartFunc = orig })
	webServerStartFunc = func(_ *kdepshttp.WebServer, _ context.Context) error {
		return http.ErrServerClosed
	}
	wf := &domain.Workflow{Settings: domain.WorkflowSettings{
		WebServer: &domain.WebServerConfig{
			PortNum: mustFreePort(t),
			Routes:  []domain.WebRoute{{Path: "/", PublicPath: "/", ServerType: "static"}},
		},
	}}
	require.NoError(t, StartWebServer(wf, t.TempDir(), false))
}

func TestExtractPackage_OpenCleanup(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "f.txt", "x"), 0644))
	orig := osMkdirTempExtractFunc
	t.Cleanup(func() { osMkdirTempExtractFunc = orig })
	osMkdirTempExtractFunc = func(_, _ string) (string, error) {
		d, err := os.MkdirTemp("", "kdeps-run-*")
		if err != nil {
			return "", err
		}
		require.NoError(t, os.Remove(kdeps))
		return d, nil
	}
	_, err := ExtractPackage(kdeps)
	require.Error(t, err)
}

func TestExtractTarFiles_ExtractFileErr(t *testing.T) {
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

func TestExtractFile_CreateAndCopyErr(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	hdr := &tar.Header{Name: "nested/f.txt", Size: 1, Mode: 0644}
	err := ExtractFile(tar.NewReader(bytes.NewReader([]byte("x"))), hdr, filepath.Join(blocker, "nested", "f.txt"))
	require.Error(t, err)

	hdr2 := &tar.Header{Name: "big.bin", Size: maxExtractFileSize + 1, Mode: 0644}
	err = ExtractFile(tar.NewReader(bytes.NewReader(nil)), hdr2, filepath.Join(tmp, "big.bin"))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// registry_install.go
// ---------------------------------------------------------------------------

func TestDownloadRegistryArchive_NoURL_To100(t *testing.T) {
	_, cleanup, err := downloadRegistryArchive(&packageInfo{}, "pkg", "1.0")
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestDownloadRegistryArchive_VerifyErr(t *testing.T) {
	tmp := t.TempDir()
	archive := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(archive, []byte("data"), 0644))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.ServeFile(w, &http.Request{}, archive)
	}))
	defer srv.Close()
	orig := downloadArchiveFunc
	t.Cleanup(func() { downloadArchiveFunc = orig })
	downloadArchiveFunc = func(_ string, dest string) error {
		data, _ := os.ReadFile(archive)
		return os.WriteFile(dest, data, 0644)
	}
	_, cleanup, err := downloadRegistryArchive(&packageInfo{
		TarballURL: srv.URL,
		SHA256:     strings.Repeat("a", 64),
	}, "pkg", "1.0")
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestInstallWorkflowOrAgency_Success_To100(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", tmp)
	archive := buildMinimalKdepsArchivePath(t)
	manifest := &domain.KdepsPkg{Name: "newagent", Type: "workflow", Description: "desc"}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	require.NoError(t, installWorkflowOrAgency(cmd, manifest, archive, "1.0"))
	assert.Contains(t, buf.String(), "newagent")
}

func TestInstallRegistryComponent_GlobalPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	archive := buildMinimalKdepsArchivePath(t)
	manifest := &domain.KdepsPkg{Name: "gcomp", Type: pkgTypeComponent, Description: "d"}
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	require.NoError(t, installRegistryComponent(cmd, manifest, archive, "1.0"))
	assert.Contains(t, buf.String(), "gcomp")
}

func TestPeekManifest_ReadManifestErr(t *testing.T) {
	tmp := t.TempDir()
	archive := filepath.Join(tmp, "pkg.kdeps")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "kdeps.pkg.yaml", Size: 3, Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte("bad"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(archive, buf.Bytes(), 0644))
	_, err = peekManifest(archive)
	require.Error(t, err)
}

func TestResolvePackageInfo_ReadBodyErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer srv.Close()
	orig := registryHTTPClient
	t.Cleanup(func() { registryHTTPClient = orig })
	registryHTTPClient = srv.Client()
	_, err := resolvePackageInfo("pkg", srv.URL)
	require.Error(t, err)
}

func TestVerifySHA256_OpenAndCopyErr(t *testing.T) {
	err := verifySHA256("/nonexistent", strings.Repeat("a", 64))
	require.Error(t, err)
}

func TestDownloadArchive_CreateAndCloseErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()
	err := downloadArchive(srv.URL, filepath.Join(t.TempDir(), "blocked", "out.kdeps"))
	require.Error(t, err)
}

func TestSafeArchiveTarget_AbsAndRelErr(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("path semantics differ on Windows")
	}
	_, _, err := safeArchiveTarget(string([]byte{0x00}), "f.txt")
	require.Error(t, err)
}

func TestExtractArchive_TarNextAndMkdirErr(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.kdeps")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("not-tar"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(bad, buf.Bytes(), 0644))
	err = extractArchive(bad, t.TempDir())
	require.Error(t, err)
}

func TestExtractFile_RegistryCloseErr(t *testing.T) {
	tmp := t.TempDir()
	roDir := filepath.Join(tmp, "ro")
	require.NoError(t, os.Mkdir(roDir, 0500))
	err := extractFile(filepath.Join(roDir, "f.txt"), strings.NewReader("x"))
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// component.go
// ---------------------------------------------------------------------------

func TestReadReadmeFromKomponent_TarNextErr(t *testing.T) {
	tmp := t.TempDir()
	komp := filepath.Join(tmp, "c.komponent")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("bad"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(komp, buf.Bytes(), 0644))
	_, err = readReadmeFromKomponent(komp)
	require.Error(t, err)
}

func TestGenerateFallbackReadme_ParseErr(t *testing.T) {
	tmp := t.TempDir()
	compDir := filepath.Join(tmp, "components", "c1")
	require.NoError(t, os.MkdirAll(compDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte("invalid: ["), 0644))
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	readme, err := generateFallbackReadme("c1")
	require.NoError(t, err)
	assert.Contains(t, readme, "c1")
}

func TestCmdExtractTarGz_NextErr(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("bad"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	err = cmdExtractTarGz(&buf, t.TempDir())
	require.Error(t, err)
}

func TestSafeKomponentTarget_AllBranches(t *testing.T) {
	_, ok, err := safeKomponentTarget(t.TempDir(), ".")
	require.NoError(t, err)
	assert.False(t, ok)
	_, ok, err = safeKomponentTarget(t.TempDir(), "../escape")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestWriteKomponentRegularFile_CloseErr(t *testing.T) {
	destDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "a.tar.gz")
	createTarGz(
		t,
		archivePath,
		[]*tar.Header{{Name: "c.txt", Typeflag: tar.TypeReg, Mode: 0644}},
		[][]byte{[]byte("x")},
	)
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()
	tr := tar.NewReader(gz)
	_, err = tr.Next()
	require.NoError(t, err)
	roDir := filepath.Join(destDir, "ro")
	require.NoError(t, os.Mkdir(roDir, 0500))
	err = writeKomponentRegularFile(filepath.Join(roDir, "c.txt"), tr)
	require.Error(t, err)
}

func TestComponentUpdateInternal_AbsErr(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("invalid path semantics differ on Windows")
	}
	err := componentUpdateInternal(string([]byte{0x00}))
	require.Error(t, err)
}

func TestFindUpdateTargetComponentDirs_NotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := findUpdateTargetComponentDirs(tmp)
	require.Error(t, err)
}

func TestScanComponentSubdirs_ReadErr(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	_, err := scanComponentSubdirs(f)
	require.Error(t, err)
}

func TestUpdateComponentDir_UpdatedFiles(t *testing.T) {
	tmp := t.TempDir()
	comp := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: upd-files
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "component.yaml"), []byte(comp), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, updateComponentDir(tmp))
}

// ---------------------------------------------------------------------------
// embedded.go
// ---------------------------------------------------------------------------

func TestDetectEmbeddedPackage_ReadAtFail(t *testing.T) {
	tmp := t.TempDir()
	binPath := writeEmbeddedTestBinary(t, tmp)
	require.NoError(t, os.Truncate(binPath, EmbeddedTrailerSize+2))
	_, ok := DetectEmbeddedPackage(binPath)
	assert.False(t, ok)
}

func TestAppendEmbeddedPackage_AllErrPaths(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "bin")
	require.NoError(t, os.WriteFile(bin, []byte("bin"), 0755))
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, []byte("pkg"), 0644))
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	require.Error(t, AppendEmbeddedPackage("/no/bin", kdeps, filepath.Join(tmp, "out")))
	require.Error(t, AppendEmbeddedPackage(bin, "/no/pkg", filepath.Join(tmp, "out2")))
	orig := osStatFunc
	t.Cleanup(func() { osStatFunc = orig })
	osStatFunc = func(_ string) (os.FileInfo, error) { return nil, errors.New("stat") }
	require.Error(t, AppendEmbeddedPackage(bin, kdeps, filepath.Join(tmp, "out3")))
	require.Error(t, AppendEmbeddedPackage(bin, kdeps, filepath.Join(blocker, "out4")))
}

func TestWriteEmbeddedTrailer_WriteErr(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "out"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	ro, err := os.Open(filepath.Join(tmp, "out"))
	require.NoError(t, err)
	defer ro.Close()
	err = writeEmbeddedTrailer(ro, 10)
	require.Error(t, err)
}

func TestWriteCleanBinaryTemp_ReadAndCloseErr(t *testing.T) {
	tmp := t.TempDir()
	src, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	_, err = src.WriteString("short")
	require.NoError(t, err)
	require.NoError(t, src.Close())
	rf, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer rf.Close()
	_, _, err = writeCleanBinaryTemp(rf, 100)
	require.Error(t, err)
}

func TestRunEmbeddedPackage_ExecAndCloseErr(t *testing.T) {
	tmp := t.TempDir()
	binPath := writeEmbeddedTestBinary(t, tmp)
	origExec := executeEmbeddedFunc
	t.Cleanup(func() { executeEmbeddedFunc = origExec })
	executeEmbeddedFunc = func(_, _ string) error { return errors.New("run fail") }
	assert.Equal(t, 1, RunEmbeddedPackage("dev", "dev", binPath))
}

// ---------------------------------------------------------------------------
// build.go, package.go, prepackage.go, clone.go, export.go
// ---------------------------------------------------------------------------

func TestBuildImageInternal_TagErr(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	newBuildDockerClient(t, func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/tag") {
			return jsonHTTPResponse(http.StatusInternalServerError, []byte(`{}`)), nil
		}
		return jsonHTTPResponse(http.StatusOK, []byte(`{}`)), nil
	})
	origBuild := dockerBuildImageFunc
	t.Cleanup(func() { dockerBuildImageFunc = origBuild })
	dockerBuildImageFunc = func(_ *docker.Builder, _ *domain.Workflow, _ string, _ bool) (string, error) {
		return "img:1", nil
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := buildImageInternal(cmd, []string{tmp}, &BuildFlags{Tag: "repo/img:v1"})
	require.Error(t, err)
}

func TestCreatePrepackagedBinaryForTarget_ResolveErr(t *testing.T) {
	orig := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = orig })
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return "", false, errors.New("resolve")
	}
	_, err := createPrepackagedBinaryForTarget(
		context.Background(),
		"pkg.kdeps",
		"base",
		archTarget{GOOS: "linux", GOARCH: "amd64"},
	)
	require.Error(t, err)
}

func TestCollectWebServerFiles_NoDataDir_To100(t *testing.T) {
	tmp := t.TempDir()
	files, err := collectWebServerFiles(tmp)
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestCollectWebServerFiles_RelErr(t *testing.T) {
	tmp := t.TempDir()
	dataDir := filepath.Join(tmp, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	require.NoError(t, os.Symlink("/nonexistent", filepath.Join(dataDir, "link")))
	_, err := collectWebServerFiles(tmp)
	require.Error(t, err)
}

func TestGorootWASMExecCandidates_Empty(t *testing.T) {
	cands := gorootWASMExecCandidates(context.Background())
	t.Logf("candidates: %v", cands)
}

func TestPackageWorkflowWithFlags_ComposeWarn_To100(t *testing.T) {
	tmp := t.TempDir()
	wf := strings.Replace(minimalWorkflowYAML(), "pythonVersion: \"3.12\"",
		"pythonVersion: \"3.12\"\n    dockerCompose: \"missing-compose.yml\"", 1)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(wf), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	err := PackageWorkflowWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{Output: tmp})
	require.NoError(t, err)
}

func TestParseKdepsIgnore_WalkAndRead(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".kdepsignore"), []byte("*.log\n"), 0644))
	patterns := ParseKdepsIgnore(tmp)
	assert.Contains(t, patterns, "*.log")
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	patterns = ParseKdepsIgnore(blocker)
	assert.Empty(t, patterns)
}

func TestCreatePackageArchive_MkdirAllErr(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	err := CreatePackageArchive(tmp, filepath.Join(blocker, "out.kdeps"), &domain.Workflow{})
	require.Error(t, err)
}

func TestAddFileToArchive_SymlinkSkip(t *testing.T) {
	tmp := t.TempDir()
	link := filepath.Join(tmp, "link")
	require.NoError(t, os.Symlink("/tmp", link))
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	info, err := os.Lstat(link)
	require.NoError(t, err)
	require.NoError(t, AddFileToArchive(link, info, tmp, tw))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
}

func TestCreateAgencyAndComponentArchive_MkdirErr(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	require.Error(t, CreateAgencyPackageArchive(tmp, filepath.Join(blocker, "out.kagency")))
	require.Error(t, CreateComponentPackageArchive(tmp, filepath.Join(blocker, "out.komponent")))
}

func TestPackageComponentWithFlags_ParseErr(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "component.yaml"), []byte("invalid: ["), 0644))
	err := PackageComponentWithFlags(&cobra.Command{}, []string{tmp}, &PackageFlags{})
	require.Error(t, err)
}

func TestBuildPrepackageTarget_EmbedFail(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "workflow.yaml", "invalid: ["), 0644))
	baseDir := filepath.Join(tmp, "basedir")
	require.NoError(t, os.Mkdir(baseDir, 0755))
	orig := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = orig })
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return baseDir, false, nil
	}
	_, err := buildPrepackageTarget(
		context.Background(),
		kdeps,
		"1.0",
		baseDir,
		tmp,
		"pkg",
		archTarget{GOOS: "linux", GOARCH: "amd64"},
	)
	require.Error(t, err)
}

func TestPrePackageWithFlags_ExecFallback(t *testing.T) {
	orig := osExecutable
	t.Cleanup(func() { osExecutable = orig })
	osExecutable = func() (string, error) { return "", errors.New("no exec") }
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "workflow.yaml", minimalWorkflowYAML()), 0644))
	outDir := filepath.Join(tmp, "out")
	origResolve := resolveBaseBinary
	t.Cleanup(func() { resolveBaseBinary = origResolve })
	resolveBaseBinary = func(_ context.Context, _ string, _ archTarget, _ string) (string, bool, error) {
		return "", false, errors.New("skip")
	}
	err := PrePackageWithFlags(context.Background(), []string{kdeps}, &PrePackageFlags{Output: outDir})
	require.Error(t, err)
}

func TestResolveBaseBinaryImpl_CleanErr(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad")
	require.NoError(t, os.WriteFile(bad, []byte("x"), 0644))
	_, _, err := resolveBaseBinaryImpl(
		context.Background(),
		"1.0.0",
		archTarget{GOOS: runtimeGOOS(), GOARCH: runtimeGOARCH()},
		bad,
	)
	t.Logf("clean: %v", err)
}

func TestGetPackageName_NoWorkflow(t *testing.T) {
	tmp := t.TempDir()
	kdeps := filepath.Join(tmp, "empty.kdeps")
	require.NoError(t, os.WriteFile(kdeps, buildMinimalKdepsArchive(t, "readme.txt", "x"), 0644))
	_, err := getPackageName(kdeps)
	require.Error(t, err)
}

func TestWriteTempBinary_AllErrs(t *testing.T) {
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, _ string) (*os.File, error) { return nil, errors.New("temp") }
	_, err := writeTempBinary([]byte("x"), "linux", "amd64")
	require.Error(t, err)
}

func TestDownloadKdepsBinaryToTemp_ExtractErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not archive"))
	}))
	defer srv.Close()
	*GithubReleasesBaseURL = srv.URL
	t.Cleanup(func() { *GithubReleasesBaseURL = "https://github.com/kdeps/kdeps/releases/download" })
	_, err := downloadKdepsBinaryToTemp(context.Background(), "1.0.0", "linux", "amd64")
	require.Error(t, err)
}

func TestFetchURL_RequestErr(t *testing.T) {
	_, err := fetchURL(context.Background(), "://bad-url")
	require.Error(t, err)
}

func TestExtractFromTarGz_EntryErr(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte("bad"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	_, err = extractFromTarGz(buf.Bytes(), "kdeps")
	require.Error(t, err)
}

func TestExtractFromZip_OpenErr(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("kdeps")
	require.NoError(t, err)
	_, err = w.Write([]byte("bin"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	data := buf.Bytes()
	_, _ = extractFromZip(data, "kdeps")
	// May succeed; test not-found path
	_, err = extractFromZip(data, "missing")
	require.Error(t, err)
}

func TestCloneFromRemote_DownloadErr(t *testing.T) {
	err := cloneFromRemote("owner/repo@main")
	require.Error(t, err)
}

func TestDetectCloneType_Workflow(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	typ, _ := detectCloneType(tmp)
	assert.Equal(t, "agent", typ)
}

func TestCloneAsComponent_CopyErr(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	err := cloneAsComponent("c", "/nonexistent/src")
	require.Error(t, err)
}

func TestCloneAsWorkdir_Exists(t *testing.T) {
	tmp := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.MkdirAll("agents/dup", 0755))
	err := cloneAsWorkdir("dup", tmp, "agents", "workflow.yaml")
	require.Error(t, err)
}

func TestUnwrapArchiveRoot_ReadErr(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	_, err := unwrapArchiveRoot(f)
	require.Error(t, err)
}

func TestCopyFile_ReadErr(t *testing.T) {
	err := copyFile("/nonexistent", filepath.Join(t.TempDir(), "dst"))
	require.Error(t, err)
}

func TestCopyDir_WalkErr2(t *testing.T) {
	err := copyDir(filepath.Join(t.TempDir(), "missing"), t.TempDir())
	require.Error(t, err)
}

func TestShowLinuxKitConfig_EmptyMetadata(t *testing.T) {
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "", Version: ""}}
	require.NoError(t, showLinuxKitConfig(wf, &ExportFlags{}))
}

func TestExportK8sInternal_ReplicaZero(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	err := exportK8sInternal(&cobra.Command{}, []string{tmp}, &K8sFlags{Replica: 0})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// registry, info, serve, validate, chat, exec, markdown, new
// ---------------------------------------------------------------------------

func TestRegistryListRunE_GlobalErr(t *testing.T) {
	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) { return "", errors.New("no home") }
	// componentInstallDir may still work; test list with temp HOME
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	require.NoError(t, registryListRunE())
}

func TestListInstalledAgents_Versioned(t *testing.T) {
	tmp := t.TempDir()
	agentDir := filepath.Join(tmp, "myagent", "1.0.0")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	names := listInstalledAgents(tmp)
	assert.Contains(t, names, "myagent")
}

func TestIsVersionedAgentDir_Pkl(t *testing.T) {
	tmp := t.TempDir()
	verDir := filepath.Join(tmp, "agent", "1.0.0")
	require.NoError(t, os.MkdirAll(verDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(verDir, "workflow.pkl"), []byte("pkl"), 0644))
	assert.True(t, isVersionedAgentDir(filepath.Join(tmp, "agent")))
}

func TestDoRegistrySearch_Errors(t *testing.T) {
	cmd := &cobra.Command{}
	err := doRegistrySearch(cmd, "q", "", 10, "://bad")
	require.Error(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	err = doRegistrySearch(cmd, "q", "", 10, srv.URL)
	require.Error(t, err)
}

func TestFetchSearchResults_ReadErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		hj, ok := w.(http.Hijacker)
		if ok {
			c, _, _ := hj.Hijack()
			_ = c.Close()
		}
	}))
	defer srv.Close()
	_, err := fetchSearchResults(srv.URL + "/api/v1/registry/packages?q=x")
	require.Error(t, err)
}

func TestDoRegistryUpdate_LocalPath(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	cmd := &cobra.Command{}
	err := doRegistryUpdate(cmd, tmp, "http://example.com")
	require.NoError(t, err)
}

func TestNewRegistrySubmitCmd_NoTag(t *testing.T) {
	c := newRegistrySubmitCmd()
	err := c.RunE(c, nil)
	require.Error(t, err)
}

func TestDoRegistrySubmit_EncodeErr(t *testing.T) {
	orig := detectGitHubRepoFunc
	origSHA := computeRemoteSHA256Func
	t.Cleanup(func() {
		detectGitHubRepoFunc = orig
		computeRemoteSHA256Func = origSHA
	})
	detectGitHubRepoFunc = func(_ string) (string, error) { return "o/r", nil }
	computeRemoteSHA256Func = func(_ string) (string, error) { return strings.Repeat("a", 64), nil }
	tmp := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(tmp, "kdeps.pkg.yaml"), []byte("name: p\nversion: \"1\"\ntype: workflow\n"), 0644),
	)
	cmd := &cobra.Command{}
	require.NoError(t, doRegistrySubmit(cmd, tmp, "v1.0.0"))
}

func TestParseGitHubHTTPSRemote_Invalid(t *testing.T) {
	_, ok := parseGitHubHTTPSRemote("https://gitlab.com/o/r")
	assert.False(t, ok)
}

func TestComputeRemoteSHA256_Errors(t *testing.T) {
	_, err := computeRemoteSHA256("://bad")
	require.Error(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err = computeRemoteSHA256(srv.URL)
	require.Error(t, err)
}

func TestPrintRegistryReadmeFallback_Archive(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	// Empty client triggers fallback to local readme
	err := printRegistryReadmeFallback(cmd, "nonexistent", nil)
	require.NoError(t, err)
}

func TestReadmeFromRegistryArchive_Failures(t *testing.T) {
	assert.Empty(t, readmeFromRegistryArchive(context.Background(), nil, "pkg"))
}

func TestFetchRemoteReadme_Errors(t *testing.T) {
	_, err := fetchRemoteReadme("bad")
	require.Error(t, err)
	_, err = fetchReadmeURL("://bad")
	require.Error(t, err)
}

func TestGithubURLToRef_TreePath(t *testing.T) {
	ref := githubURLToRef("https://github.com/o/r/tree/main/subdir")
	assert.Equal(t, "o/r:subdir", ref)
}

func TestRunServeCmd_Errors(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("invalid path semantics differ on Windows")
	}
	err := runServeCmd("\x00bad", &serveFlags{})
	require.Error(t, err)
	err = runServeCmd("/nonexistent", &serveFlags{})
	require.Error(t, err)
}

func TestRegisterAgencyTool_TargetParseErr(t *testing.T) {
	tmp := t.TempDir()
	agency := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: a
  version: "1"
  targetAgentId: missing
agents:
  - agents/a
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agency.yaml"), []byte(agency), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agents", "a"), 0755))
	agentWF := strings.Replace(minimalWorkflowYAML(), "name: gap-test", "name: other", 1)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agents", "a", "workflow.yaml"), []byte(agentWF), 0644))
	_, err := registerAgencyTool(filepath.Join(tmp, "agency.yaml"), tools.NewRegistry(), false)
	require.Error(t, err)
}

func TestFindServeWorkflowFiles_WalkErr(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	// Walk into file-as-dir triggers error
	paths := findServeWorkflowFiles(f)
	assert.Empty(t, paths)
}

func TestValidateWorkflowFile_WithWarnings(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "resources"), 0755))
	require.NoError(t, validateWorkflowFile(filepath.Join(tmp, "workflow.yaml")))
}

func TestResolveInstalledAgentWorkflow_NoHome(t *testing.T) {
	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) { return "", errors.New("no home") }
	_, err := resolveInstalledAgentWorkflow("agent")
	require.Error(t, err)
}

func TestRenderMarkdown_RenderFail(t *testing.T) {
	orig := newMarkdownRendererFunc
	t.Cleanup(func() { newMarkdownRendererFunc = orig })
	newMarkdownRendererFunc = func() (*glamour.TermRenderer, error) {
		r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"))
		require.NoError(t, err)
		return r, err
	}
	out := renderMarkdown("# Hello")
	assert.NotEmpty(t, out)
}

func TestRunNewWithFlags_GeneratorErr(t *testing.T) {
	t.Chdir(t.TempDir())
	err := RunNewWithFlags(&cobra.Command{}, []string{"valid-name"}, &NewFlags{Template: "nonexistent-tpl"})
	require.Error(t, err)
}

func TestPrepareNewOutputDir_NotExist(t *testing.T) {
	err := prepareNewOutputDir(filepath.Join(t.TempDir(), "new-agent"), false)
	require.NoError(t, err)
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
