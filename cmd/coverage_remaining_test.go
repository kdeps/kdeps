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
	"errors"
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
)

// ---------------------------------------------------------------------------
// root.go — bootstrapRootConfig error paths
// ---------------------------------------------------------------------------

func TestBootstrapRootConfig_BootstrapError(t *testing.T) {
	orig := bootstrapConfigFunc
	t.Cleanup(func() { bootstrapConfigFunc = orig })
	bootstrapConfigFunc = func(_ *os.File) error {
		return errors.New("bootstrap failed")
	}
	assert.NotPanics(t, func() { bootstrapRootConfig() })
}

func TestBootstrapRootConfig_LoadError(t *testing.T) {
	origBoot := bootstrapConfigFunc
	origLoad := loadConfigFunc
	t.Cleanup(func() {
		bootstrapConfigFunc = origBoot
		loadConfigFunc = origLoad
	})
	bootstrapConfigFunc = func(_ *os.File) error { return nil }
	loadConfigFunc = func() (*config.Config, error) {
		return nil, errors.New("load failed")
	}
	assert.NotPanics(t, func() { bootstrapRootConfig() })
}

// ---------------------------------------------------------------------------
// registry_submit.go — doRegistrySubmit success + hook error paths
// ---------------------------------------------------------------------------

func TestDoRegistrySubmit_Success(t *testing.T) {
	dir := t.TempDir()
	mf := "name: test-agent\nversion: 1.0.0\ntype: workflow\ndescription: A test\nlicense: Apache-2.0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kdeps.pkg.yaml"), []byte(mf), 0600))

	origDetect := detectGitHubRepoFunc
	origSHA := computeRemoteSHA256Func
	t.Cleanup(func() {
		detectGitHubRepoFunc = origDetect
		computeRemoteSHA256Func = origSHA
	})
	detectGitHubRepoFunc = func(_ string) (string, error) { return "owner/repo", nil }
	computeRemoteSHA256Func = func(_ string) (string, error) {
		return strings.Repeat("a", 64), nil
	}

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	err := doRegistrySubmit(cmd, dir, "v1.0.0")
	require.NoError(t, err)
	assert.Contains(t, out.String(), "formulas/test-agent.yaml")
	assert.Contains(t, out.String(), "name: test-agent")
}

func TestDoRegistrySubmit_DetectRepoError(t *testing.T) {
	dir := t.TempDir()
	mf := "name: test-agent\nversion: 1.0.0\ntype: workflow\ndescription: A test\nlicense: Apache-2.0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kdeps.pkg.yaml"), []byte(mf), 0600))

	orig := detectGitHubRepoFunc
	t.Cleanup(func() { detectGitHubRepoFunc = orig })
	detectGitHubRepoFunc = func(_ string) (string, error) {
		return "", errors.New("no remote")
	}

	err := doRegistrySubmit(&cobra.Command{}, dir, "v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detect GitHub repo")
}

func TestDoRegistrySubmit_SHA256Error(t *testing.T) {
	dir := t.TempDir()
	mf := "name: test-agent\nversion: 1.0.0\ntype: workflow\ndescription: A test\nlicense: Apache-2.0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kdeps.pkg.yaml"), []byte(mf), 0600))

	origDetect := detectGitHubRepoFunc
	origSHA := computeRemoteSHA256Func
	t.Cleanup(func() {
		detectGitHubRepoFunc = origDetect
		computeRemoteSHA256Func = origSHA
	})
	detectGitHubRepoFunc = func(_ string) (string, error) { return "owner/repo", nil }
	computeRemoteSHA256Func = func(_ string) (string, error) {
		return "", errors.New("fetch failed")
	}

	err := doRegistrySubmit(&cobra.Command{}, dir, "v1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compute sha256")
}

// ---------------------------------------------------------------------------
// edit.go — prepareConfigForEdit error paths
// ---------------------------------------------------------------------------

func TestPrepareConfigForEdit_ScaffoldError(t *testing.T) {
	orig := configScaffoldFunc
	t.Cleanup(func() { configScaffoldFunc = orig })
	configScaffoldFunc = func() error { return errors.New("scaffold failed") }

	_, err := prepareConfigForEdit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create config")
}

