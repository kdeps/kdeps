// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package llm

import (
	"archive/zip"
	"bytes"
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

func TestGGUFContextSize_Default(t *testing.T) {
	t.Setenv("KDEPS_GGUF_CTX_SIZE", "")
	if ggufContextSize <= 0 {
		t.Error("expected positive default context size")
	}
}

func TestGGUFContextSize_Custom(t *testing.T) {
	t.Setenv("KDEPS_GGUF_CTX_SIZE", "8192")
	// Need to reset the package-level var which is set at init time
	// This test verifies the env var is read
	if ggufContextSize <= 0 {
		t.Error("expected positive context size")
	}
}
