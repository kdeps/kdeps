// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package llm

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripperFunc adapts a function to http.RoundTripper for test mocks.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// stubLlamaServerAsset overrides resolveLlamaServerAssetFn to return a fixed
// zip-suffixed URL without a real GitHub API call, restoring the original on
// cleanup. Used by tests that only care about the download/extract steps.
func stubLlamaServerAsset(t *testing.T, url string) {
	t.Helper()
	orig := resolveLlamaServerAssetFn
	t.Cleanup(func() { resolveLlamaServerAssetFn = orig })
	resolveLlamaServerAssetFn = func(string) (string, error) { return url, nil }
}

func TestGGufLlamaServerBinary_EnvVar(t *testing.T) {
	t.Setenv("KDEPS_LLAMA_SERVER_BIN", "/custom/path/llama-server")
	result := ggufLlamaServerBinary()
	if result != "/custom/path/llama-server" {
		t.Errorf("expected /custom/path/llama-server, got %q", result)
	}
}

func TestGGufLlamaServerBinary_NoEnvVar(t *testing.T) {
	t.Setenv("KDEPS_LLAMA_SERVER_BIN", "")
	result := ggufLlamaServerBinary()
	// Without env var, falls through to ensureLlamaServerBinary
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestCachedLlamaServerPath_Default(t *testing.T) {
	t.Setenv("HOME", "/test/home/gguf")
	result := cachedLlamaServerPath()
	expected := filepath.Join("/test/home/gguf", ".kdeps", "bin", "llama-server")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractZipFile_ValidZip(t *testing.T) {
	dir := t.TempDir()
	destDir := filepath.Join(dir, "extracted")

	// Create a valid zip in memory with llama-server binary
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("llama-server")
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("fake-binary-content"))
	zw.Close()

	zipPath := filepath.Join(dir, "test.zip")
	err = os.WriteFile(zipPath, buf.Bytes(), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = extractZipFile(zipPath, destDir)
	if err != nil {
		t.Fatalf("extractZipFile failed: %v", err)
	}

	_, err = os.Stat(destDir)
	if os.IsNotExist(err) {
		t.Fatal("expected extracted file to exist")
	}
}

func TestExtractZipFile_MissingFile(t *testing.T) {
	err := extractZipFile("/nonexistent/zip/file.zip", t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent zip file")
	}
}

func TestExtractZipFile_InvalidZipContent(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "corrupt.zip")
	err := os.WriteFile(zipPath, []byte("not-a-zip-file"), 0600)
	require.NoError(t, err)

	err = extractZipFile(zipPath, filepath.Join(dir, "out"))
	require.Error(t, err)
}

func TestExtractZipFile_NoLlamaServerBinary(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("some-other-file.txt")
	require.NoError(t, err)
	_, err = f.Write([]byte("not the binary"))
	require.NoError(t, err)
	zw.Close()

	zipPath := filepath.Join(dir, "test.zip")
	err = os.WriteFile(zipPath, buf.Bytes(), 0600)
	require.NoError(t, err)

	err = extractZipFile(zipPath, filepath.Join(dir, "out"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llama-server binary not found")
}

func TestEnsureLlamaServerBinary_Cached(t *testing.T) {
	dir := t.TempDir()
	cachedPath := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	if err := os.MkdirAll(filepath.Dir(cachedPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachedPath, []byte("fake-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", dir)
	result := ensureLlamaServerBinary()
	if result != cachedPath {
		t.Errorf("expected cached path %q, got %q", cachedPath, result)
	}
}

func TestEnsureLlamaServerBinary_InstallFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")

	// Mock HTTP client to make installLlamaServer fail
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("mock download failure")
	})

	result := ensureLlamaServerBinary()
	assert.Equal(t, "llama-server", result)
}

func TestDownloadFile_Success(t *testing.T) {
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("test-data"))),
			Header:     make(http.Header),
		}, nil
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, "output.zip")
	err := downloadFile(dest, "https://example.com/test.zip")
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "test-data", string(data))
}