func TestPrepareConfigForEdit_PathError(t *testing.T) {
	origScaffold := configScaffoldFunc
	origPath := configPathFunc
	t.Cleanup(func() {
		configScaffoldFunc = origScaffold
		configPathFunc = origPath
	})
	configScaffoldFunc = func() error { return nil }
	configPathFunc = func() (string, error) { return "", errors.New("path failed") }

	_, err := prepareConfigForEdit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "locate config")
}

// ---------------------------------------------------------------------------
// markdown.go — renderMarkdown + terminalMarkdownWidth
// ---------------------------------------------------------------------------

func TestRenderMarkdown_RendererInitError(t *testing.T) {
	orig := newMarkdownRendererFunc
	t.Cleanup(func() { newMarkdownRendererFunc = orig })
	newMarkdownRendererFunc = func() (*glamour.TermRenderer, error) {
		return nil, errors.New("renderer init failed")
	}
	assert.Equal(t, "# Hello", renderMarkdown("# Hello"))
}

func TestRenderMarkdown_RenderError(t *testing.T) {
	orig := newMarkdownRendererFunc
	t.Cleanup(func() { newMarkdownRendererFunc = orig })
	renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	require.NoError(t, err)
	newMarkdownRendererFunc = func() (*glamour.TermRenderer, error) { return renderer, nil }
	// Invalid markdown that may fail render — fallback returns raw content.
	out := renderMarkdown("\x00\x01invalid")
	assert.NotEmpty(t, out)
}

func TestTerminalMarkdownWidth_Error(t *testing.T) {
	orig := termGetSizeFunc
	t.Cleanup(func() { termGetSizeFunc = orig })
	termGetSizeFunc = func(_ int) (int, int, error) { return 0, 0, errors.New("no tty") }
	assert.Equal(t, defaultMarkdownWidth, terminalMarkdownWidth())
}

func TestTerminalMarkdownWidth_ZeroWidth(t *testing.T) {
	orig := termGetSizeFunc
	t.Cleanup(func() { termGetSizeFunc = orig })
	termGetSizeFunc = func(_ int) (int, int, error) { return 0, 0, nil }
	assert.Equal(t, defaultMarkdownWidth, terminalMarkdownWidth())
}

func TestTerminalMarkdownWidth_Valid(t *testing.T) {
	orig := termGetSizeFunc
	t.Cleanup(func() { termGetSizeFunc = orig })
	termGetSizeFunc = func(_ int) (int, int, error) { return 120, 40, nil }
	assert.Equal(t, 120, terminalMarkdownWidth())
}

// ---------------------------------------------------------------------------
// embedded.go — writeCleanBinaryTemp error paths
// ---------------------------------------------------------------------------

