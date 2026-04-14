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
// license honours and attribution when redistributing derived code.

//go:build !js

package cmd_test

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cmd "github.com/kdeps/kdeps/v2/cmd"
)

// buildTarGz creates an in-memory tar.gz archive with the given files and
// returns it as a byte slice. The files are placed under a top-level wrapper
// dir to simulate GitHub's archive format.
func buildTarGz(t *testing.T, wrapperDir string, files map[string]string) []byte {
	t.Helper()
	tmpDir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(tmpDir, wrapperDir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}

	// Write tar.gz to a temp file, then read back.
	archivePath := filepath.Join(tmpDir, "archive.tar.gz")
	f, err := os.Create(archivePath)
	require.NoError(t, err)

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	err = filepath.WalkDir(filepath.Join(tmpDir, wrapperDir), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(tmpDir, path)
		if d.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     rel + "/",
				Mode:     0o755,
			})
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		hdrErr := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     rel,
			Size:     int64(len(data)),
			Mode:     0o644,
		})
		if hdrErr != nil {
			return hdrErr
		}
		_, writeErr := tw.Write(data)
		return writeErr
	})
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())

	data, readErr := os.ReadFile(archivePath)
	require.NoError(t, readErr)
	return data
}

// --- clone tests ---

func TestCloneFromRemote_InvalidRef(t *testing.T) {
	err := cmd.CloneFromRemote("noslash")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected owner/repo")
}

func TestCloneFromRemote_AgentCloned(t *testing.T) {
	archiveData := buildTarGz(t, "my-agent-abc123", map[string]string{
		"workflow.yaml": "apiVersion: kdeps.io/v1\nkind: Workflow\n",
		"README.md":     "# My Agent\n",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := cmd.CloneFromRemote("owner/my-agent")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "agents", "my-agent", "workflow.yaml"))
}

func TestCloneFromRemote_AgencyCloned(t *testing.T) {
	archiveData := buildTarGz(t, "my-agency-abc123", map[string]string{
		"agency.yml": "apiVersion: kdeps.io/v1\nkind: Agency\n",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := cmd.CloneFromRemote("owner/my-agency")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "agencies", "my-agency", "agency.yml"))
}

func TestCloneFromRemote_WithSubdir(t *testing.T) {
	archiveData := buildTarGz(t, "repo-abc123", map[string]string{
		"my-scraper/workflow.yaml": "apiVersion: kdeps.io/v1\nkind: Workflow\n",
		"other/unrelated.txt":      "ignore me",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := cmd.CloneFromRemote("owner/repo:my-scraper")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "agents", "my-scraper", "workflow.yaml"))
}

func TestCloneFromRemote_AlreadyExists(t *testing.T) {
	archiveData := buildTarGz(t, "repo-abc123", map[string]string{
		"workflow.yaml": "apiVersion: kdeps.io/v1\nkind: Workflow\n",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "agents", "repo"), 0o755))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := cmd.CloneFromRemote("owner/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// --- detect clone type ---

func TestDetectCloneType_Agency(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agency.yml"), []byte(""), 0o644))
	typ, manifest := cmd.DetectCloneType(dir)
	assert.Equal(t, "agency", typ)
	assert.Equal(t, "agency.yml", manifest)
}

func TestDetectCloneType_Agent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(""), 0o644))
	typ, manifest := cmd.DetectCloneType(dir)
	assert.Equal(t, "agent", typ)
	assert.Equal(t, "workflow.yaml", manifest)
}

func TestDetectCloneType_Component(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.yaml"), []byte(""), 0o644))
	typ, manifest := cmd.DetectCloneType(dir)
	assert.Equal(t, "component", typ)
	assert.Equal(t, "component.yaml", manifest)
}

func TestDetectCloneType_Unknown(t *testing.T) {
	typ, manifest := cmd.DetectCloneType(t.TempDir())
	assert.Equal(t, "", typ)
	assert.Equal(t, "", manifest)
}

// ---------------------------------------------------------------------------
// cloneAsComponent tests
// ---------------------------------------------------------------------------

