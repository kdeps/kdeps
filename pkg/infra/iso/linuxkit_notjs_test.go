// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build !js

package iso

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureLinuxKit_InPath(t *testing.T) {
	tmpDir := t.TempDir()
	fakeLinuxKit := filepath.Join(tmpDir, "linuxkit")
	if err := os.WriteFile(fakeLinuxKit, []byte("#!/bin/sh\necho ok"), 0755); err != nil {
		t.Fatalf("failed to create fake linuxkit: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	path, err := EnsureLinuxKit(context.Background())
	if err != nil {
		t.Fatalf("EnsureLinuxKit failed: %v", err)
	}
	if path != fakeLinuxKit {
		t.Errorf("expected %s, got %s", fakeLinuxKit, path)
	}
}

func TestEnsureLinuxKit_InCache(t *testing.T) {
	// Mock HOME to control cache location
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create cached binary
	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps", "linuxkit")
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	cachedBinary := filepath.Join(cacheDir, "linuxkit-"+linuxkitVersion)
	if err := os.WriteFile(cachedBinary, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create cached binary: %v", err)
	}

	// Ensure it's NOT in PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	path, err := EnsureLinuxKit(context.Background())
	if err != nil {
		t.Fatalf("EnsureLinuxKit failed: %v", err)
	}
	if path != cachedBinary {
		t.Errorf("expected %s, got %s", cachedBinary, path)
	}
}

func TestDownloadFile_RenameError(t *testing.T) {
	// Create a test HTTP server that serves binary content.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("binary content"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	// Make the destination path an existing directory so os.Rename fails.
	dest := filepath.Join(tmpDir, "dest")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// downloadFile writes to dest.tmp, then tries to rename to dest.
	// Since dest is a directory, the rename on Unix returns EISDIR.
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Fatal("expected error when dest is a directory, got nil")
	}
	if !strings.Contains(err.Error(), "rename") {
		t.Errorf("expected rename error, got: %v", err)
	}
}

func TestLinuxkitCacheDir(t *testing.T) {
	dir, err := linuxkitCacheDir()
	if err != nil {
		t.Fatalf("linuxkitCacheDir returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty cache dir")
	}
	// Must be under the user's home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}
	if !strings.HasPrefix(dir, home) {
		t.Errorf("cache dir %q is not under home %q", dir, home)
	}
}

func TestLinuxKitDownloadURL(t *testing.T) {
	url := LinuxKitDownloadURL()
	if url == "" {
		t.Fatal("expected non-empty download URL")
	}
	if !strings.Contains(url, linuxkitVersion) {
		t.Errorf("URL %q does not contain version %q", url, linuxkitVersion)
	}
	if !strings.Contains(url, runtime.GOOS) {
		t.Errorf("URL %q does not contain GOOS %q", url, runtime.GOOS)
	}
	if !strings.Contains(url, runtime.GOARCH) {
		t.Errorf("URL %q does not contain GOARCH %q", url, runtime.GOARCH)
	}
}

func TestDownloadFile_Success(t *testing.T) {
	content := []byte("binary content")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	if err := downloadFile(t.Context(), ts.URL, dest); err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "should-not-exist")
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

func TestDownloadFile_InvalidURL(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out")
	err := downloadFile(t.Context(), "://invalid-url", dest)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestDownloadFile_HTTPDoError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that connects to an invalid address")
	}

	// Use a short timeout so the test doesn't hang for 30s waiting on the dial timeout.
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	dest := filepath.Join(t.TempDir(), "out")
	err := downloadFile(ctx, "http://127.0.0.1:1/download", dest)
	if err == nil {
		t.Error("expected error for connection refused or timeout, got nil")
	}
}

func TestDownloadFile_CreateError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("content"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}

	// dest inside read-only directory — os.Create on tmpFile (dest+".tmp") will fail.
	dest := filepath.Join(readOnlyDir, "output")
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Error("expected error for read-only directory, got nil")
	}
}

func TestDownloadFile_CopyError(t *testing.T) {
	// Create a server that writes partial content then hijacks and closes the connection.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		_, _ = w.Write([]byte("partial"))
		flusher.Flush()

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hijacker.Hijack()
		_ = conn.Close()
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "output")
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Error("expected error for incomplete response / connection reset, got nil")
	}
}