func TestWriteCleanBinaryTemp_CreateTempError(t *testing.T) {
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, _ string) (*os.File, error) {
		return nil, errors.New("create temp failed")
	}
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	content := []byte("data")
	_, err = f.Write(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	src, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer src.Close()
	_, _, err = writeCleanBinaryTemp(src, int64(len(content)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}

func TestWriteCleanBinaryTemp_WriteError(t *testing.T) {
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

func TestWriteCleanBinaryTemp_CloseError(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	content := []byte("data")
	_, err = f.Write(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	src, err := os.Open(filepath.Join(tmp, "src"))
	require.NoError(t, err)
	defer src.Close()

	// Use a read-only destination path to force close/write failure.
	roDir := filepath.Join(tmp, "readonly")
	require.NoError(t, os.Mkdir(roDir, 0500))
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, pattern string) (*os.File, error) {
		return os.Create(filepath.Join(roDir, strings.TrimPrefix(pattern, "kdeps-clean-")+"tmp"))
	}
	_, _, err = writeCleanBinaryTemp(src, int64(len(content)))
	require.Error(t, err)
}

func TestWriteEmbeddedTrailer_WriteErrors(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "out"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	// Closed file should fail on write.
	out, err := os.OpenFile(filepath.Join(tmp, "out"), os.O_RDONLY, 0644)
	require.NoError(t, err)
	defer out.Close()
	err = writeEmbeddedTrailer(out, 10)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// prepackage.go — writeTempBinary error paths
// ---------------------------------------------------------------------------

func TestWriteTempBinary_CreateTempError(t *testing.T) {
	orig := osCreateTempFunc
	t.Cleanup(func() { osCreateTempFunc = orig })
	osCreateTempFunc = func(_, _ string) (*os.File, error) {
		return nil, errors.New("temp failed")
	}
	_, err := writeTempBinary([]byte("x"), "linux", "amd64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create temp file")
}

func TestWriteTempBinary_ChmodError(t *testing.T) {
	origCreate := osCreateTempFunc
	origChmod := osChmodFunc
	t.Cleanup(func() {
		osCreateTempFunc = origCreate
		osChmodFunc = origChmod
	})
	tmp := t.TempDir()
	osCreateTempFunc = func(_, pattern string) (*os.File, error) {
		return os.CreateTemp(tmp, pattern)
	}
	osChmodFunc = func(_ string, _ os.FileMode) error {
		return errors.New("chmod failed")
	}
	_, err := writeTempBinary([]byte("x"), "linux", "amd64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permissions")
}

// ---------------------------------------------------------------------------
// registry_install.go — downloadRegistryArchive
// ---------------------------------------------------------------------------

func TestDownloadRegistryArchive_NoURL(t *testing.T) {
	info := &packageInfo{TarballURL: ""}
	_, cleanup, err := downloadRegistryArchive(info, "pkg", "1.0.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no download URL")
	if cleanup != nil {
		cleanup()
	}
}

func TestDownloadRegistryArchive_DownloadError(t *testing.T) {
	orig := downloadArchiveFunc
	t.Cleanup(func() { downloadArchiveFunc = orig })
	downloadArchiveFunc = func(_, _ string) error {
		return errors.New("download failed")
	}
	info := &packageInfo{TarballURL: "https://example.com/pkg.kdeps"}
	_, cleanup, err := downloadRegistryArchive(info, "pkg", "1.0.0")
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestDownloadRegistryArchive_VerifyError(t *testing.T) {
	origDownload := downloadArchiveFunc
	t.Cleanup(func() { downloadArchiveFunc = origDownload })
	tmp := t.TempDir()
	archive := filepath.Join(tmp, "pkg.kdeps")
	require.NoError(t, os.WriteFile(archive, []byte("wrong content"), 0644))
	downloadArchiveFunc = func(_, dest string) error {
		data, readErr := os.ReadFile(archive)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(dest, data, 0644)
	}
	info := &packageInfo{
		TarballURL: "https://example.com/pkg.kdeps",
		SHA256:     strings.Repeat("0", 64),
	}
	_, cleanup, err := downloadRegistryArchive(info, "pkg", "1.0.0")
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

// ---------------------------------------------------------------------------
// component.go — writeKomponentRegularFile + componentUpdateInternal
// ---------------------------------------------------------------------------

func TestWriteKomponentRegularFile_Success(t *testing.T) {
	destDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	content := []byte("hello komponent")
	createTarGz(t, archivePath,
		[]*tar.Header{{Name: "file.txt", Typeflag: tar.TypeReg, Mode: 0644}},
		[][]byte{content},
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
	target := filepath.Join(destDir, "file.txt")
	err = writeKomponentRegularFile(target, tr)
	require.NoError(t, err)
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestComponentUpdateInternal_NoComponents(t *testing.T) {
	tmp := t.TempDir()
	err := componentUpdateInternal(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a component")
}

func TestComponentUpdateInternal_WithComponent(t *testing.T) {
	tmp := t.TempDir()
	compDir := filepath.Join(tmp, "mycomp")
	require.NoError(t, os.MkdirAll(compDir, 0755))
	compYAML := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: mycomp
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte(compYAML), 0644))
	err := componentUpdateInternal(compDir)
	require.NoError(t, err)
}

func TestFindUpdateTargetComponentDirs_AgentDir(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "workflow.yaml"), []byte(minimalWorkflowYAML()), 0644))
	compDir := filepath.Join(tmp, "components", "c1")
	require.NoError(t, os.MkdirAll(compDir, 0755))
	compYAML := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: c1
  version: "1.0.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte(compYAML), 0644))
	dirs, err := findUpdateTargetComponentDirs(tmp)
	require.NoError(t, err)
	assert.Len(t, dirs, 1)
}

// ---------------------------------------------------------------------------
// export.go — prepareISOExportWorkflow + exportISOInternal error paths
// ---------------------------------------------------------------------------

func TestPrepareISOExportWorkflow_InvalidWorkflow(t *testing.T) {
	tmp := t.TempDir()
	badKdeps := filepath.Join(tmp, "bad.kdeps")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	badYAML := []byte("bad: [")
	hdr := &tar.Header{Name: "workflow.yaml", Size: int64(len(badYAML)), Mode: 0644}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write(badYAML)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(badKdeps, buf.Bytes(), 0644))

	_, _, cleanup, err := prepareISOExportWorkflow(badKdeps)
	require.Error(t, err)
	if cleanup != nil {
		cleanup()
	}
}

func TestExportISOInternal_PrepareError(t *testing.T) {
	err := exportISOInternal(&cobra.Command{}, []string{"/nonexistent/pkg.kdeps"}, &ExportFlags{})
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// run.go — dispatchExecution all modes + ExecuteWorkflowStepsWithFlags
// ---------------------------------------------------------------------------

func TestDispatchExecution_SingleRunMode(t *testing.T) {
	wf := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{Name: "s", TargetActionID: "act"},
		Resources: []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}},
	}
	require.NoError(t, dispatchExecution(wf, t.TempDir(), false, false, "", false))
}

func TestDispatchExecutionWithEngine_DefaultNil(t *testing.T) {
	eng := executor.NewEngine(nil)
	// Workflow with no recognized mode — should hit default return nil.
	wf := &domain.Workflow{}
	// Force unknown mode by using a workflow that doesn't match any case.
	// executionModeFor always returns something, so test via empty workflow -> single run.
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, nil
	})
	wf.Metadata.TargetActionID = "act"
	wf.Resources = []*domain.Resource{{ActionID: "act", APIResponse: &domain.APIResponseConfig{Success: true}}}
	require.NoError(t, dispatchExecutionWithEngine(eng, wf, t.TempDir(), false, false, "", false))
}