func TestCloneAsComponent_WithKomponent(t *testing.T) {
	// Source dir contains a pre-built .komponent archive.
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "mycomp.komponent"), []byte("fake"), 0o644))

	installDir := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", installDir)

	require.NoError(t, cmd.CloneAsComponent("mycomp", srcDir))
	assert.FileExists(t, filepath.Join(installDir, "mycomp.komponent"))
}

func TestCloneAsComponent_DirectoryCopy(t *testing.T) {
	// Source dir has no .komponent file — copy whole directory.
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "component.yaml"),
		[]byte("apiVersion: kdeps.io/v1\n"),
		0o644,
	))

	installDir := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", installDir)

	require.NoError(t, cmd.CloneAsComponent("mycomp", srcDir))
	assert.FileExists(t, filepath.Join(installDir, "mycomp", "component.yaml"))
}

// ---------------------------------------------------------------------------
// findFileWithSuffix tests
// ---------------------------------------------------------------------------

func TestFindFileWithSuffix_Found(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.komponent"), []byte("x"), 0o644))
	result := cmd.FindFileWithSuffix(dir, ".komponent")
	assert.Equal(t, filepath.Join(dir, "foo.komponent"), result)
}

func TestFindFileWithSuffix_NotFound(t *testing.T) {
	result := cmd.FindFileWithSuffix(t.TempDir(), ".komponent")
	assert.Empty(t, result)
}

func TestFindFileWithSuffix_NonExistentDir(t *testing.T) {
	result := cmd.FindFileWithSuffix("/nonexistent/path", ".komponent")
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// Clone component type
// ---------------------------------------------------------------------------

func TestCloneFromRemote_ComponentType(t *testing.T) {
	archiveData := buildTarGz(t, "my-comp-abc123", map[string]string{
		"component.yaml":    "apiVersion: kdeps.io/v1\nkind: Component\nmetadata:\n  name: my-comp\n",
		"my-comp.komponent": "fake",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	installDir := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", installDir)

	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := cmd.CloneFromRemote("owner/my-comp")
	require.NoError(t, err)
	// Component should be installed into KDEPS_COMPONENT_DIR.
	assert.FileExists(t, filepath.Join(installDir, "my-comp.komponent"))
}

// ---------------------------------------------------------------------------
// downloadAndExtract / downloadFileTo edge cases
// ---------------------------------------------------------------------------

func TestDownloadAndExtract_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	err := cmd.CloneFromRemote("owner/repo")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// registry install GitHub path (formerly newRegistryCloneCmd) cobra RunE coverage
// ---------------------------------------------------------------------------

func TestRegistryInstall_GitHubRef_Execute(t *testing.T) {
	archiveData := buildTarGz(t, "clonetest-abc123", map[string]string{
		"workflow.yaml": "apiVersion: kdeps.io/v1\nkind: Workflow\n",
	})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(archiveData)
	}))
	defer ts.Close()

	origArchive := *cmd.GithubArchiveBaseURL
	*cmd.GithubArchiveBaseURL = ts.URL
	defer func() { *cmd.GithubArchiveBaseURL = origArchive }()

	dir := t.TempDir()
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// Exercise via exported DoRegistryInstall (GitHub ref path).
	c := cobra.Command{}
	err := cmd.DoRegistryInstall(&c, "owner/clonetest", "http://unused")
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dir, "agents", "clonetest", "workflow.yaml"))
}

func TestRegistryInstall_GitHubRef_ExecuteError(t *testing.T) {
	// A ref without "/" should go to registry path, not GitHub path.
	// Use an unreachable registry URL to confirm it hits the registry branch.
	c := cobra.Command{}
	err := cmd.DoRegistryInstall(&c, "badref-no-slash", "http://127.0.0.1:1")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// unwrapArchiveRoot edge cases
// ---------------------------------------------------------------------------

func TestUnwrapArchiveRoot_SingleDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "repo-abc123")
	require.NoError(t, os.Mkdir(sub, 0o755))

	result, err := cmd.UnwrapArchiveRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, sub, result)
}

func TestUnwrapArchiveRoot_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "dir1"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "dir2"), 0o755))

	result, err := cmd.UnwrapArchiveRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, result)
}

func TestUnwrapArchiveRoot_FileEntry(t *testing.T) {
	// Single file (not dir) - falls through to returning dir itself.
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644))

	result, err := cmd.UnwrapArchiveRoot(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, result)
}