func TestEnsureLinuxKit_DownloadErrorWithCancelledContext(t *testing.T) {
	// Use a cancelled context so downloadFile returns immediately without network access.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // context already cancelled — downloadFile fails immediately

	_, err := EnsureLinuxKit(ctx)
	if err == nil {
		t.Fatal("expected download error with cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "failed to download linuxkit") {
		t.Fatalf("expected 'failed to download linuxkit' error, got: %v", err)
	}
}

func TestEnsureLinuxKit_MkdirAllError(t *testing.T) {
	// Create a file at the cache directory path so MkdirAll fails with ENOTDIR.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	// Create parent directory for cache
	cacheParent := filepath.Join(tmpDir, ".cache", "kdeps")
	if err := os.MkdirAll(cacheParent, 0750); err != nil {
		t.Fatalf("failed to create cache parent: %v", err)
	}

	// Place a FILE where cacheDir (a subdirectory of cacheParent) would be created.
	// linuxkitCacheDir returns filepath.Join(home, ".cache", "kdeps", "linuxkit"),
	// so creating a file at that path tricks MkdirAll into failing.
	cacheDirPath := filepath.Join(cacheParent, "linuxkit")
	if err := os.WriteFile(cacheDirPath, []byte("not-a-directory"), 0644); err != nil {
		t.Fatalf("failed to create file at cacheDir path: %v", err)
	}

	_, err := EnsureLinuxKit(t.Context())
	if err == nil {
		t.Fatal("expected error from MkdirAll failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create cache directory") {
		t.Fatalf("expected 'failed to create cache directory' error, got: %v", err)
	}
}

func TestEnsureLinuxKit_DownloadError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that may attempt network access")
	}

	// Ensure linuxkit is not on PATH, and no cached binary exists.
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	// Use a short timeout so the test doesn't hang if GitHub is unreachable.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	path, err := EnsureLinuxKit(ctx)
	if err != nil {
		// Expected: download fails because GitHub is unreachable or context times out.
		if !strings.Contains(err.Error(), "failed to download linuxkit") {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		// Rare case: network available and download succeeded.
		if path == "" {
			t.Error("expected non-empty path on success")
		}
	}
}

func TestEnsureLinuxKit_CacheDirError(t *testing.T) {
	origHome := osUserHomeDir
	origPath := os.Getenv("PATH")
	t.Cleanup(func() {
		osUserHomeDir = origHome
		os.Setenv("PATH", origPath)
	})

	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home directory")
	}
	os.Setenv("PATH", "")

	_, err := EnsureLinuxKit(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get home directory")
}

func TestDownloadLinuxKit_ChmodError(t *testing.T) {
	tmpDir := t.TempDir()
	cachedBinary := filepath.Join(tmpDir, "linuxkit-test")

	origChmod := osChmod
	t.Cleanup(func() { osChmod = origChmod })
	osChmod = func(_ string, _ os.FileMode) error {
		return errors.New("chmod failed")
	}

	origDo := httpClientDo
	t.Cleanup(func() { httpClientDo = origDo })
	httpClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("binary")),
			Header:     make(http.Header),
		}, nil
	}

	_, err := downloadLinuxKit(context.Background(), tmpDir, cachedBinary)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make linuxkit executable")
}

func TestCreateRawBIOSWorkDir_HomeDirError(t *testing.T) {
	orig := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = orig })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home directory")
	}

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get home directory")
}

func TestEnsureLinuxKit_CachedBinary(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	cacheDir, err := linuxkitCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(cacheDir, 0750))

	cachedBinary := filepath.Join(cacheDir, "linuxkit-"+linuxkitVersion)
	require.NoError(t, os.WriteFile(cachedBinary, []byte("cached"), 0o700))

	path, err := EnsureLinuxKit(t.Context())
	require.NoError(t, err)
	assert.Equal(t, cachedBinary, path)
}

func TestDownloadLinuxKit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cachedBinary := filepath.Join(tmpDir, "linuxkit-"+linuxkitVersion)

	origDo := httpClientDo
	t.Cleanup(func() { httpClientDo = origDo })
	httpClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("linuxkit-binary")),
			Header:     make(http.Header),
		}, nil
	}

	path, err := downloadLinuxKit(t.Context(), tmpDir, cachedBinary)
	require.NoError(t, err)
	assert.Equal(t, cachedBinary, path)
	assert.FileExists(t, cachedBinary)
}

func TestDefaultLinuxKitRunner_ExecMock(t *testing.T) {
	origExec := execCommandContext
	t.Cleanup(func() { execCommandContext = origExec })

	runner := &DefaultLinuxKitRunner{BinaryPath: "linuxkit"}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("kernel: {}"), 0644))

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 0")
	}

	require.NoError(t, runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, ""))
	require.NoError(t, runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, "4096M"))
	require.NoError(t, runner.CacheImport(t.Context(), filepath.Join(tmpDir, "image.tar")))

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}

	err := runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit build failed")

	err = runner.CacheImport(t.Context(), filepath.Join(tmpDir, "image.tar"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit cache import failed")
}