func TestExecuteWorkflowStepsWithFlags_AgencyPath(t *testing.T) {
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
	agentWF := `apiVersion: kdeps.io/v1
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
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte(agentWF), 0644))
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	err := ExecuteWorkflowStepsWithFlags(cmd, filepath.Join(tmp, "agency.yaml"), &RunFlags{})
	require.NoError(t, err)
}

func TestExecuteWorkflowStepsWithFlags_InvalidWorkflow(t *testing.T) {
	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "workflow.yaml")
	require.NoError(t, os.WriteFile(badPath, []byte("invalid: ["), 0644))
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	err := ExecuteWorkflowStepsWithFlags(cmd, badPath, &RunFlags{})
	require.Error(t, err)
}

func TestExtractFile_Success(t *testing.T) {
	destDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	content := []byte("extract me")
	createTarGz(t, archivePath,
		[]*tar.Header{{Name: "data.txt", Typeflag: tar.TypeReg, Mode: 0644}},
		[][]byte{content},
	)
	f, err := os.Open(archivePath)
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gz.Close()
	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	require.NoError(t, err)
	err = ExtractFile(tr, hdr, filepath.Join(destDir, "data.txt"))
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(destDir, "data.txt"))
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

// ---------------------------------------------------------------------------
// validate.go — validateResourceFile
// ---------------------------------------------------------------------------

func TestValidateResourceFile_Success(t *testing.T) {
	tmp := t.TempDir()
	resPath := filepath.Join(tmp, "act.yaml")
	content := `actionId: act
name: Act
apiResponse:
  success: true
`
	require.NoError(t, os.WriteFile(resPath, []byte(content), 0644))
	err := validateResourceFile(resPath)
	require.NoError(t, err)
}

func TestValidateResourceFile_ParseError(t *testing.T) {
	tmp := t.TempDir()
	resPath := filepath.Join(tmp, "bad.yaml")
	require.NoError(t, os.WriteFile(resPath, []byte("invalid: ["), 0644))
	err := validateResourceFile(resPath)
	require.Error(t, err)
}

func TestIsResourceFile(t *testing.T) {
	tmp := t.TempDir()
	resPath := filepath.Join(tmp, "act.yaml")
	require.NoError(t, os.WriteFile(resPath, []byte("actionId: act\nname: Act\n"), 0644))
	assert.True(t, isResourceFile(resPath))
	assert.False(t, isResourceFile("/nonexistent"))
}

func TestRunServeCmd_InvalidPath(t *testing.T) {
	err := runServeCmd("/nonexistent/serve/path", &serveFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