func TestDownloadFile_TransportError(t *testing.T) {
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})

	err := downloadFile(t.TempDir()+"/out.zip", "https://example.com/fail.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestDownloadFile_Non200Status(t *testing.T) {
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     make(http.Header),
		}, nil
	})

	err := downloadFile(t.TempDir()+"/out.zip", "https://example.com/notfound.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "download returned 404")
}

func TestDownloadFile_CreateFileError(t *testing.T) {
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("data"))),
			Header:     make(http.Header),
		}, nil
	})

	err := downloadFile("/nonexistent-parent-dir-xyz/output.zip", "https://example.com/test.zip")
	require.Error(t, err)
}

func TestInstallLlamaServer_DownloadError(t *testing.T) {
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("download failed")
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	err := installLlamaServer(dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "download llama-server")
}

func TestInstallLlamaServer_ExtractError(t *testing.T) {
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte("not-a-zip"))),
			Header:     make(http.Header),
		}, nil
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0750))

	err := installLlamaServer(dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extract llama-server")
}

func TestInstallLlamaServer_Success(t *testing.T) {
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")
	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("llama-server")
	require.NoError(t, err)
	_, err = f.Write([]byte("fake-llama-server-binary"))
	require.NoError(t, err)
	zw.Close()

	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			Header:     make(http.Header),
		}, nil
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0750))

	err = installLlamaServer(dest)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "fake-llama-server-binary", string(data))

	// Regression: the installed binary must be executable (0600 cannot be
	// exec'd), or ServeModel's cmd.Start() fails with "permission denied".
	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode().Perm()&0100, "installed llama-server binary must have owner-execute permission")
}

func TestResolvedGGUFURL_ModelNotInCache(t *testing.T) {
	ReloadGGUFRegistry()
	t.Cleanup(ReloadGGUFRegistry)

	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	result := ResolvedGGUFURL("nonexistent-model-xyz")
	assert.Equal(t, "", result)
}

func TestResolvedGGUFURL_InMemoryHit(t *testing.T) {
	ReloadGGUFRegistry()
	t.Cleanup(ReloadGGUFRegistry)

	modelsDir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", modelsDir)

	path, ok := GGUFCachedPath("qwen3.5:4b", modelsDir)
	require.True(t, ok)

	servedGGUFsMu.Lock()
	servedGGUFs[path] = 19999
	servedGGUFsMu.Unlock()
	t.Cleanup(func() {
		servedGGUFsMu.Lock()
		delete(servedGGUFs, path)
		servedGGUFsMu.Unlock()
	})

	origDo := httpDefaultClientDo
	t.Cleanup(func() { httpDefaultClientDo = origDo })
	httpDefaultClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}

	result := ResolvedGGUFURL("qwen3.5:4b")
	assert.Equal(t, "http://127.0.0.1:19999", result)
}

func TestResolvedGGUFURL_CrossProcessHit(t *testing.T) {
	ReloadGGUFRegistry()
	t.Cleanup(ReloadGGUFRegistry)

	modelsDir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", modelsDir)

	path, ok := GGUFCachedPath("qwen3.5:4b", modelsDir)
	require.True(t, ok)

	portFile := path + ".port"
	err := os.WriteFile(portFile, []byte("19998"), 0600)
	require.NoError(t, err)

	origDo := httpDefaultClientDo
	t.Cleanup(func() { httpDefaultClientDo = origDo })
	httpDefaultClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}

	result := ResolvedGGUFURL("qwen3.5:4b")
	assert.Equal(t, "http://127.0.0.1:19998", result)
}

func TestResolvedGGUFURL_DefaultPortHit(t *testing.T) {
	ReloadGGUFRegistry()
	t.Cleanup(ReloadGGUFRegistry)

	modelsDir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", modelsDir)

	path, ok := GGUFCachedPath("qwen3.5:4b", modelsDir)
	require.True(t, ok)

	servedGGUFsMu.Lock()
	delete(servedGGUFs, path)
	servedGGUFsMu.Unlock()

	origDo := httpDefaultClientDo
	t.Cleanup(func() { httpDefaultClientDo = origDo })
	httpDefaultClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}

	result := ResolvedGGUFURL("qwen3.5:4b")
	assert.Equal(t, BackendGGUFHostURL, result)
}

func TestExtractZipFile_MkdirError(t *testing.T) {
	dir := t.TempDir()
	// Create a file where the parent directory should be, so MkdirAll fails.
	filePath := filepath.Join(dir, "a_file")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0600))

	destDir := filepath.Join(filePath, "llama-server")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("llama-server")
	require.NoError(t, err)
	_, err = f.Write([]byte("data"))
	require.NoError(t, err)
	zw.Close()

	zipPath := filepath.Join(dir, "test.zip")
	require.NoError(t, os.WriteFile(zipPath, buf.Bytes(), 0600))

	err = extractZipFile(zipPath, destDir)
	require.Error(t, err)
}

func TestCachedLlamaServerPath_HomeDirError(t *testing.T) {
	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) { return "", errors.New("no home") }

	result := cachedLlamaServerPath()
	assert.Equal(t, "", result)
}

func TestEnsureLlamaServerBinary_InstallSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")

	// Pre-create the parent dir so downloadFile's os.Create succeeds.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".kdeps", "bin"), 0750))

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("llama-server")
	require.NoError(t, err)
	_, err = f.Write([]byte("binary-data"))
	require.NoError(t, err)
	zw.Close()

	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			Header:     make(http.Header),
		}, nil
	})

	result := ensureLlamaServerBinary()
	expected := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	assert.Equal(t, expected, result)
}

func TestResolvedGGUFURL_ModelsDirError(t *testing.T) {
	ReloadGGUFRegistry()
	t.Cleanup(ReloadGGUFRegistry)

	t.Setenv("KDEPS_MODELS_DIR", "")

	orig := userHomeDirFunc
	t.Cleanup(func() { userHomeDirFunc = orig })
	userHomeDirFunc = func() (string, error) { return "", errors.New("no home") }

	result := ResolvedGGUFURL("qwen3.5:4b")
	assert.Equal(t, "", result)
}

func TestDownloadFile_InvalidRequest(t *testing.T) {
	err := downloadFile(t.TempDir()+"/out.zip", "\n")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build download request")
}

func TestLocalContextSize_Default(t *testing.T) {
	if localContextSize <= 0 {
		t.Error("expected positive default context size")
	}
}

func TestLocalContextSize_SetLocalContextSize(t *testing.T) {
	orig := localContextSize
	t.Cleanup(func() { localContextSize = orig })
	SetLocalContextSize(8192)
	if localContextSize != 8192 {
		t.Errorf("expected 8192, got %d", localContextSize)
	}
}

func TestDetectOSArch_LinuxAmd64(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "linux"
	testArch = "amd64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	result := detectOSArch()
	assert.Equal(t, "ubuntu-x64", result)
}

func TestDetectOSArch_LinuxArm64(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "linux"
	testArch = "arm64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	result := detectOSArch()
	assert.Equal(t, "ubuntu-arm64", result)
}

func TestDetectOSArch_DarwinAmd64(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "darwin"
	testArch = "amd64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	result := detectOSArch()
	assert.Equal(t, "macos-x64", result)
}

func TestDetectOSArch_WindowsAmd64(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "windows"
	testArch = "amd64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	result := detectOSArch()
	assert.Equal(t, "win-cpu-x64", result)
}

func TestDetectOSArch_Unsupported(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "plan9"
	testArch = "riscv64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	result := detectOSArch()
	assert.Equal(t, "", result)
}

func TestInstallLlamaServer_UnsupportedPlatform(t *testing.T) {
	origOS := testOS
	origArch := testArch
	testOS = "plan9"
	testArch = "riscv64"
	t.Cleanup(func() { testOS = origOS; testArch = origArch })
	err := installLlamaServer("/tmp/llama-server")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported platform")
}

func TestInstallLlamaServer_ChmodError(t *testing.T) {
	stubLlamaServerAsset(t, "https://example.com/llama-server.zip")
	dir := t.TempDir()
	dest := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0750))

	// Create a valid zip with llama-server binary.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	f, err := zw.Create("llama-server")
	require.NoError(t, err)
	_, err = f.Write([]byte("fake-binary-data"))
	require.NoError(t, err)
	zw.Close()

	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(buf.Bytes())),
			Header:     make(http.Header),
		}, nil
	})

	// Make dest directory read-only so Chmod on the extracted file fails.
	require.NoError(t, os.WriteFile(dest, []byte("placeholder"), 0400))
	_ = os.Remove(dest)
	// Create a non-writable parent condition: create the dest as a directory
	require.NoError(t, os.MkdirAll(dest, 0555))

	err = installLlamaServer(dest)
	require.Error(t, err)
	_ = os.Chmod(dest, 0755)
}

// Regression coverage for the stale-build-tag bug: llama.cpp cuts new releases
// under new build-number tags constantly, so resolveLlamaServerAsset must find
// the current release's asset by OS/arch suffix rather than assuming a fixed
// filename (which 404s the moment the tag rolls over).

func mockGithubReleaseJSON(t *testing.T, body string) {
	t.Helper()
	origTransport := githubReleaseHTTPClient.Transport
	t.Cleanup(func() { githubReleaseHTTPClient.Transport = origTransport })
	githubReleaseHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader([]byte(body))),
			Header:     make(http.Header),
		}, nil
	})
}

const fakeLlamaCppReleaseJSON = `{
  "assets": [
    {"name": "llama-b9852-bin-macos-arm64.tar.gz", "browser_download_url": "https://example.com/llama-b9852-bin-macos-arm64.tar.gz"},
    {"name": "llama-b9852-bin-macos-x64.tar.gz", "browser_download_url": "https://example.com/llama-b9852-bin-macos-x64.tar.gz"},
    {"name": "llama-b9852-bin-ubuntu-x64.tar.gz", "browser_download_url": "https://example.com/llama-b9852-bin-ubuntu-x64.tar.gz"},
    {"name": "llama-b9852-bin-ubuntu-arm64.tar.gz", "browser_download_url": "https://example.com/llama-b9852-bin-ubuntu-arm64.tar.gz"},
    {"name": "llama-b9852-bin-win-cpu-x64.zip", "browser_download_url": "https://example.com/llama-b9852-bin-win-cpu-x64.zip"},
    {"name": "llama-b9852-bin-win-cuda-12.4-x64.zip", "browser_download_url": "https://example.com/llama-b9852-bin-win-cuda-12.4-x64.zip"}
  ]
}`

func TestResolveLlamaServerAsset_FindsMatchByOSArchSuffix(t *testing.T) {
	mockGithubReleaseJSON(t, fakeLlamaCppReleaseJSON)

	url, err := resolveLlamaServerAsset("macos-arm64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/llama-b9852-bin-macos-arm64.tar.gz", url)
}

func TestResolveLlamaServerAsset_DoesNotDependOnBuildTag(t *testing.T) {
	// A future release under a different build tag (e.g. b12345 instead of
	// b9852) must still resolve correctly since matching is by suffix, not by
	// a hardcoded tag.
	mockGithubReleaseJSON(t, `{"assets": [
		{"name": "llama-b12345-bin-ubuntu-x64.tar.gz", "browser_download_url": "https://example.com/future-ubuntu-x64.tar.gz"}
	]}`)

	url, err := resolveLlamaServerAsset("ubuntu-x64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/future-ubuntu-x64.tar.gz", url)
}

func TestResolveLlamaServerAsset_WindowsPicksZip(t *testing.T) {
	mockGithubReleaseJSON(t, fakeLlamaCppReleaseJSON)

	url, err := resolveLlamaServerAsset("win-cpu-x64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/llama-b9852-bin-win-cpu-x64.zip", url)
}

func TestResolveLlamaServerAsset_NoMatchingAsset(t *testing.T) {
	mockGithubReleaseJSON(t, fakeLlamaCppReleaseJSON)

	_, err := resolveLlamaServerAsset("freebsd-x64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no llama-server asset found")
}

func TestResolveLlamaServerAsset_HTTPTransportError(t *testing.T) {
	origTransport := githubReleaseHTTPClient.Transport
	t.Cleanup(func() { githubReleaseHTTPClient.Transport = origTransport })
	githubReleaseHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})

	_, err := resolveLlamaServerAsset("macos-arm64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch latest release")
}

func TestResolveLlamaServerAsset_Non200Status(t *testing.T) {
	origTransport := githubReleaseHTTPClient.Transport
	t.Cleanup(func() { githubReleaseHTTPClient.Transport = origTransport })
	githubReleaseHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     make(http.Header),
		}, nil
	})

	_, err := resolveLlamaServerAsset("macos-arm64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch latest release returned 404")
}

func TestResolveLlamaServerAsset_InvalidJSON(t *testing.T) {
	mockGithubReleaseJSON(t, "not json")

	_, err := resolveLlamaServerAsset("macos-arm64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode release")
}

// extractTarGzFile tests, mirroring the existing extractZipFile coverage.

func writeTestTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	tarGzPath := filepath.Join(dir, "test.tar.gz")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0600, Size: int64(len(content))}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0600))
	return tarGzPath
}

func TestExtractTarGzFile_ValidArchive(t *testing.T) {
	tarGzPath := writeTestTarGz(t, map[string]string{
		"build/bin/llama-server": "fake-binary-content",
	})
	destDir := filepath.Join(filepath.Dir(tarGzPath), "extracted")

	err := extractTarGzFile(tarGzPath, destDir)
	require.NoError(t, err)

	data, err := os.ReadFile(destDir)
	require.NoError(t, err)
	assert.Equal(t, "fake-binary-content", string(data))
}

func TestExtractTarGzFile_NoLlamaServerBinary(t *testing.T) {
	tarGzPath := writeTestTarGz(t, map[string]string{
		"some-other-file.txt": "not the binary",
	})

	err := extractTarGzFile(tarGzPath, filepath.Join(filepath.Dir(tarGzPath), "out"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llama-server binary not found")
}

func TestExtractTarGzFile_MissingFile(t *testing.T) {
	err := extractTarGzFile("/nonexistent/archive.tar.gz", t.TempDir())
	require.Error(t, err)
}

func TestExtractTarGzFile_InvalidGzipContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.tar.gz")
	require.NoError(t, os.WriteFile(path, []byte("not-a-gzip-file"), 0600))

	err := extractTarGzFile(path, filepath.Join(dir, "out"))
	require.Error(t, err)
}

// TestInstallLlamaServer_TarGzSuccess pins the full non-Windows install path:
// resolveLlamaServerAssetFn returns a .tar.gz asset, installLlamaServer must
// extract it with extractTarGzFile (not extractZipFile) and leave the binary
// executable.
func TestInstallLlamaServer_TarGzSuccess(t *testing.T) {
	stubLlamaServerAsset(t, "https://example.com/llama-server.tar.gz")

	tarGzPath := writeTestTarGz(t, map[string]string{
		"build/bin/llama-server": "fake-targz-binary",
	})
	archiveBytes, err := os.ReadFile(tarGzPath)
	require.NoError(t, err)

	origTransport := downloadHTTPClient.Transport
	t.Cleanup(func() { downloadHTTPClient.Transport = origTransport })
	downloadHTTPClient.Transport = roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(archiveBytes)),
			Header:     make(http.Header),
		}, nil
	})

	dir := t.TempDir()
	dest := filepath.Join(dir, ".kdeps", "bin", "llama-server")
	require.NoError(t, os.MkdirAll(filepath.Dir(dest), 0750))

	err = installLlamaServer(dest)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, "fake-targz-binary", string(data))

	info, err := os.Stat(dest)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode().Perm()&0100, "installed llama-server binary must have owner-execute permission")
}
